package sdk

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	bufra "github.com/avvmoto/buf-readerat"
	"github.com/snabb/httpreaderat"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/octosts"
	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

// NewGitHubClient creates a new GitHub client, using a new token from OctoSTS,
// for the given org, repo and policy name.
//
// A new token is created for each client, and is not refreshed. It can be
// revoked with Close.
func NewGitHubClient(ctx context.Context, org, repo, policyName string) GitHubClient {
	ts := &tokenSource{
		org:        org,
		repo:       repo,
		policyName: policyName,
	}
	return GitHubClient{
		inner: github.NewClient(oauth2.NewClient(ctx, ts)),
		ts:    ts,
		// TODO: Make this configurable?
		bufSize: 1024 * 1024, // 1MB buffer for requests
	}
}

type tokenSource struct {
	org, repo, policyName string
	once                  sync.Once
	tok                   *oauth2.Token
	err                   error
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	// The token is only fetched once, and is cached for future calls.
	// It's not refreshed, and will expire eventually.
	ts.once.Do(func() {
		ctx := context.Background()
		clog.FromContext(ctx).Debugf("getting octosts token for %s/%s - %s", ts.org, ts.repo, ts.policyName)
		otok, err := octosts.Token(ctx, ts.policyName, ts.org, ts.repo)
		ts.tok, ts.err = &oauth2.Token{AccessToken: otok}, err
	})
	return ts.tok, ts.err
}

type GitHubClient struct {
	inner   *github.Client
	ts      *tokenSource
	bufSize int
}

func (c GitHubClient) Client() *github.Client { return c.inner }

func (c GitHubClient) Close(ctx context.Context) error {
	if c.ts.tok == nil {
		return nil // If there's no token, there's nothing to revoke.
	}

	// We don't want to cancel the context, as we want to revoke the token even if the context is done.
	ctx = context.WithoutCancel(ctx)

	if err := octosts.Revoke(ctx, c.ts.tok.AccessToken); err != nil {
		// Callers might just `defer c.Close()` so we log the error here too
		clog.FromContext(ctx).Errorf("failed to revoke token: %v", err)
		return fmt.Errorf("revoking token: %w", err)
	}

	return nil
}

func (c GitHubClient) AddLabel(ctx context.Context, pr *github.PullRequest, label string) error {
	log := clog.FromContext(ctx)

	hasLabel := slices.ContainsFunc(pr.Labels, func(l *github.Label) bool { return *l.Name == label })
	if hasLabel {
		log.Debugf("PR %d has label %v, nothing to do", *pr.Number, label)
		return nil
	}

	log.Infof("Adding label %q to PR %d", label, *pr.Number)
	_, resp, err := c.inner.Issues.AddLabelsToIssue(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, []string{label})
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("failed to add label to pull request: %w %v", err, resp.Status)
	}
	return nil
}

func (c GitHubClient) RemoveLabel(ctx context.Context, pr *github.PullRequest, label string) error {
	log := clog.FromContext(ctx)

	hasLabel := slices.ContainsFunc(pr.Labels, func(l *github.Label) bool { return *l.Name == label })
	if !hasLabel {
		log.Debugf("PR %d doesn't have label %v, nothing to do", *pr.Number, label)
		return nil
	}

	log.Infof("Removing label %q from PR %d", label, *pr.Number)
	resp, err := c.inner.Issues.RemoveLabelForIssue(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, label)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("failed to add label to pull request: %w %v", err, resp.Status)
	}
	return nil
}

func (c GitHubClient) SetComment(ctx context.Context, pr *github.PullRequest, botName, content string) error {
	cs, _, err := c.inner.Issues.ListComments(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, nil)
	if err != nil {
		return fmt.Errorf("listing comments: %w", err)
	}
	content = fmt.Sprintf("<!-- bot:%s -->\n\n%s", botName, content)

	for _, com := range cs {
		if strings.Contains(*com.Body, fmt.Sprintf("<!-- bot:%s -->", botName)) {
			if _, resp, err := c.inner.Issues.EditComment(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *com.ID, &github.IssueComment{
				Body: &content,
			}); err != nil || resp.StatusCode != 200 {
				return fmt.Errorf("editing comment: %w %v", err, resp.Status)
			}
			return nil
		}
	}
	if _, resp, err := c.inner.Issues.CreateComment(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, &github.IssueComment{
		Body: &content,
	}); err != nil || resp.StatusCode != 201 {
		return fmt.Errorf("creating comment: %w %v", err, resp.Status)
	}
	return nil
}

// Deprecated: use FetchWorkflowRunLogs instead.
func (c GitHubClient) GetWorkflowRunLogs(ctx context.Context, wre github.WorkflowRunEvent) ([]byte, error) {
	logURL, resp, err := c.inner.Actions.GetWorkflowRunLogs(ctx, *wre.Repo.Owner.Login, *wre.Repo.Name, *wre.WorkflowRun.ID, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate log retrieval: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected status code when getting logs: %s", resp.Status)
	}

	logsResp, err := http.Get(logURL.String())
	if err != nil {
		return nil, fmt.Errorf("error fetching logs from URL: %w", err)
	}
	defer logsResp.Body.Close()

	body, err := io.ReadAll(logsResp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading log response body: %w", err)
	}

	if logsResp.StatusCode != http.StatusOK {
		if logsResp.StatusCode == http.StatusNotFound || logsResp.StatusCode == http.StatusGone {
			return nil, fmt.Errorf("logs not found or expired")
		}
		return nil, fmt.Errorf("failed to fetch logs, status %d: %s", logsResp.StatusCode, string(body))
	}

	return body, nil
}

