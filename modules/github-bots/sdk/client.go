/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	"net/http"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/google/go-github/v84/github"
	"golang.org/x/oauth2"
)

// NewClient returns a *github.Client whose HTTP transport is instrumented with
// httpmetrics.WrapTransport. base provides authentication: typically the
// Transport from oauth2.NewClient(ctx, ts), or a *ghinstallation.Transport for
// the GitHub App installation flow.
//
// This is the low-level primitive used by NewGitHubClient and
// NewInstallationClient; it's also the supported entry point for callers that
// want a bare *github.Client (for example, the DAF githubreconciler's
// ClientCache) without the lifecycle helpers attached to GitHubClient.
func NewClient(base http.RoundTripper) *github.Client {
	return github.NewClient(&http.Client{
		Transport: httpmetrics.WrapTransport(base),
	})
}

// NewClientWithToken returns a *github.Client authenticated with a static
// access token (typically a personal access token from $GITHUB_TOKEN or the
// gh CLI). The transport is instrumented via NewClient.
//
// Use this for one-off CLI tools and local utilities where the caller already
// has a raw token in hand. Production bots should prefer NewGitHubClient (Octo
// STS) or NewInstallationClient (GitHub App) instead.
func NewClientWithToken(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return NewClient(oauth2.NewClient(ctx, ts).Transport)
}
