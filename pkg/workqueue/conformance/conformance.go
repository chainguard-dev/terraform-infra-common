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

	// Use this, which the implementations can adjust to a suitable delay.
	delay := workqueue.BackoffPeriod

	// Cap the maximum backoff to 2x the delay, so that tests run in a
	// reasonable amount of time.
	workqueue.MaximumBackoffPeriod = 2 * delay

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

		// Check that "foo" has disappeared because it now has a NotBefore
		// set due to the backoff of the requeue.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"bar"},
		})

		// Sleep for the backoff period.
		time.Sleep(workqueue.BackoffPeriod)

		// Now "foo" should be back and before "bar" because it has a higher
		// priority.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo", "bar"},
		})
	})

	ct.scenario("simple not before", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key with NotBefore.
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			NotBefore: time.Now().UTC().Add(delay),
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should appear empty.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the NotBefore delay.
		time.Sleep(delay)

		// The queue should now have the key.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Queue the same key again with a NotBefore.
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			NotBefore: time.Now().UTC().Add(delay),
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should appear empty because the later NotBefore won.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Queue the same key again with a NotBefore that's twice as long.
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			NotBefore: time.Now().UTC().Add(2 * delay),
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should appear empty
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the NotBefore delay.
		time.Sleep(delay)

		// The queue should STILL appear empty because the doubled delay
		// should have won.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the NotBefore delay one last time.
		time.Sleep(delay)

		// The queue should now have the key.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})
	})

	ct.scenario("queue not before with priorities", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue the first key without NotBefore or Priority
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			// No NotBefore or Priority
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should have the key.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Queue a second key with a short NotBefore and higher Priority
		if err := wq.Queue(ctx, "bar", workqueue.Options{
			NotBefore: time.Now().UTC().Add(delay),
			Priority:  1000,
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// Initially the queue doesn't show the key
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Sleep for the NotBefore delay.
		time.Sleep(delay)

		// Now the second key appears and jumps the queue.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"bar", "foo"},
		})
	})

	ct.scenario("requeue doesn't reset not before", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key without NotBefore set.
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			// No NotBefore.
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should have the key.
		_, qd := checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the first key.
		owned, err := qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// The key should now be in progress
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

		// Queue the key again with a short NotBefore delay.
		if err := wq.Queue(ctx, owned.Name(), workqueue.Options{
			NotBefore: time.Now().UTC().Add(delay),
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should still have the key just in-progress.
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

		// Requeue the key.
		if err := owned.Requeue(ctx); err != nil {
			t.Fatalf("Requeue failed: %v", err)
		}

		// The requeue should not reset NotBefore.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the NotBefore delay.
		time.Sleep(delay)

		// Now the key should show as queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})
	})

	ct.scenario("requeuing a priority task has backoff", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key without NotBefore set, but with a Priority
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			Priority: 1000,
			// No NotBefore.
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should have the key.
		_, qd := checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the first key.
		owned, err := qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// The key should now be in progress
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

		// Requeue the key.
		if err := owned.Requeue(ctx); err != nil {
			t.Fatalf("Requeue failed: %v", err)
		}

		// The key shouldn't appear until the backoff period has passed.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the backoff period.
		time.Sleep(workqueue.BackoffPeriod)

		// Now the key should show as queued.
		_, qd = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the key again.
		owned, err = qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// The key should now be in progress
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

		// Requeue the key.
		if err := owned.Requeue(ctx); err != nil {
			t.Fatalf("Requeue failed: %v", err)
		}

		// The key shouldn't appear until 2x the backoff period has passed.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the backoff period.
		time.Sleep(workqueue.BackoffPeriod)

		// The key shouldn't appear until 2x the backoff period has passed.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the backoff period AGAIN.
		time.Sleep(workqueue.BackoffPeriod)

		// Now the key should show as queued.
		_, qd = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the key a final time.
		owned, err = qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// The key should now be in progress
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

		// Requeue the key.
		if err := owned.Requeue(ctx); err != nil {
			t.Fatalf("Requeue failed: %v", err)
		}

		// The key shouldn't appear until 2x the backoff period has passed
		// because we capped the maximum backoff at 2x.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the backoff period.
		time.Sleep(2 * workqueue.BackoffPeriod)

		// Now the key should show as queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
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

// TestMaxRetry tests the max retry functionality with Fail method
func TestMaxRetry(t *testing.T, ctor func(int) workqueue.Interface) {
	ct := &conformanceTester{
		t:           t,
		ctor:        ctor,
		concurrency: 5,
	}

	// Use this, which the implementations can adjust to a suitable delay.
	delay := workqueue.BackoffPeriod

	// Cap the maximum backoff to 2x the delay, so that tests run in a
	// reasonable amount of time.
	workqueue.MaximumBackoffPeriod = 2 * delay

	ct.scenario("max retry limit with fail", func(ctx context.Context, t *testing.T, wq workqueue.Interface) {
		// Queue a key with a priority
		if err := wq.Queue(ctx, "foo", workqueue.Options{
			Priority: 1000,
		}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// The queue should have the key.
		_, qd := checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the key
		owned, err := qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// The key should now be in progress
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

		// Get the initial attempt count, should be 1
		attempts := owned.GetAttempts()
		if attempts != 1 {
			t.Fatalf("Expected attempt count 1, got %d", attempts)
		}

		// Requeue the key to increment the retry count
		if err := owned.Requeue(ctx); err != nil {
			t.Fatalf("Requeue failed: %v", err)
		}

		// The key shouldn't appear until the backoff period has passed.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Sleep for the backoff period.
		time.Sleep(workqueue.BackoffPeriod)

		// Now the key should show as queued.
		_, qd = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})

		// Start processing the key again
		owned, err = qd[0].Start(ctx)
		if err != nil {
			t.Fatalf("Start failed: %v", err)
		}

		// The key should now be in progress
		_, _ = checkQueue(t, wq, ExpectedState{
			WorkInProgress: []string{"foo"},
		})

		// Get the attempt count, should now be 2
		attempts = owned.GetAttempts()
		if attempts != 2 {
			t.Fatalf("Expected attempt count 2, got %d", attempts)
		}

		// Now fail the key instead of requeuing it
		if err := owned.Deadletter(ctx); err != nil {
			t.Fatalf("Fail failed: %v", err)
		}

		// After failing the task, it should be removed from both queued and in-progress
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Make sure it doesn't show up again even after waiting
		time.Sleep(2 * workqueue.BackoffPeriod)
		_, _ = checkQueue(t, wq, ExpectedState{})
	})
}
