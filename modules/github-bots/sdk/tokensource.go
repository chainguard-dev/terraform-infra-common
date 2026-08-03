/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"

	"chainguard.dev/sdk/octosts"
	"github.com/chainguard-dev/clog"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OctoTokenFunc is the function used to mint Octo STS tokens. It is exposed as
// a package-level variable so tests can override it without going through the
// network. Production code should not reassign this.
var OctoTokenFunc = octosts.Token

// NewRepoTokenSource returns an oauth2.TokenSource that mints repo-scoped
// tokens from Octo STS for the given (org, repo) using identity as the policy
// name. The returned source caches valid tokens via oauth2.ReuseTokenSource.
//
// The supplied ctx is used as the parent of each token-refresh request, so it
// should be long-lived: passing a per-request context risks "context
// cancelled" errors on later refreshes.
//
// Deprecated: use octosts.NewTokenSourceFromValues instead for proper error handling.
func NewRepoTokenSource(ctx context.Context, identity, org, repo string) oauth2.TokenSource {
	ts, err := octosts.NewTokenSourceFromValues(ctx, identity, org, repo)
	if err != nil {
		clog.WarnContextf(ctx, "failed to create Octo STS token source for %s/%s: %v", org, repo, err)
		return &errorTokenSource{}
	}
	return ts
}

// NewOrgTokenSource returns an oauth2.TokenSource that mints org-scoped tokens
// from Octo STS for the given org using identity as the policy name. The
// returned source caches valid tokens via oauth2.ReuseTokenSource.
//
// The supplied ctx is used as the parent of each token-refresh request, so it
// should be long-lived: passing a per-request context risks "context
// cancelled" errors on later refreshes.
//
// Deprecated: use octosts.NewTokenSourceFromValues instead for proper error handling.
func NewOrgTokenSource(ctx context.Context, identity, org string) oauth2.TokenSource {
	ts, err := octosts.NewTokenSourceFromValues(ctx, identity, org, "")
	if err != nil {
		clog.WarnContextf(ctx, "failed to create Octo STS token source for %s: %v", org, err)
		return &errorTokenSource{}
	}
	return ts
}

// errorTokenSource is a token source that always returns an error. It is used
// when the Octo STS token source cannot be constructed, so that callers can
// still use the returned oauth2.TokenSource without panicking.
type errorTokenSource struct{}

func (e *errorTokenSource) Token() (*oauth2.Token, error) {
	return nil, status.Errorf(codes.Unavailable, "token source unavailable")
}
