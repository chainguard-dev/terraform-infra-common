/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	"time"

	"chainguard.dev/sdk/octosts"
	"golang.org/x/oauth2"
)

// OctoTokenFunc is the function used to mint Octo STS tokens. It is exposed as
// a package-level variable so tests can override it without going through the
// network. Production code should not reassign this.
var OctoTokenFunc = octosts.Token

// repoTokenSource is the inner oauth2.TokenSource implementation behind
// NewRepoTokenSource and NewOrgTokenSource. It does not revoke previously
// issued tokens on refresh; callers that need revoke semantics should layer
// that concern on top (see GitHubClient.Close in github.go).
type repoTokenSource struct {
	ctx      context.Context
	identity string
	org      string
	repo     string
}

func (ts *repoTokenSource) Token() (*oauth2.Token, error) {
	// Cap the Octo STS request at one minute. We deliberately derive from
	// ts.ctx so that token refreshes use a stable context even when individual
	// API request contexts are cancelled.
	ctx, cancel := context.WithTimeout(ts.ctx, 1*time.Minute)
	defer cancel()
	tok, err := OctoTokenFunc(ctx, ts.identity, ts.org, ts.repo)
	if err != nil {
		return nil, err
	}
	// Octo STS issues tokens valid for 60 minutes. Refresh at the 55-minute
	// mark to leave a small safety margin.
	return &oauth2.Token{
		AccessToken: tok,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(55 * time.Minute),
	}, nil
}

// NewRepoTokenSource returns an oauth2.TokenSource that mints repo-scoped
// tokens from Octo STS for the given (org, repo) using identity as the policy
// name. The returned source caches valid tokens via oauth2.ReuseTokenSource.
//
// The supplied ctx is used as the parent of each token-refresh request, so it
// should be long-lived: passing a per-request context risks "context
// cancelled" errors on later refreshes.
func NewRepoTokenSource(ctx context.Context, identity, org, repo string) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, &repoTokenSource{
		ctx:      ctx,
		identity: identity,
		org:      org,
		repo:     repo,
	})
}

// NewOrgTokenSource returns an oauth2.TokenSource that mints org-scoped tokens
// from Octo STS for the given org using identity as the policy name. The
// returned source caches valid tokens via oauth2.ReuseTokenSource.
//
// The supplied ctx is used as the parent of each token-refresh request, so it
// should be long-lived: passing a per-request context risks "context
// cancelled" errors on later refreshes.
func NewOrgTokenSource(ctx context.Context, identity, org string) oauth2.TokenSource {
	return oauth2.ReuseTokenSource(nil, &repoTokenSource{
		ctx:      ctx,
		identity: identity,
		org:      org,
	})
}
