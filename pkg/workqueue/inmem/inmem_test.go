/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package inmem

import (
	"context"
	"testing"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/conformance"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWorkQueue(t *testing.T) {
	// Adjust this to a suitable period for testing things.
	// The conformance tests own adjusting MaximumBackoffPeriod.
	workqueue.BackoffPeriod = 1 * time.Second

	conformance.TestSemantics(t, NewWorkQueue)

	conformance.TestConcurrency(t, NewWorkQueue)

	conformance.TestMaxRetry(t, NewWorkQueue)
}

func Test_Get_KeyNotFound(t *testing.T) {
	wq := NewWorkQueue(10)
	ctx := context.Background()

	resp, err := wq.Get(ctx, "nonexistent-key")
	if err == nil {
		t.Fatal("Expected error for nonexistent key")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected gRPC status error, got %T", err)
	}

	if st.Code() != codes.NotFound {
		t.Errorf("Expected NotFound status code, got %v", st.Code())
	}

	if resp != nil {
		t.Error("Expected nil response for not found key")
	}
}

func Test_Get_QueuedKey(t *testing.T) {
	wq := NewWorkQueue(10)
	ctx := context.Background()

	// Queue a key
	err := wq.Queue(ctx, "test-key", workqueue.Options{
		Priority: 5,
	})
	if err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	// Get the key
	resp, err := wq.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if resp.Status != workqueue.KeyState_QUEUED {
		t.Errorf("Expected status QUEUED, got %v", resp.Status)
	}
	if resp.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got %v", resp.Key)
	}
	if resp.Priority != 5 {
		t.Errorf("Expected priority 5, got %v", resp.Priority)
	}
	if resp.QueuedTime == 0 {
		t.Error("Expected queued timestamp to be non-zero")
	}
}

func Test_Get_InProgressKey(t *testing.T) {
	wq := NewWorkQueue(10)
	ctx := context.Background()

	// Queue a key
	err := wq.Queue(ctx, "test-key", workqueue.Options{
		Priority: 3,
	})
	if err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	// Start processing the key
	_, qd, err := wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}

	if len(qd) == 0 {
		t.Fatal("Expected at least one queued key")
	}

	_, err = qd[0].Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Get the key
	resp, err := wq.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if resp.Status != workqueue.KeyState_IN_PROGRESS {
		t.Errorf("Expected status IN_PROGRESS, got %v", resp.Status)
	}
	if resp.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got %v", resp.Key)
	}
}
