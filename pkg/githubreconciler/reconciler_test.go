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
	"time"

	"github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
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

	// Test with reconciler function
	var reconcileCalled bool
	var calledResourceType ResourceType
	reconcileFunc := func(_ context.Context, res *Resource, _ *github.Client) error {
		reconcileCalled = true
		calledResourceType = res.Type
		return nil
	}

	r3 := NewReconciler(cc, WithReconciler(reconcileFunc))
	if r3.reconcileFunc == nil {
		t.Error("Expected reconciler to be set")
	}

	// Test that the function is actually the one we provided
	ctx := context.Background()
	testIssueResource := &Resource{Owner: "test", Repo: "test", Number: 1, Type: ResourceTypeIssue}
	testPRResource := &Resource{Owner: "test", Repo: "test", Number: 2, Type: ResourceTypePullRequest}

	r3.reconcileFunc(ctx, testIssueResource, nil)
	if !reconcileCalled {
		t.Error("Expected reconcile function to be called")
	}
	if calledResourceType != ResourceTypeIssue {
		t.Errorf("Expected resource type to be issue, got %s", calledResourceType)
	}

	reconcileCalled = false
	r3.reconcileFunc(ctx, testPRResource, nil)
	if !reconcileCalled {
		t.Error("Expected reconcile function to be called")
	}
	if calledResourceType != ResourceTypePullRequest {
		t.Errorf("Expected resource type to be pull_request, got %s", calledResourceType)
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
				r.reconcileFunc = func(_ context.Context, res *Resource, _ *github.Client) error {
					if res.Owner != "owner" || res.Repo != "repo" || res.Number != 123 || res.Type != ResourceTypeIssue {
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
				r.reconcileFunc = func(_ context.Context, res *Resource, _ *github.Client) error {
					if res.Owner != "owner" || res.Repo != "repo" || res.Number != 456 || res.Type != ResourceTypePullRequest {
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
			name:            "no reconciler configured",
			url:             "https://github.com/owner/repo/issues/123",
			setupReconciler: func(_ *Reconciler) {},
			wantErr:         true,
			wantErrContains: "no reconciler configured",
		},
		{
			name: "reconciler returns error",
			url:  "https://github.com/owner/repo/issues/123",
			setupReconciler: func(r *Reconciler) {
				r.reconcileFunc = func(_ context.Context, _ *Resource, _ *github.Client) error {
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

			// Apply test-specific setup
			tt.setupReconciler(r)

			// Wrap function to track calls
			if r.reconcileFunc != nil {
				origFunc := r.reconcileFunc
				r.reconcileFunc = func(_ context.Context, res *Resource, gh *github.Client) error {
					switch res.Type {
					case ResourceTypeIssue:
						issueCalled = true
					case ResourceTypePullRequest:
						prCalled = true
					}
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

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestReconciler_RateLimitHandling(t *testing.T) {
	tests := []struct {
		name             string
		reconcilerError  error
		wantRequeue      bool
		wantRequeueAfter time.Duration
		wantErrContains  string
	}{{
		name:             "gRPC ResourceExhausted error triggers requeue",
		reconcilerError:  status.Error(codes.ResourceExhausted, "resource exhausted"),
		wantRequeue:      true,
		wantRequeueAfter: 2 * time.Minute,
	}, {
		name:             "wrapped gRPC ResourceExhausted error triggers requeue",
		reconcilerError:  fmt.Errorf("failed to call API: %w", status.Error(codes.ResourceExhausted, "quota exceeded")),
		wantRequeue:      true,
		wantRequeueAfter: 2 * time.Minute,
	}, {
		name:             "GitHub RateLimitError triggers requeue",
		reconcilerError:  &github.RateLimitError{Rate: github.Rate{Reset: github.Timestamp{Time: time.Now().Add(5 * time.Minute)}}},
		wantRequeue:      true,
		wantRequeueAfter: 5 * time.Minute,
	}, {
		name:             "GitHub AbuseRateLimitError with RetryAfter triggers requeue",
		reconcilerError:  &github.AbuseRateLimitError{RetryAfter: ptrDuration(2 * time.Minute)},
		wantRequeue:      true,
		wantRequeueAfter: 2 * time.Minute,
	}, {
		name:             "GitHub AbuseRateLimitError without RetryAfter uses default",
		reconcilerError:  &github.AbuseRateLimitError{},
		wantRequeue:      true,
		wantRequeueAfter: time.Minute,
	}, {
		name:            "non-rate-limit error does not trigger requeue",
		reconcilerError: errors.New("regular error"),
		wantRequeue:     false,
		wantErrContains: "regular error",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cc := NewClientCache(mockTokenSourceFunc)

			r := NewReconciler(cc, WithReconciler(func(_ context.Context, _ *Resource, _ *github.Client) error {
				return tt.reconcilerError
			}))

			err := r.Reconcile(ctx, "https://github.com/owner/repo/issues/123")

			if tt.wantRequeue {
				delay, ok := workqueue.GetRequeueDelay(err)
				if !ok {
					t.Errorf("Expected requeue error, got = %v", err)
				}
				// Jitter adds 0% to +100% to the base delay
				minDelay := tt.wantRequeueAfter
				maxDelay := tt.wantRequeueAfter + (tt.wantRequeueAfter)
				if delay < minDelay || delay > maxDelay {
					t.Errorf("Requeue delay = %v, want between %v and %v", delay, minDelay, maxDelay)
				}
			} else {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if _, ok := workqueue.GetRequeueDelay(err); ok {
					t.Errorf("Did not expect requeue error, got = %v", err)
				}
				if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Error = %v, want error containing %v", err, tt.wantErrContains)
				}
			}
		})
	}
}

func ptrDuration(d time.Duration) *time.Duration {
	return &d
}
