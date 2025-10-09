/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTokenSource_Token_Success(t *testing.T) {
	ctx := context.Background()
	wantIdentity := fmt.Sprintf("identity-%d", rand.Int64())
	wantOrg := fmt.Sprintf("org-%d", rand.Int64())
	wantRepo := fmt.Sprintf("repo-%d", rand.Int64())
	wantToken := fmt.Sprintf("token-%d", rand.Int64())

	// Mock the octoTokenFunc
	originalFunc := octoTokenFunc
	t.Cleanup(func() { octoTokenFunc = originalFunc })

	octoTokenFunc = func(_ context.Context, identity, org, repo string) (string, error) {
		if identity != wantIdentity {
			t.Errorf("identity: got = %q, wanted = %q", identity, wantIdentity)
		}
		if org != wantOrg {
			t.Errorf("org: got = %q, wanted = %q", org, wantOrg)
		}
		if repo != wantRepo {
			t.Errorf("repo: got = %q, wanted = %q", repo, wantRepo)
		}
		return wantToken, nil
	}

	ts := &tokenSource{
		ctx:      ctx,
		identity: wantIdentity,
		org:      wantOrg,
		repo:     wantRepo,
	}

	token, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.AccessToken != wantToken {
		t.Errorf("AccessToken: got = %q, wanted = %q", token.AccessToken, wantToken)
	}

	if token.TokenType != "Bearer" {
		t.Errorf("TokenType: got = %q, wanted = %q", token.TokenType, "Bearer")
	}

	// Check expiry is approximately 55 minutes in the future
	expectedExpiry := time.Now().Add(55 * time.Minute)
	if token.Expiry.Before(expectedExpiry.Add(-1*time.Second)) || token.Expiry.After(expectedExpiry.Add(1*time.Second)) {
		t.Errorf("Expiry: got = %v, wanted approximately = %v", token.Expiry, expectedExpiry)
	}
}

func TestTokenSource_Token_NotFoundError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		org  string
		repo string
	}{{
		name: "org only",
		org:  fmt.Sprintf("org-%d", rand.Int64()),
		repo: "",
	}, {
		name: "org and repo",
		org:  fmt.Sprintf("org-%d", rand.Int64()),
		repo: fmt.Sprintf("repo-%d", rand.Int64()),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the octoTokenFunc to return NotFound
			originalFunc := octoTokenFunc
			t.Cleanup(func() { octoTokenFunc = originalFunc })

			octoTokenFunc = func(_ context.Context, _, _, _ string) (string, error) {
				return "", status.Error(codes.NotFound, "installation not found")
			}

			ts := &tokenSource{
				ctx:      ctx,
				identity: fmt.Sprintf("identity-%d", rand.Int64()),
				org:      tt.org,
				repo:     tt.repo,
			}

			token, err := ts.Token()
			if err == nil {
				t.Fatal("expected error but got none")
			}

			if token != nil {
				t.Errorf("token: got = %v, wanted = nil", token)
			}

			// Check that error is a requeue error with the correct delay
			delay, ok, isError := workqueue.GetRequeueDelay(err)
			if !ok {
				t.Errorf("error type: got non-requeue error, wanted requeue error")
			} else if delay != 10*time.Minute {
				t.Errorf("requeue duration: got = %v, wanted = %v", delay, 10*time.Minute)
			}
			if !isError {
				t.Error("expected isError = true for RetryAfter, got false")
			}
		})
	}
}

func TestTokenSource_Token_OtherError(t *testing.T) {
	ctx := context.Background()
	wantErr := fmt.Errorf("error-%d", rand.Int64())

	// Mock the octoTokenFunc to return a different error
	originalFunc := octoTokenFunc
	t.Cleanup(func() { octoTokenFunc = originalFunc })

	octoTokenFunc = func(_ context.Context, _, _, _ string) (string, error) {
		return "", wantErr
	}

	ts := &tokenSource{
		ctx:      ctx,
		identity: fmt.Sprintf("identity-%d", rand.Int64()),
		org:      fmt.Sprintf("org-%d", rand.Int64()),
		repo:     "",
	}

	token, err := ts.Token()
	if err == nil {
		t.Fatal("expected error but got none")
	}

	if token != nil {
		t.Errorf("token: got = %v, wanted = nil", token)
	}

	if !errors.Is(err, wantErr) {
		t.Errorf("error: got = %v, wanted = %v", err, wantErr)
	}
}

