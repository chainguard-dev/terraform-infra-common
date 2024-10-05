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
	"github.com/google/go-cmp/cmp"
)

type ExpectedState struct {
	WorkInProgress []string
	Queued         []string
}

func checkQueue(t *testing.T, wq workqueue.Interface, es ExpectedState) ([]workqueue.ObservedInProgressKey, []workqueue.QueuedKey) {
	t.Helper()
	wip, qd, err := wq.Enumerate(context.Background())
	if err != nil {
		t.Fatalf("Enumerate failed: %v", err)
	}

	var l []string // nolint: prealloc
	for _, k := range wip {
		l = append(l, k.Name())
	}
	if diff := cmp.Diff(es.WorkInProgress, l); diff != "" {
		t.Fatalf("Unexpected in-progress keys (-want, +got):\n%s", diff)
	}

	l = nil
	for _, k := range qd {
		l = append(l, k.Name())
	}
	if diff := cmp.Diff(es.Queued, l); diff != "" {
		t.Fatalf("Unexpected queued keys (-want, +got):\n%s", diff)
	}
	return wip, qd
}

type conformanceTester struct {
	t           *testing.T
	ctor        func(int) workqueue.Interface
	concurrency int
}

func (ct *conformanceTester) scenario(name string, f func(context.Context, *testing.T, workqueue.Interface)) {
	ct.t.Run(name, func(t *testing.T) {
		wq := ct.ctor(ct.concurrency)
		if wq == nil {
			t.Fatal("NewWorkQueue returned nil")
		}
		// For conformance, we always expect the queue to start empty, but drain
		// it because a durable queue will bleed across tests.
		if err := drain(wq); err != nil {
			t.Fatalf("Drain failed: %v", err)
		}

		_, _ = checkQueue(t, wq, ExpectedState{})

		t.Cleanup(func() {
			if err := drain(wq); err != nil {
				t.Fatalf("Drain failed: %v", err)
			}

			// Ensure we return to an empty queue.
			_, _ = checkQueue(t, wq, ExpectedState{})
		})

		f(context.Background(), t, wq)
	})
}

func TestSemantics(t *testing.T, ctor func(int) workqueue.Interface) {
	ct := &conformanceTester{
		t:           t,
		ctor:        ctor,
		concurrency: 5,
	}

	ct.scenario("simple queue ordering", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After we queue something, we should have one thing queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Queue another key, it should appear after the first.
		time.Sleep(1 * time.Millisecond) // Ensure a diff timestamp.
		if err := wq.Queue(ctx, "bar", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After we queue something, we should have two things queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo", "bar"},
		})
	})

	ct.scenario("queue more than concurrency limit", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue more keys than the limit, and then check that we only return the
		// expected number of keys (the limit).
		for i := 0; i < 5*ct.concurrency; i++ {
			time.Sleep(1 * time.Millisecond)
			if err := wq.Queue(ctx, fmt.Sprintf("key-%d", i), workqueue.Options{}); err != nil {
				t.Fatalf("Queue failed: %v", err)
			}
		}

		// Now we should see the limit number of keys.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"key-0", "key-1", "key-2", "key-3", "key-4"},
		})
	})

	ct.scenario("simple deduplication", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After we queue something, we should have one thing queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Queue the same key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// We should see exactly the same result.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})
	})

	ct.scenario("priority ordering", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			// No priority.
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After we queue something, we should have one thing queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Queue another key, it should appear after the first.
		time.Sleep(1 * time.Millisecond) // Ensure a diff timestamp.
		if err := wq.Queue(ctx, "bar", workqueue.Options{
			// No priority
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After we queue something, we should have two things queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo", "bar"},
		})

		// Queue the same key, but with a high priority.
		if err := wq.Queue(ctx, "bar", workqueue.Options{
			Priority: 1000,
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After queuing with a higher priority, we should see the order flip.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"bar", "foo"},
		})

		// Queue the first key, but with the same high priority.
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			Priority: 1000,
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The order should flip back.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo", "bar"},
		})

		// Queue the second key, but with an even higher priority.
		if err := wq.Queue(ctx, "bar", workqueue.Options{
			Priority: 1001,
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The order should flip again.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"bar", "foo"},
		})
	})

	ct.scenario("start and complete with context check", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After we queue something, we should have one thing queued.
		_, qd := checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the first key.
		owned, err := qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Check that the key is now in progress.
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

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

		// Check that the queue is now empty.
		_, _ = checkQueue(t, wq, ExpectedState{})
	})

	ct.scenario("start and requeue", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// Queue a second key.
		time.Sleep(1 * time.Millisecond) // Ensure a diff timestamp.
		if err := wq.Queue(ctx, "bar", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// We should have both keys queued.
		_, qd := checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo", "bar"},
		})

		// Start processing the first key.
		owned, err := qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Check that the key is now in progress.
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
			Queued:         []string{"bar"},
		})

		// Requeue the in-progress key.
		if err := owned.Requeue(ctx); err != nil {
			t.Fatalf("Requeue failed: %v", err)
		}

		// Check that the key is back in the queue, but after the other key.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"bar", "foo"},
		})
	})

	ct.scenario("start and queue", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// We should have the key queued.
		_, qd := checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the first key.
		owned, err := qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Queue the in-progress key.
		if err := wq.Queue(ctx, owned.Name(), workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// Check that the key is queued and in-progress.
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
			Queued:         []string{"foo"},
		})
	})

	ct.scenario("start queue and requeue with priority", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key with a high priority.
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			Priority: 1000,
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// Queue a second key with a low priority.
		time.Sleep(1 * time.Millisecond) // Ensure a diff timestamp.
		if err := wq.Queue(ctx, "bar", workqueue.Options{
			// No priority
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// We should have the key queued.
		_, qd := checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo", "bar"},
		})

		// Start processing the high-priority key.
		owned, err := qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// Queue the in-progress key with no priority.
		if err := wq.Queue(ctx, owned.Name(), workqueue.Options{
			// No priority
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// Check that the key is queued and in-progress, and the queued key
		// is behind the other key.
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
			Queued:         []string{"bar", "foo"},
		})

		// Requeue the in-progress high-priority key.
		if err := owned.Requeue(ctx); err != nil {
			t.Fatalf("Requeue failed: %v", err)
		}

		// Check that the order of the queued keys has flipped.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo", "bar"},
		})
	})
}

func drain(wq workqueue.Interface) error {
	for {
		wip, qd, err := wq.Enumerate(context.Background())
		if err != nil {
			return fmt.Errorf("enumerate failed: %w", err)
		}
		if len(wip) == 0 && len(qd) == 0 {
			return nil
		}
		for _, k := range wip {
			if err := k.Requeue(context.Background()); err != nil {
				return fmt.Errorf("requeue failed: %w", err)
			}
		}
		for _, k := range qd {
			owned, err := k.Start(context.Background())
			if err != nil {
				return fmt.Errorf("start failed: %w", err)
			}
			if err := owned.Complete(context.Background()); err != nil {
				return fmt.Errorf("complete failed: %w", err)
			}
		}
	}
}
