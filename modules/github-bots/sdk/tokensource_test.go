/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
)

func TestRepoTokenSource_Token_Success(t *testing.T) {
	ctx := t.Context()
	wantIdentity := fmt.Sprintf("identity-%d", rand.Int64())
	wantOrg := fmt.Sprintf("org-%d", rand.Int64())
	wantRepo := fmt.Sprintf("repo-%d", rand.Int64())
	wantToken := fmt.Sprintf("token-%d", rand.Int64())

	originalFunc := OctoTokenFunc
	t.Cleanup(func() { OctoTokenFunc = originalFunc })

	OctoTokenFunc = func(_ context.Context, identity, org, repo string) (string, error) {
		if identity != wantIdentity {
			t.Errorf("identity: got = %q, want = %q", identity, wantIdentity)
		}
		if org != wantOrg {
			t.Errorf("org: got = %q, want = %q", org, wantOrg)
		}
		if repo != wantRepo {
			t.Errorf("repo: got = %q, want = %q", repo, wantRepo)
		}
		return wantToken, nil
	}

	ts := NewRepoTokenSource(ctx, wantIdentity, wantOrg, wantRepo)
	token, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token.AccessToken != wantToken {
		t.Errorf("AccessToken: got = %q, want = %q", token.AccessToken, wantToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("TokenType: got = %q, want = %q", token.TokenType, "Bearer")
	}
	// Expiry should be ~55 minutes out, matching the documented refresh margin.
	expectedExpiry := time.Now().Add(55 * time.Minute)
	if token.Expiry.Before(expectedExpiry.Add(-1*time.Second)) || token.Expiry.After(expectedExpiry.Add(1*time.Second)) {
		t.Errorf("Expiry: got = %v, want approximately = %v", token.Expiry, expectedExpiry)
	}
}

func TestOrgTokenSource_Token_PassesEmptyRepo(t *testing.T) {
	ctx := t.Context()
	wantIdentity := fmt.Sprintf("identity-%d", rand.Int64())
	wantOrg := fmt.Sprintf("org-%d", rand.Int64())

	originalFunc := OctoTokenFunc
	t.Cleanup(func() { OctoTokenFunc = originalFunc })

	var gotRepo string
	OctoTokenFunc = func(_ context.Context, _, _, repo string) (string, error) {
		gotRepo = repo
		return "tok", nil
	}

	ts := NewOrgTokenSource(ctx, wantIdentity, wantOrg)
	if _, err := ts.Token(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRepo != "" {
		t.Errorf("repo: got = %q, want empty string", gotRepo)
	}
}

func TestRepoTokenSource_Token_ErrorPropagates(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("octosts down")

	originalFunc := OctoTokenFunc
	t.Cleanup(func() { OctoTokenFunc = originalFunc })

	OctoTokenFunc = func(_ context.Context, _, _, _ string) (string, error) {
		return "", wantErr
	}

	ts := NewRepoTokenSource(ctx, "id", "org", "repo")
	tok, err := ts.Token()
	if err == nil {
		t.Fatal("expected error but got none")
	}
	if tok != nil {
		t.Errorf("token: got = %v, want nil", tok)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error: got = %v, want = %v", err, wantErr)
	}
}

func TestRepoTokenSource_ReuseToken(t *testing.T) {
	ctx := t.Context()

	originalFunc := OctoTokenFunc
	t.Cleanup(func() { OctoTokenFunc = originalFunc })

	callCount := 0
	OctoTokenFunc = func(_ context.Context, _, _, _ string) (string, error) {
		callCount++
		return fmt.Sprintf("token-%d", callCount), nil
	}

	ts := NewRepoTokenSource(ctx, "id", "org", "repo")

	first, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := ts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// oauth2.ReuseTokenSource should cache the first token until it expires.
	if first.AccessToken != second.AccessToken {
		t.Errorf("AccessToken: got = %q, want = %q (cached)", second.AccessToken, first.AccessToken)
	}
	if callCount != 1 {
		t.Errorf("OctoTokenFunc call count: got = %d, want = 1", callCount)
	}
}
