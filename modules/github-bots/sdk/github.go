package sdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

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
	inner *github.Client
	ts    *tokenSource
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

// SetComment adds or replaces a bot comment on the given pull request.
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

// AddComment adds a new comment to the given pull request.
func (c GitHubClient) AddComment(ctx context.Context, pr *github.PullRequest, content string) error {
	if _, resp, err := c.inner.Issues.CreateComment(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, &github.IssueComment{
		Body: &content,
	}); err != nil || resp.StatusCode != 201 {
		return fmt.Errorf("creating comment: %w %v", err, resp.Status)
	}
	return nil
}

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
