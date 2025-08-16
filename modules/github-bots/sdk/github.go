package sdk

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	bufra "github.com/avvmoto/buf-readerat"
	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitHttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/snabb/httpreaderat"

	"chainguard.dev/sdk/octosts"
	"github.com/chainguard-dev/clog"
	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
)

// NewGitHubClient creates a new GitHub client, using a new token from OctoSTS,
// for the given org, repo and policy name.
//
// A new token is created for each client, and is not refreshed. It can be
// revoked with Close.
func NewGitHubClient(ctx context.Context, org, repo, policyName string, opts ...GitHubClientOption) GitHubClient {
	ts := oauth2.ReuseTokenSource(nil, &tokenSource{
		org:        org,
		repo:       repo,
		policyName: policyName,
	})

	client := GitHubClient{
		inner:   github.NewClient(oauth2.NewClient(ctx, ts)),
		ts:      ts,
		bufSize: 1024 * 1024, // 1MB buffer for requests
		org:     org,
		repo:    repo,
	}

	for _, opt := range opts {
		opt(&client)
	}

	return client
}

type installationTokenSource struct {
	ctx       context.Context
	transport *ghinstallation.Transport
}

func (ts *installationTokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.transport.Token(ts.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	expiry, _, err := ts.transport.Expiry()
	if err != nil {
		return nil, fmt.Errorf("failed to get token expiry: %w", err)
	}
	return &oauth2.Token{
		AccessToken: token,
		Expiry:      expiry,
	}, nil
}

func NewInstallationClient(ctx context.Context, org, repo string, tr *ghinstallation.Transport, opts ...GitHubClientOption) GitHubClient {
	ts := oauth2.ReuseTokenSource(nil, &installationTokenSource{
		ctx:       ctx,
		transport: tr,
	})
	client := GitHubClient{
		inner:   github.NewClient(&http.Client{Transport: tr}),
		ts:      ts,
		bufSize: 1024 * 1024, // 1MB buffer for requests
		org:     org,
		repo:    repo,
	}

	for _, opt := range opts {
		opt(&client)
	}

	return client
}

type tokenSource struct {
	org, repo, policyName, tok string
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 1*time.Minute, errors.New("get octosts token timeout"))
	defer cancel()

	clog.FromContext(ctx).Debugf("getting octosts token for %s/%s - %s", ts.org, ts.repo, ts.policyName)
	tok, err := octosts.Token(ctx, ts.policyName, ts.org, ts.repo)
	if err != nil {
		return nil, err
	}

	// If there's a previous token, attempt to revoke it.
	if ts.tok != "" {
		ctx, cancel := context.WithTimeoutCause(context.Background(), 1*time.Minute, errors.New("revoke previous token timeout"))
		defer cancel()

		if err := octosts.Revoke(ctx, ts.tok); err != nil {
			// This isn't an error, but we should log it.
			clog.FromContext(ctx).Warnf("failed to revoke token: %v", err)
		}
	}

	ts.tok = tok
	return &oauth2.Token{
		AccessToken: tok,
		// We don't actually know when it will expire, but it's probably in 1
		// hour, and we want to refresh it before it expires.
		Expiry: time.Now().Add(45 * time.Minute),
	}, nil
}

// GitHubClientOption configures the client, these are ran after the default setup.
type GitHubClientOption func(*GitHubClient)

func WithSecondaryRateLimitWaiter() GitHubClientOption {
	return func(c *GitHubClient) {
		c.inner.Client().Transport = NewSecondaryRateLimitWaiterClient(
			&oauth2.Transport{
				Base: c.inner.Client().Transport,
			},
		).Transport
	}
}

func WithBufferSize(bufSize int) GitHubClientOption {
	return func(c *GitHubClient) {
		c.bufSize = bufSize
	}
}

// WithClient sets the inner GitHub client to the given client
// useful for testing
func WithClient(client *github.Client) GitHubClientOption {
	return func(c *GitHubClient) {
		c.inner = client
	}
}

