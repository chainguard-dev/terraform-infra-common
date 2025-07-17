/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/inmem"
)

func TestRequeueWithDelay(t *testing.T) {
	ctx := context.Background()
	wq := inmem.NewWorkQueue(10)

	// Queue a test key
	key := "test-key"
	if err := wq.Queue(ctx, key, workqueue.Options{Priority: 1}); err != nil {
		t.Fatalf("Failed to queue key: %v", err)
	}

	// Define test cases
	tests := []struct {
		name          string
		callback      Callback
		wantRequeued  bool
		wantMinDelay  time.Duration
		wantCompleted bool
	}{
		{
			name: "successful processing",
			callback: func(_ context.Context, _ string, _ workqueue.Options) error {
				return nil
			},
			wantCompleted: true,
		},
		{
			name: "requeue with 5 second delay",
			callback: func(_ context.Context, _ string, _ workqueue.Options) error {
				return workqueue.RequeueAfter(5 * time.Second)
			},
			wantRequeued: true,
			wantMinDelay: 5 * time.Second,
		},
		{
			name: "requeue with 1 minute delay",
			callback: func(_ context.Context, _ string, _ workqueue.Options) error {
				return workqueue.RequeueAfter(time.Minute)
			},
			wantRequeued: true,
			wantMinDelay: time.Minute,
		},
		{
			name: "non-retriable error",
			callback: func(_ context.Context, _ string, _ workqueue.Options) error {
				return workqueue.NonRetriableError(context.Canceled, "test non-retriable")
			},
			wantCompleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Queue the key
			if err := wq.Queue(ctx, key, workqueue.Options{Priority: 1}); err != nil {
				t.Fatalf("Failed to queue key: %v", err)
			}

			// Process with our test callback
			if err := Handle(ctx, wq, 1, tt.callback); err != nil {
				t.Fatalf("Handle failed: %v", err)
			}

			// Check the results
			wip, queued, err := wq.Enumerate(ctx)
			if err != nil {
				t.Fatalf("Failed to enumerate: %v", err)
			}

			if tt.wantCompleted {
				// Should not be in WIP or queued
				if len(wip) > 0 {
					t.Errorf("Expected no WIP items, got %d", len(wip))
				}
				if len(queued) > 0 {
					t.Errorf("Expected no queued items, got %d", len(queued))
				}
			} else if tt.wantRequeued {
				// Should be requeued but with a delay, so it won't show up in Enumerate yet
				if len(wip) > 0 {
					t.Errorf("Expected no WIP items after requeue, got %d", len(wip))
				}

				// The item should be queued but not visible due to the delay
				// To verify it was requeued, we need to check the internal state
				// For now, we can verify by waiting or by checking that it's not immediately available
				if len(queued) > 0 {
					// If we see queued items, they should not be startable due to delay
					qk := queued[0]
					_, err := qk.Start(ctx)
					if err != nil {
						t.Errorf("Could not start queued item, which suggests it has a future NotBefore: %v", err)
					}
				} else {
					// This is expected - the item is queued with a future NotBefore time
					// Let's verify by checking that after a small wait, we still don't see it
					// (since our delays are much longer than a millisecond)
					time.Sleep(10 * time.Millisecond)
					_, queued2, err := wq.Enumerate(ctx)
					if err != nil {
						t.Fatalf("Failed to enumerate after delay: %v", err)
					}
					if len(queued2) > 0 {
						t.Errorf("Item appeared in queue too soon - delay might not be working")
					}
				}
			}
		})
	}
}

func TestServiceCallbackWithDelay(t *testing.T) {
	// Create a mock client that returns a response with RequeueAfterSeconds
	mockClient := &mockWorkqueueClient{
		processFunc: func(_ context.Context, _ *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
			return &workqueue.ProcessResponse{
				RequeueAfterSeconds: 30,
			}, nil
		},
	}

	callback := ServiceCallback(mockClient)
	err := callback(context.Background(), "test-key", workqueue.Options{})

	// Should return a requeue error
	delay, ok := workqueue.GetRequeueDelay(err)
	if !ok {
		t.Fatalf("Expected requeue error, got: %v", err)
	}
	if delay != 30*time.Second {
		t.Errorf("Expected 30 second delay, got %v", delay)
	}
}

// Mock client for testing
type mockWorkqueueClient struct {
	workqueue.WorkqueueServiceClient
	processFunc func(context.Context, *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error)
}

func (m *mockWorkqueueClient) Process(ctx context.Context, req *workqueue.ProcessRequest, _ ...grpc.CallOption) (*workqueue.ProcessResponse, error) {
	return m.processFunc(ctx, req)
}
