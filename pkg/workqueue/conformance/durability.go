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
func TestDurability(t *testing.T, ctor func(uint) workqueue.Interface) {
	ctx := context.Background()

	// Queue some stuff
	{
		wq := ctor(5)
		if wq == nil {
			t.Fatal("NewWorkQueue returned nil")
		}

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
	}

	// Now with a separate instance of the workqueue, make sure it is still
	// there, and then drain it.
	{
		wq := ctor(5)
		if wq == nil {
			t.Fatal("NewWorkQueue returned nil")
		}

		// After we queue something, we should have one thing queued.
		wip, qd, err := wq.Enumerate(ctx)
		if err != nil {
			t.Fatalf("Enumerate failed: %v", err)
		}
		if want, got := 0, len(wip); want != got {
			t.Errorf("Expected %d in-progress keys, got %d", want, got)
		}
		if want, got := 1, len(qd); want != got {
			t.Errorf("Expected %d queued keys, got %d", want, got)
		}

		if err := drain(ctx, wq); err != nil {
			t.Fatalf("Drain failed: %v", err)
		}
	}
}