type GitHubClient struct {
	inner     *github.Client
	ts        oauth2.TokenSource
	org, repo string
	bufSize   int
}

func (c GitHubClient) Client() *github.Client { return c.inner }

func (c GitHubClient) Close(ctx context.Context) error {
	log := clog.FromContext(ctx)

	// TODO: We shouldn't get a token here if it's the first time, just to revoke it.
	tok, err := c.ts.Token()
	if err != nil {
		// Callers might just `defer c.Close()` so we log the error here too
		log.Warnf("failed to get token for revocation: %v", err)
		return fmt.Errorf("getting token for revocation: %w", err)
	}

	// We don't want to cancel the context, as we want to revoke the token even if the context is done.
	// Re-wrap the current context with a 1 minute timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeoutCause(context.WithoutCancel(ctx), 1*time.Minute, errors.New("revoking token timeout"))
	defer cancel()

	if err := octosts.Revoke(ctx, tok.AccessToken); err != nil {
		// Callers might just `defer c.Close()` so we log the error here too
		clog.FromContext(ctx).Errorf("failed to revoke token: %v", err)
		return fmt.Errorf("revoking token: %w", err)
	}

	return nil
}

// GitAuth returns a go-git transport.AuthMethod using the GitHubClient's
// credentials. This is useful for authentication in go-git operations like
// cloning and fetching repositories.
func (c GitHubClient) GitAuth() (transport.AuthMethod, error) {
	tok, err := c.ts.Token()
	if err != nil {
		return nil, fmt.Errorf("getting token from client's token source: %w", err)
	}

	auth := &gitHttp.BasicAuth{
		// https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation#about-authentication-as-a-github-app-installation
		Username: "x-access-token",
		Password: tok.AccessToken,
	}

	return auth, nil
}

// RepoURL returns the HTTPS git URL of the GitHubClient's configured
// repository.
func (c GitHubClient) RepoURL() (string, error) {
	if c.org == "" || c.repo == "" {
		return "", errors.New("GitHubClient is not configured with both an org and repo")
	}

	return fmt.Sprintf("https://github.com/%s/%s.git", c.org, c.repo), nil
}

