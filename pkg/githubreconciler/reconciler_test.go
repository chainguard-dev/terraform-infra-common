/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
)

// mockTokenSourceFunc returns a mock token source
func mockTokenSourceFunc(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
	return &mockTokenSource{token: fmt.Sprintf("token-%s-%s", org, repo)}, nil
}

// errorTokenSourceFunc returns an error
func errorTokenSourceFunc(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
	return nil, fmt.Errorf("mock token source error for %s/%s", org, repo)
}

func TestNewReconciler(t *testing.T) {
	cc := NewClientCache(mockTokenSourceFunc)

	// Test with default options
	r := NewReconciler(cc)
	if r.clientCache != cc {
		t.Error("Expected client cache to be set")
	}
	if r.stateManager == nil {
		t.Error("Expected default state manager to be created")
	}
	if r.stateManager.Identity() != "github-reconciler" {
		t.Errorf("Expected default identity 'github-reconciler', got %s", r.stateManager.Identity())
	}

	// Test with custom state manager
	customSM := NewStateManager("custom-identity")
	r2 := NewReconciler(cc, WithStateManager(customSM))
	if r2.stateManager != customSM {
		t.Error("Expected custom state manager to be used")
	}

	// Test with reconciler functions
	var issueCalled, prCalled bool
	issueFunc := func(_ context.Context, _ *Resource, _ *github.Client) error {
		issueCalled = true
		return nil
	}

	prFunc := func(_ context.Context, _ *Resource, _ *github.Client) error {
		prCalled = true
		return nil
	}

	r3 := NewReconciler(cc, WithIssueReconciler(issueFunc), WithPullRequestReconciler(prFunc))
	if r3.issueFunc == nil {
		t.Error("Expected issue reconciler to be set")
	}
	if r3.prFunc == nil {
		t.Error("Expected PR reconciler to be set")
	}

	// Test that the functions are actually the ones we provided
	ctx := context.Background()
	testResource := &Resource{Owner: "test", Repo: "test", Number: 1, Type: ResourceTypeIssue}

	r3.issueFunc(ctx, testResource, nil)
	if !issueCalled {
		t.Error("Expected issue function to be called")
	}

	r3.prFunc(ctx, testResource, nil)
	if !prCalled {
		t.Error("Expected PR function to be called")
	}
}

func TestReconciler_Reconcile(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		setupReconciler func(*Reconciler)
		tokenError      bool
		wantErr         bool
		wantErrContains string
		wantIssueCalled bool
		wantPRCalled    bool
	}{
		{
			name: "reconcile issue successfully",
			url:  "https://github.com/owner/repo/issues/123",
			setupReconciler: func(r *Reconciler) {
				r.issueFunc = func(_ context.Context, res *Resource, _ *github.Client) error {
					if res.Owner != "owner" || res.Repo != "repo" || res.Number != 123 {
						return fmt.Errorf("unexpected resource: %+v", res)
					}
					return nil
				}
			},
			wantIssueCalled: true,
		},
		{
			name: "reconcile PR successfully",
			url:  "https://github.com/owner/repo/pull/456",
			setupReconciler: func(r *Reconciler) {
				r.prFunc = func(_ context.Context, res *Resource, _ *github.Client) error {
					if res.Owner != "owner" || res.Repo != "repo" || res.Number != 456 {
						return fmt.Errorf("unexpected resource: %+v", res)
					}
					return nil
				}
			},
			wantPRCalled: true,
		},
		{
			name:            "invalid URL",
			url:             "not-a-url",
			setupReconciler: func(_ *Reconciler) {},
			wantErr:         true,
			wantErrContains: "parsing URL",
		},
		{
			name:            "no issue reconciler configured",
			url:             "https://github.com/owner/repo/issues/123",
			setupReconciler: func(_ *Reconciler) {},
			wantErr:         true,
			wantErrContains: "no reconciler configured for issue",
		},
		{
			name:            "no PR reconciler configured",
			url:             "https://github.com/owner/repo/pull/456",
			setupReconciler: func(_ *Reconciler) {},
			wantErr:         true,
			wantErrContains: "no reconciler configured for pull_request",
		},
		{
			name: "reconciler returns error",
			url:  "https://github.com/owner/repo/issues/123",
			setupReconciler: func(r *Reconciler) {
				r.issueFunc = func(_ context.Context, _ *Resource, _ *github.Client) error {
					return errors.New("reconciler error")
				}
			},
			wantErr:         true,
			wantErrContains: "reconciler error",
			wantIssueCalled: true,
		},
		{
			name:            "client cache error",
			url:             "https://github.com/owner/repo/issues/123",
			tokenError:      true,
			setupReconciler: func(_ *Reconciler) {},
			wantErr:         true,
			wantErrContains: "getting GitHub client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Setup client cache
			var cc *ClientCache
			if tt.tokenError {
				cc = NewClientCache(errorTokenSourceFunc)
			} else {
				cc = NewClientCache(mockTokenSourceFunc)
			}

			// Track calls
			issueCalled := false
			prCalled := false

			r := NewReconciler(cc)

			// Wrap functions to track calls
			if r.issueFunc != nil {
				origFunc := r.issueFunc
				r.issueFunc = func(_ context.Context, res *Resource, gh *github.Client) error {
					issueCalled = true
					return origFunc(ctx, res, gh)
				}
			}

			if r.prFunc != nil {
				origFunc := r.prFunc
				r.prFunc = func(_ context.Context, res *Resource, gh *github.Client) error {
					prCalled = true
					return origFunc(ctx, res, gh)
				}
			}

			// Apply test-specific setup
			tt.setupReconciler(r)

			// If the test setup added functions, wrap them too
			if r.issueFunc != nil && !issueCalled {
				origFunc := r.issueFunc
				r.issueFunc = func(_ context.Context, res *Resource, gh *github.Client) error {
					issueCalled = true
					return origFunc(ctx, res, gh)
				}
			}

			if r.prFunc != nil && !prCalled {
				origFunc := r.prFunc
				r.prFunc = func(_ context.Context, res *Resource, gh *github.Client) error {
					prCalled = true
					return origFunc(ctx, res, gh)
				}
			}

			err := r.Reconcile(ctx, tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("Reconcile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.wantErrContains != "" && !errors.Is(err, errors.New(tt.wantErrContains)) {
				if !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Reconcile() error = %v, want error containing %v", err, tt.wantErrContains)
				}
			}

			if issueCalled != tt.wantIssueCalled {
				t.Errorf("issue reconciler called = %v, want %v", issueCalled, tt.wantIssueCalled)
			}

			if prCalled != tt.wantPRCalled {
				t.Errorf("PR reconciler called = %v, want %v", prCalled, tt.wantPRCalled)
			}
		})
	}
}

func TestReconciler_GetStateManager(t *testing.T) {
	cc := NewClientCache(mockTokenSourceFunc)

	// Test with default state manager
	r1 := NewReconciler(cc)
	sm1 := r1.GetStateManager()
	if sm1 == nil {
		t.Fatal("GetStateManager() returned nil")
	}
	if sm1.Identity() != "github-reconciler" {
		t.Errorf("GetStateManager() identity = %v, want %v", sm1.Identity(), "github-reconciler")
	}

	// Test with custom state manager
	customSM := NewStateManager("custom-id")
	r2 := NewReconciler(cc, WithStateManager(customSM))
	sm2 := r2.GetStateManager()
	if sm2 != customSM {
		t.Error("GetStateManager() did not return the custom state manager")
	}
	if sm2.Identity() != "custom-id" {
		t.Errorf("GetStateManager() identity = %v, want %v", sm2.Identity(), "custom-id")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
