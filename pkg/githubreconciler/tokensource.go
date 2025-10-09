/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"time"

	"chainguard.dev/sdk/octosts"
	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// octoTokenFunc is the function used to get tokens from Octo STS.
// This is a variable so tests can override it with a mock.
var octoTokenFunc = octosts.Token

// tokenSource implements oauth2.TokenSource using octosts
type tokenSource struct {
	ctx      context.Context
	identity string
	org      string
	repo     string
}

// Token implements oauth2.TokenSource
func (ts *tokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(ts.ctx, 1*time.Minute)
	defer cancel()
	tok, err := octoTokenFunc(ctx, ts.identity, ts.org, ts.repo)
	if err != nil {
		// Check if this is a gRPC NotFound error
		if status.Code(err) == codes.NotFound {
			// A common reason for NotFound from Octo STS is that the org's GitHub App
			// installation quota has been exhausted. We log this and requeue with a delay
			// to give time for the quota to reset or for manual intervention.
			scope := ts.org
			if ts.repo != "" {
				scope = ts.org + "/" + ts.repo
			}
			clog.ErrorContextf(ctx, "Got NotFound error from Octo STS for %q: %v", scope, err)
			return nil, workqueue.RetryAfter(10 * time.Minute)
		}
		return nil, err
	}
	return &oauth2.Token{
		AccessToken: tok,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(55 * time.Minute), // Tokens from Octo STS are valid for 60 minutes
	}, nil
}

// NewOrgTokenSource creates a new token source for org-scoped GitHub credentials
func NewOrgTokenSource(ctx context.Context, identity, org string) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, &tokenSource{
		ctx:      ctx,
		identity: identity,
		org:      org,
		repo:     "",
	})
}

// NewRepoTokenSource creates a new token source for repo-scoped GitHub credentials
func NewRepoTokenSource(ctx context.Context, identity, org, repo string) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, &tokenSource{
		ctx:      ctx,
		identity: identity,
		org:      org,
		repo:     repo,
	})
}