// checkRateLimiting checks for github API rate limiting. It attempts to use
// the returned Retry-After header to calculate the delay returned from the API.
//
// Modified from https://github.com/wolfi-dev/wolfictl/blob/main/pkg/gh/github.go
func checkRateLimiting(_ context.Context, githubErr error) (bool, time.Duration) {
	// Default delay is 30 seconds
	delay := time.Duration(30 * int(time.Second))
	isRateLimited := false

	// If GitHub returned an error of type RateLimitError, we can attempt to
	// compute the next time to try the request again by reading its rate limit information
	var rateLimitError *github.RateLimitError
	if errors.As(githubErr, &rateLimitError) {
		isRateLimited = true
		delay = time.Until(*rateLimitError.Rate.Reset.GetTime())
	}

	// If GitHub returned a Retry-After header, use its value, otherwise use the default
	var abuseRateLimitError *github.AbuseRateLimitError
	if errors.As(githubErr, &abuseRateLimitError) {
		isRateLimited = true
		delay = abuseRateLimitError.GetRetryAfter()
	}
	return isRateLimited, delay
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
	if err := validateResponse(ctx, err, resp, "add label to pull request"); err != nil {
		return err
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
	if err := validateResponse(ctx, err, resp, "remove label from pull request"); err != nil {
		return err
	}
	return nil
}

// FindCommentByMarker finds a comment containing the specified marker string.
// It handles pagination automatically and returns the first matching comment.
func (c GitHubClient) FindCommentByMarker(ctx context.Context, owner, repo string, issueNumber int, marker string) (*github.IssueComment, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := c.inner.Issues.ListComments(ctx, owner, repo, issueNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("listing comments: %w", err)
		}

		for _, comment := range comments {
			if comment.Body != nil && strings.Contains(*comment.Body, marker) {
				return comment, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil, nil
}

// UpdateOrCreateComment updates an existing comment with the marker or creates a new one.
// The marker should be unique to identify the comment (e.g., "<!-- bot:name -->").
func (c GitHubClient) UpdateOrCreateComment(ctx context.Context, owner, repo string, issueNumber int, marker, content string) error {
	// Find existing comment
	existingComment, err := c.FindCommentByMarker(ctx, owner, repo, issueNumber, marker)
	if err != nil {
		return fmt.Errorf("finding existing comment: %w", err)
	}

	// Ensure marker is in content
	if !strings.Contains(content, marker) {
		content = marker + "\n\n" + content
	}

	if existingComment != nil {
		// Update existing comment
		_, resp, err := c.inner.Issues.EditComment(ctx, owner, repo, *existingComment.ID, &github.IssueComment{
			Body: &content,
		})
		if err := validateResponse(ctx, err, resp, "edit comment"); err != nil {
			return err
		}
	} else {
		// Create new comment
		_, resp, err := c.inner.Issues.CreateComment(ctx, owner, repo, issueNumber, &github.IssueComment{
			Body: &content,
		})
		if err != nil || resp.StatusCode != 201 {
			return validateResponse(ctx, err, resp, "create comment")
		}
	}

	return nil
}

// ListCommentsWithPagination calls the provided function for each comment on an issue/PR.
// The function should return true to stop iteration early.
func (c GitHubClient) ListCommentsWithPagination(ctx context.Context, owner, repo string, issueNumber int, f func(*github.IssueComment) (bool, error)) error {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		comments, resp, err := c.inner.Issues.ListComments(ctx, owner, repo, issueNumber, opts)
		if err != nil {
			return fmt.Errorf("listing comments: %w", err)
		}

		for _, comment := range comments {
			stop, err := f(comment)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil
}

// SetComment adds or replaces a bot comment on the given pull request.
func (c GitHubClient) SetComment(ctx context.Context, pr *github.PullRequest, botName, content string) error {
	marker := fmt.Sprintf("<!-- bot:%s -->", botName)
	content = fmt.Sprintf("%s\n\n%s", marker, content)
	return c.UpdateOrCreateComment(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, marker, content)
}

// AddComment adds a new comment to the given pull request.
func (c GitHubClient) AddComment(ctx context.Context, pr *github.PullRequest, botName, content string) error {
	content = fmt.Sprintf("<!-- bot:%s -->\n\n%s", botName, content)
	if _, resp, err := c.inner.Issues.CreateComment(ctx, *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name, *pr.Number, &github.IssueComment{
		Body: &content,
	}); err != nil || resp.StatusCode != 201 {
		return validateResponse(ctx, err, resp, "creating comment")
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
			return nil, errors.New("logs not found or expired")
		}
		return nil, fmt.Errorf("failed to fetch logs, status %d: %s", logsResp.StatusCode, string(body))
	}

	return body, nil
}

// FetchWorkflowRunLogs returns a Reader for the logs of the given WorkflowRun
func (c GitHubClient) FetchWorkflowRunLogs(ctx context.Context, wr *github.WorkflowRun, store httpreaderat.Store) (*zip.Reader, error) {
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

	htdrd, err := httpreaderat.New(nil, req, store)
	if err != nil {
		return nil, err
	}

	bhtrdr := bufra.NewBufReaderAt(htdrd, c.bufSize)

	zr, err := zip.NewReader(bhtrdr, htdrd.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}
	return zr, nil
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

	return 0, errors.New("no matching pull request found")
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

	var zr *zip.Reader
	if err := c.ListArtifactsFunc(ctx, wr, &github.ListOptions{PerPage: 30}, func(a *github.Artifact) (bool, error) {
		if *a.Name != name {
			return false, nil
		}

		aid := a.GetID()
		url, ghresp, err := c.inner.Actions.DownloadArtifact(ctx, owner, repo, aid, 10)
		if err != nil {
			return false, fmt.Errorf("failed to download artifact (%s) [%d]: %w", name, aid, err)
		}

		if ghresp.StatusCode != http.StatusFound {
			return false, fmt.Errorf("failed to find artifact (%s) [%d]: %s", name, aid, ghresp.Status)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
		if err != nil {
			return false, err
		}

		htdrd, err := httpreaderat.New(nil, req, nil)
		if err != nil {
			return false, err
		}

		bhtrdr := bufra.NewBufReaderAt(htdrd, c.bufSize)

		r, err := zip.NewReader(bhtrdr, htdrd.Size())
		if err != nil {
			return false, fmt.Errorf("failed to create zip reader: %w", err)
		}
		zr = r
		return true, nil
	}); err != nil {
		return nil, err
	}

	if zr == nil {
		return nil, fmt.Errorf("artifact %s for workflow_run %d not found", name, *wr.ID)
	}

	return zr, nil
}

// ListArtifactsFunc executes a paginated list of all artifacts for a given
// workflow run and executes the provided function on each of the artifacts.
// The provided function should return a boolean to indicate whether the list
// operation can stop making API calls.
func (c GitHubClient) ListArtifactsFunc(ctx context.Context, wr *github.WorkflowRun, opt *github.ListOptions, f func(artifact *github.Artifact) (bool, error)) error {
	if opt == nil {
		opt = &github.ListOptions{}
	}
	owner, repo := *wr.Repository.Owner.Login, *wr.Repository.Name
	for {
		artifacts, resp, err := c.inner.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, *wr.ID, opt)
		if err := validateResponse(ctx, err, resp, "list workflow artifacts"); err != nil {
			return err
		}
		for _, artifact := range artifacts.Artifacts {
			stop, err := f(artifact)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil
}

func validateResponse(ctx context.Context, err error, resp *github.Response, action string) error {
	// resp may be nil if err is nonempty. However, err may contain a rate limit
	// error so we have to inspect for rate limiting if resp is non-nil
	if resp != nil {
		githubErr := github.CheckResponse(resp.Response)
		if githubErr != nil {
			rateLimited, delay := checkRateLimiting(ctx, githubErr)
			// if we were not rate limited, return err
			if !rateLimited {
				return err
			}
			// For now we don't handle rate limiting, just log that we got rate
			// limited and what the delay from github is
			return fmt.Errorf("hit rate limiting: delay returned from GitHub %v", delay.Seconds())
		}
	}
	if err != nil {
		if resp != nil {
			return fmt.Errorf("failed to %s: %w %v", action, err, resp.Status)
		}
		return fmt.Errorf("failed to %s: %w", action, err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to %s: %v", action, resp.Status)
	}
	return nil
}

// CloneOpts contains options for cloning a repository.
type CloneOpts struct {
	// Shallow indicates whether to perform a shallow clone (depth 1).
	Shallow bool
}

// CloneRepo clones the repository into a destination directory, and checks out a ref.
//
// ref should be "refs/heads/<branch>" or "refs/tags/<tag>" or "refs/pull/<pr>/merge" or a commit SHA.
// destDir is the directory to clone the repository into. It will be created if it doesn't exist.
// if opts is nil, a full clone will be performed.
//
// It returns the git.Repository object for the cloned repository.
func (c GitHubClient) CloneRepo(ctx context.Context, ref, destDir string, opts *CloneOpts) (*git.Repository, error) {
	log := clog.FromContext(ctx)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	repo, err := c.RepoURL()
	if err != nil {
		return nil, fmt.Errorf("getting repository URL: %w", err)
	}
	auth, err := c.GitAuth()
	if err != nil {
		return nil, fmt.Errorf("retrieving GitHub client's auth: %w", err)
	}

	// git clone <repo>
	// git fetch origin <ref>:<ref>
	// git checkout <ref>
	r, err := git.PlainCloneContext(ctx, destDir, false, &git.CloneOptions{
		URL:  repo,
		Auth: auth,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	depth := 0
	if opts != nil && opts.Shallow {
		depth = 1
	}
	if err := r.FetchContext(ctx, &git.FetchOptions{
		RemoteURL: repo,
		RefSpecs:  []config.RefSpec{config.RefSpec(fmt.Sprintf("+%s:%s", ref, ref))},
		Depth:     depth,
		Tags:      git.NoTags,
		Auth:      auth,
	}); errors.Is(err, git.NoErrAlreadyUpToDate) {
		log.Info("local repository already up to date")
	} else if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}
	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}
	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName(ref),
		Force:  true, // There should not be any local changes, but just in case.
	}); err != nil {
		status, serr := wt.Status()
		if serr != nil {
			return nil, fmt.Errorf("failed to get worktree status after failed checkout: %w", errors.Join(serr, err))
		}
		log.With("status", status).Error("failed checkout")
		return nil, fmt.Errorf("failed to checkout ref %s: %w", ref, err)
	}
	return r, nil
}

// GetRelease fetches the release by tag
func (c GitHubClient) GetRelease(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, error) {
	release, resp, err := c.inner.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get release by tag %s: %w", tag, err)
	}
	if err := validateResponse(ctx, err, resp, fmt.Sprintf("get release by tag %s", tag)); err != nil {
		return nil, err
	}
	return release, nil
}

// GetFileContent fetches the content of a file at a given ref
func (c GitHubClient) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	opts := &github.RepositoryContentGetOptions{
		Ref: ref,
	}

	fileContent, _, resp, err := c.inner.Repositories.GetContents(
		ctx,
		owner,
		repo,
		path,
		opts,
	)

	if err := validateResponse(ctx, err, resp, fmt.Sprintf("get file contents for %s at ref %s", path, ref)); err != nil {
		return "", err
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode content: %w", err)
	}

	return content, nil
}

// SearchContentInFilename searches for a text in a filename in a specific repository
func (c GitHubClient) SearchContentInFilename(ctx context.Context, owner, repo, path, content string, opt *github.ListOptions) (*github.CodeSearchResult, error) {
	if opt == nil {
		opt = &github.ListOptions{}
	}
	query := fmt.Sprintf("%s in:file filename:%s repo:%s/%s", content, path, owner, repo)
	result, resp, err := c.inner.Search.Code(
		ctx,
		query,
		&github.SearchOptions{
			ListOptions: *opt,
		},
	)

	if err := validateResponse(ctx, err, resp, fmt.Sprintf("search content %s in repository", content)); err != nil {
		return &github.CodeSearchResult{}, err
	}

	return result, nil
}

// SearchFilenameInRepository searches for a filename in a specific repository
func (c GitHubClient) SearchFilenameInRepository(ctx context.Context, owner, repo, path string, opt *github.ListOptions) (*github.CodeSearchResult, error) {
	if opt == nil {
		opt = &github.ListOptions{}
	}
	query := fmt.Sprintf("filename:%s repo:%s/%s", path, owner, repo)
	result, resp, err := c.inner.Search.Code(
		ctx,
		query,
		&github.SearchOptions{
			ListOptions: *opt,
		},
	)

	if err := validateResponse(ctx, err, resp, fmt.Sprintf("search filename %s in repository", path)); err != nil {
		return &github.CodeSearchResult{}, err
	}

	return result, nil
}

// ListFiles lists the files in a directory at a given ref
func (c GitHubClient) ListFiles(ctx context.Context, owner, repo, path, ref string) ([]*github.RepositoryContent, error) {
	opts := &github.RepositoryContentGetOptions{Ref: ref}

	_, dirContents, resp, err := c.inner.Repositories.GetContents(
		ctx, owner, repo, path, opts,
	)
	if err := validateResponse(ctx, err, resp, fmt.Sprintf("list file contents for %s at ref %s", path, ref)); err != nil {
		return nil, err
	}

	return dirContents, nil
}
