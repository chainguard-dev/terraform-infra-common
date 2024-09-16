/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package conformance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

func TestSemantics(t *testing.T, ctor func(int) workqueue.Interface) {
	wq := ctor(5)
	if wq == nil {
		t.Fatal("NewWorkQueue returned nil")
	}

	ctx := context.Background()

	// Before we queue anything, we should have nothing in progress or queued.
	wip, qd, err := wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 0, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 0, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}

	// Queue a key!
	if err := wq.Queue(ctx, "foo"); err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	// After we queue something, we should have one thing queued.
	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 0, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 1, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}

	// Queue a new key!
	time.Sleep(1 * time.Millisecond)
	if err := wq.Queue(ctx, "bar"); err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	// After we queue something, we should have two things queued.
	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 0, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 2, len(qd); want != got {
		t.Fatalf("Expected %d queued keys, got %d", want, got)
	}
	if want, got := "foo", qd[0].Name(); want != got {
		t.Errorf("Expected first queued key to be %q, got %q", want, got)
	}

	// Start processing the first key.
	owned, err := qd[0].Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// After we start processing the first key, we should have one thing in progress and one thing queued.
	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 1, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 1, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}

	// Return the first key to the queue.
	time.Sleep(1 * time.Millisecond)
	if err := owned.Requeue(ctx); err != nil {
		t.Fatalf("Requeue failed: %v", err)
	}

	// After we return the first key to the queue, we should have both things queued,
	// but the first queued key should now be "bar".
	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 0, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 2, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}
	if want, got := "bar", qd[0].Name(); want != got {
		t.Errorf("Expected first queued key to be %q, got %q", want, got)
	}

	// Start processing the first key.
	owned, err = qd[0].Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Queue the in-progress key.
	if err := wq.Queue(ctx, owned.Name()); err != nil {
		t.Fatalf("Queue failed: %v", err)
	}

	// After we queue the in-progress key, we should have both things queued,
	// but the in-progress key is still in-progress.
	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 1, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 2, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}
	if want, got := "foo", qd[0].Name(); want != got {
		t.Errorf("Expected first queued key to be %q, got %q", want, got)
	}

	// Return the in-progress key to the queue.
	time.Sleep(1 * time.Millisecond)
	if err := owned.Requeue(ctx); err != nil {
		t.Fatalf("Requeue failed: %v", err)
	}

	// Now we should just have the two keys queued.
	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 0, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 2, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}
	if want, got := "foo", qd[0].Name(); want != got {
		t.Errorf("Expected first queued key to be %q, got %q", want, got)
	}

	// Start processing the first key.
	owned, err = qd[0].Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Check that we have a context that's live!
	select {
	case <-owned.Context().Done():
		t.Fatal("Context shouldn't complete yet!")
	case <-time.After(2 * time.Second):
		// Good!
	}

	// Complete the first key.
	if err := owned.Complete(ctx); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Check that the context was canceled.
	select {
	case <-owned.Context().Done():
		// Good!
	default:
		t.Fatal("Context should have completed!")
	}

	// Now we should just have the one key queued.
	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 0, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 1, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}
	if want, got := "bar", qd[0].Name(); want != got {
		t.Errorf("Expected first queued key to be %q, got %q", want, got)
	}

	// Queue more keys than the limit, and then check that we only return the
	// expected number of keys (the limit).
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Millisecond)
		if err := wq.Queue(ctx, fmt.Sprintf("key-%d", i)); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}
	}

	wip, qd, err = wq.Enumerate(ctx)
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}
	if want, got := 0, len(wip); want != got {
		t.Errorf("Expected %d in-progress keys, got %d", want, got)
	}
	if want, got := 5, len(qd); want != got {
		t.Errorf("Expected %d queued keys, got %d", want, got)
	}
	// The first key should be "bar"
	if want, got := "bar", qd[0].Name(); want != got {
		t.Errorf("Expected first queued key to be %q, got %q", want, got)
	}
	// The remaining 4 keys should be key-0 through key-3
	for i := 0; i < 4; i++ {
		if want, got := fmt.Sprintf("key-%d", i), qd[i+1].Name(); want != got {
			t.Errorf("Expected queued key %d to be %q, got %q", i, want, got)
		}
	}

	if err := drain(ctx, wq); err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
}

func drain(ctx context.Context, wq workqueue.Interface) error {
	for {
		wip, qd, err := wq.Enumerate(ctx)
		if err != nil {
			return fmt.Errorf("enumerate failed: %w", err)
		}
		if len(wip) == 0 && len(qd) == 0 {
			return nil
		}
		for _, k := range wip {
			if k.IsOrphaned() {
				if err := k.Requeue(ctx); err != nil {
					return fmt.Errorf("requeue failed: %w", err)
				}
			}
		}
		for _, k := range qd {
			owned, err := k.Start(ctx)
			if err != nil {
				return fmt.Errorf("start failed: %w", err)
			}
			if err := owned.Complete(ctx); err != nil {
				return fmt.Errorf("complete failed: %w", err)
			}
		}
	}
}
