package sdk

import (
	"context"
	"fmt"
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
