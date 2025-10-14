/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package conformance

import (
	"context"
	"testing"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

// Unlike the other tests, this one is only expected for implementations that
// have some degree of durability.  For instance, the in-memory implementation
// will not pass this test.
func TestDurability(t *testing.T, ctor func(int) workqueue.Interface) {
	ctx := context.Background()

	// Queue some stuff
	{
		wq := ctor(5)
		if wq == nil {
			t.Fatal("NewWorkQueue returned nil")
		}
		t.Cleanup(func() {
			if err := drain(wq); err != nil {
				t.Fatalf("Drain failed: %v", err)
			}

			// Ensure we return to an empty queue.
			_, _ = checkQueue(t, wq, ExpectedState{})
		})

		// Before we queue anything, we should have nothing in progress or queued.
		_, _ = checkQueue(t, wq, ExpectedState{})

		// Queue a key!
		if err := wq.Queue(ctx, "foo", workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}

		// After we queue something, we should have one thing queued.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})
	}

	// Now with a separate instance of the workqueue, make sure it is still
	// there, and then drain it.
	{
		wq := ctor(5)
		if wq == nil {
			t.Fatal("NewWorkQueue returned nil")
		}

		// A durable workqueue should still have the item in the queue.
		_, _ = checkQueue(t, wq, ExpectedState{
			Queued: []string{"foo"},
		})
	}
}