func TestNewOrgTokenSource(t *testing.T) {
	ctx := context.Background()
	wantIdentity := fmt.Sprintf("identity-%d", rand.Int64())
	wantOrg := fmt.Sprintf("org-%d", rand.Int64())
	wantToken := fmt.Sprintf("token-%d", rand.Int64())

	// Mock the octoTokenFunc
	originalFunc := octoTokenFunc
	t.Cleanup(func() { octoTokenFunc = originalFunc })

	var capturedIdentity, capturedOrg, capturedRepo string
	octoTokenFunc = func(_ context.Context, identity, org, repo string) (string, error) {
		capturedIdentity = identity
		capturedOrg = org
		capturedRepo = repo
		return wantToken, nil
	}

	ts := NewOrgTokenSource(ctx, wantIdentity, wantOrg)
	if ts == nil {
		t.Fatal("expected token source but got nil")
	}

	// Get a token to verify parameters
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.AccessToken != wantToken {
		t.Errorf("AccessToken: got = %q, wanted = %q", token.AccessToken, wantToken)
	}

	if capturedIdentity != wantIdentity {
		t.Errorf("identity: got = %q, wanted = %q", capturedIdentity, wantIdentity)
	}

	if capturedOrg != wantOrg {
		t.Errorf("org: got = %q, wanted = %q", capturedOrg, wantOrg)
	}

	if capturedRepo != "" {
		t.Errorf("repo: got = %q, wanted = %q", capturedRepo, "")
	}
}

func TestNewRepoTokenSource(t *testing.T) {
	ctx := context.Background()
	wantIdentity := fmt.Sprintf("identity-%d", rand.Int64())
	wantOrg := fmt.Sprintf("org-%d", rand.Int64())
	wantRepo := fmt.Sprintf("repo-%d", rand.Int64())
	wantToken := fmt.Sprintf("token-%d", rand.Int64())

	// Mock the octoTokenFunc
	originalFunc := octoTokenFunc
	t.Cleanup(func() { octoTokenFunc = originalFunc })

	var capturedIdentity, capturedOrg, capturedRepo string
	octoTokenFunc = func(_ context.Context, identity, org, repo string) (string, error) {
		capturedIdentity = identity
		capturedOrg = org
		capturedRepo = repo
		return wantToken, nil
	}

	ts := NewRepoTokenSource(ctx, wantIdentity, wantOrg, wantRepo)
	if ts == nil {
		t.Fatal("expected token source but got nil")
	}

	// Get a token to verify parameters
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.AccessToken != wantToken {
		t.Errorf("AccessToken: got = %q, wanted = %q", token.AccessToken, wantToken)
	}

	if capturedIdentity != wantIdentity {
		t.Errorf("identity: got = %q, wanted = %q", capturedIdentity, wantIdentity)
	}

	if capturedOrg != wantOrg {
		t.Errorf("org: got = %q, wanted = %q", capturedOrg, wantOrg)
	}

	if capturedRepo != wantRepo {
		t.Errorf("repo: got = %q, wanted = %q", capturedRepo, wantRepo)
	}
}

func TestTokenSource_ReuseToken(t *testing.T) {
	ctx := context.Background()

	// Mock the octoTokenFunc
	originalFunc := octoTokenFunc
	t.Cleanup(func() { octoTokenFunc = originalFunc })

	callCount := 0
	octoTokenFunc = func(_ context.Context, _, _, _ string) (string, error) {
		callCount++
		return fmt.Sprintf("token-%d", callCount), nil
	}

	ts := NewOrgTokenSource(ctx, fmt.Sprintf("identity-%d", rand.Int64()), fmt.Sprintf("org-%d", rand.Int64()))

	// First call should get token
	token1, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second call should reuse token
	token2, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be the same token (oauth2.ReuseTokenSource caches valid tokens)
	if token1.AccessToken != token2.AccessToken {
		t.Errorf("AccessToken: got = %q, wanted = %q (same token)", token2.AccessToken, token1.AccessToken)
	}

	// octoTokenFunc should only be called once due to caching
	if callCount != 1 {
		t.Errorf("call count: got = %d, wanted = 1", callCount)
	}
}