// FetchWorkflowRunLogs returns a Reader for the logs of the given WorkflowRun
func (c GitHubClient) FetchWorkflowRunLogs(ctx context.Context, wr *github.WorkflowRun) (io.ReaderAt, error) {
	url, ghresp, err := c.inner.Actions.GetWorkflowRunLogs(ctx, *wr.Repository.Owner.Login, *wr.Repository.Name, *wr.ID, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate log retrieval: %w", err)
	}
	defer ghresp.Body.Close()

	if ghresp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("failed to find log artifact (%d) for workflow [%s]: %s", *wr.ID, *wr.Name, ghresp.Status)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	htdrd, err := httpreaderat.New(nil, req, nil)
	if err != nil {
		return nil, err
	}

	return bufra.NewBufReaderAt(htdrd, c.bufSize), nil
}

func (c GitHubClient) GetWorkloadRunPullRequestNumber(ctx context.Context, wre github.WorkflowRunEvent) (int, error) {
	opts := &github.PullRequestListOptions{
		State:       "open",
		Head:        fmt.Sprintf("%s:%s", *wre.Repo.Owner.Login, *wre.WorkflowRun.HeadBranch), // Filtering by branch name
		ListOptions: github.ListOptions{PerPage: 10},
	}
	// Iterate through all pages of the results
	for {
		pulls, resp, err := c.inner.PullRequests.List(ctx, *wre.Repo.Owner.Login, *wre.Repo.Name, opts)
		if err != nil {
			return 0, fmt.Errorf("failed to list pull requests: %w", err)
		}

		// Check each pull request to see if the commit SHA matches
		for _, pr := range pulls {
			if *pr.Head.SHA == *wre.WorkflowRun.HeadSHA {
				return *pr.Number, nil
			}
		}

		// Check if there is another page of results
		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage // Update to fetch the next page
	}

	return 0, fmt.Errorf("no matching pull request found")
}

// Deprecated: Use FetchWorkflowRunArtifact instead.
func (c GitHubClient) GetWorkflowRunArtifact(ctx context.Context, wr *github.WorkflowRun, name string) (*zip.Reader, error) {
	owner, repo := *wr.Repository.Owner.Login, *wr.Repository.Name

	artifacts, _, err := c.inner.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, *wr.ID, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow run [%d] artifacts: %w", *wr.ID, err)
	}

	var zr *zip.Reader
	for _, a := range artifacts.Artifacts {
		if *a.Name != name {
			continue
		}

		aid := a.GetID()
		url, ghresp, err := c.inner.Actions.DownloadArtifact(ctx, owner, repo, aid, 10)
		if err != nil {
			return nil, fmt.Errorf("failed to download artifact (%s) [%d]: %w", name, aid, err)
		}

		if ghresp.StatusCode != http.StatusFound {
			return nil, fmt.Errorf("failed to find artifact (%s) [%d]: %s", name, aid, ghresp.Status)
		}

		client := &http.Client{}

		resp, err := client.Get(url.String())
		if err != nil {
			return nil, fmt.Errorf("could not download artifact: %w", err)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read artifact: %w", err)
		}

		buf := bytes.NewReader(data)

		r, err := zip.NewReader(buf, resp.ContentLength)
		if err != nil {
			return nil, fmt.Errorf("failed to create zip reader: %w", err)
		}
		zr = r
	}

	if zr == nil {
		return nil, fmt.Errorf("artifact %s for workflow_run %d not found", name, *wr.ID)
	}

	return zr, nil
}

// FetchWorkflowRunArtifact returns a zip reader for the artifact with `name` from the given WorkflowRun.
func (c GitHubClient) FetchWorkflowRunArtifact(ctx context.Context, wr *github.WorkflowRun, name string) (*zip.Reader, error) {
	owner, repo := *wr.Repository.Owner.Login, *wr.Repository.Name

	artifacts, _, err := c.inner.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, *wr.ID, &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow run [%d] artifacts: %w", *wr.ID, err)
	}

	var zr *zip.Reader
	for _, a := range artifacts.Artifacts {
		if *a.Name != name {
			continue
		}

		aid := a.GetID()
		url, ghresp, err := c.inner.Actions.DownloadArtifact(ctx, owner, repo, aid, 10)
		if err != nil {
			return nil, fmt.Errorf("failed to download artifact (%s) [%d]: %w", name, aid, err)
		}

		if ghresp.StatusCode != http.StatusFound {
			return nil, fmt.Errorf("failed to find artifact (%s) [%d]: %s", name, aid, ghresp.Status)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
		if err != nil {
			return nil, err
		}

		htdrd, err := httpreaderat.New(nil, req, nil)
		if err != nil {
			return nil, err
		}

		bhtrdr := bufra.NewBufReaderAt(htdrd, c.bufSize)

		r, err := zip.NewReader(bhtrdr, htdrd.Size())
		if err != nil {
			return nil, fmt.Errorf("failed to create zip reader: %w", err)
		}
		zr = r
	}

	if zr == nil {
		return nil, fmt.Errorf("artifact %s for workflow_run %d not found", name, *wr.ID)
	}

	return zr, nil
}
