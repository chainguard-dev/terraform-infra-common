/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/google/go-github/v88/github"
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
	// NewClient passes no fallible options, so the only error source in
	// NewClientWithOptions (option failures such as WithEnterpriseURLs parsing
	// URLs) cannot occur here; WithTransport never fails.
	client, err := NewClientWithOptions(base)
	if err != nil {
		panic(fmt.Sprintf("sdk.NewClient: %v", err))
	}
	return client
}

// NewClientWithOptions returns a *github.Client whose HTTP transport is
// instrumented with httpmetrics.WrapTransport (as in NewClient), configured
// with any additional go-github client options. Use this to reach APIs the
// bare NewClient does not expose — for example, github.WithEnterpriseURLs to
// target a GitHub Enterprise instance.
//
// Unlike NewClient, this returns an error: some options parse URLs and can
// fail. base provides authentication as described on NewClient.
func NewClientWithOptions(base http.RoundTripper, opts ...github.ClientOptionsFunc) (*github.Client, error) {
	allOpts := append([]github.ClientOptionsFunc{github.WithTransport(httpmetrics.WrapTransport(base))}, opts...)
	client, err := github.NewClient(allOpts...)
	if err != nil {
		return nil, fmt.Errorf("creating github client: %w", err)
	}
	return client, nil
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
