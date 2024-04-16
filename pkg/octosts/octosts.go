/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package octosts

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"cloud.google.com/go/compute/metadata"
	"github.com/chainguard-dev/clog"
	"google.golang.org/api/idtoken"

	"chainguard.dev/sdk/sts"
)

const (
	OctoSTSEndpoint = "https://octo-sts.dev"
)

// Token mints a new octo sts token based on the policy for a given repo.
func Token(ctx context.Context, policyName, org, repo string) (string, error) {
	// To help enable local development, we allow the use of a GitHub token,
	// but *only when not running on GCE*.
	if tok := os.Getenv("GH_TOKEN"); tok != "" && !metadata.OnGCE() {
		clog.Warnf("using GH_TOKEN for token exchange")
		return tok, nil
	}
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" && !metadata.OnGCE() {
		clog.Warnf("using GITHUB_TOKEN for token exchange")
		return tok, nil
	}

	scope := org
	if repo != "" {
		scope = fmt.Sprintf("%s/%s", org, repo)
	}

	xchg := sts.New(
		OctoSTSEndpoint,
		policyName,
		sts.WithScope(scope),
		sts.WithIdentity(policyName),
	)

	ts, err := idtoken.NewTokenSource(ctx, "octo-sts.dev" /* aud */)
	if err != nil {
		return "", err
	}

	token, err := ts.Token()
	if err != nil {
		return "", err
	}

	res, err := xchg.Exchange(ctx, token.AccessToken)
	if err != nil {
		return "", err
	}

	return res, nil
}

// Revoke revokes the given security token.
func Revoke(ctx context.Context, tok string) error {
	req, err := http.NewRequest(http.MethodDelete, "https://api.github.com/installation/token", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Add("Authorization", "Bearer "+tok)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// The token was revoked!
	return nil
}
