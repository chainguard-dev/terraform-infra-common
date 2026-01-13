/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package conformance

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/dispatcher"
)

func TestConcurrency(t *testing.T, ctor func(int) workqueue.Interface) {
	wq := ctor(5)
	if wq == nil {
		t.Fatal("NewWorkQueue returned nil")
	}

	inflight := int32(0)

	var cb dispatcher.Callback = func(_ context.Context, key string, _ workqueue.Options) error {
		newVal := atomic.AddInt32(&inflight, 1)
		defer atomic.AddInt32(&inflight, -1)
		if newVal > 5 {
			t.Errorf("Too many inflight: %d", newVal)
		}

		t.Logf("Processing %q", key)
		// This is intentionally much longer than the tick below, to ensure that
		// we handle multiple concurrent dispatch invocations.
		time.Sleep(time.Second)
		return nil
	}

	eg := errgroup.Group{}
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		if err := eg.Wait(); err != nil {
			t.Errorf("Error group failed: %v", err)
		}
	}()
	defer cancel()

	eg.Go(func() error {
		// This is intentionally MUCH lower than the sleep above, to ensure that
		// we see a lot of concurrent dispatch invocations.
		tick := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
				// Do this in a go routine, so it doesn't block the
				// dispatch loop.
				eg.Go(func() error {
					return dispatcher.Handle(context.WithoutCancel(ctx), wq, 5, 0, cb)
				})
			}
		}
	})

	for i := 0; i < 1000; i++ {
		bi, err := rand.Int(rand.Reader, big.NewInt(40))
		if err != nil {
			t.Fatalf("Failed to generate random number: %v", err)
		}

		if err := wq.Queue(ctx, bi.String(), workqueue.Options{}); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	for {
		wip, qd, _, err := wq.Enumerate(ctx)
		if err != nil {
			t.Fatalf("Enumerate failed: %v", err)
		}
		if len(wip) == 0 && len(qd) == 0 {
			break
		}
		t.Logf("Waiting for work to complete (wip: %d, qd: %d)", len(wip), len(qd))
		time.Sleep(100 * time.Millisecond)
	}
}
