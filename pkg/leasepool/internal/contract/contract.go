/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package contract

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
)

// Test runs the same public operations against each leasepool backend.
// newPool returns another client to the same namespace.
func Test(t *testing.T, newPool func(*testing.T) leasepool.Interface) {
	t.Helper()
	t.Run("exclusive_and_independent_names", func(t *testing.T) {
		a, b := newPool(t), newPool(t)
		l, err := a.TryAcquire(t.Context(), t.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := l.Release(t.Context()); err != nil {
				t.Error(err)
			}
		}()
		if _, err := b.TryAcquire(t.Context(), t.Name()); !errors.Is(err, leasepool.ErrBusy) {
			t.Fatalf("contender: got %v, want busy", err)
		}
		other, err := b.TryAcquire(t.Context(), t.Name()+"/other")
		if err != nil {
			t.Fatal(err)
		}
		if err := other.Release(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := l.Release(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !errors.Is(context.Cause(l.Context()), leasepool.ErrReleased) {
			t.Fatalf("cause: got %v, want released", context.Cause(l.Context()))
		}
		successor, err := b.TryAcquire(t.Context(), t.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := successor.Release(t.Context()); err != nil {
				t.Error(err)
			}
		}()
		if err := l.Release(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := a.TryAcquire(t.Context(), t.Name()); !errors.Is(err, leasepool.ErrBusy) {
			t.Fatalf("stale release removed successor: %v", err)
		}
	})
	t.Run("concurrent_acquisition", func(t *testing.T) {
		const attempts = 12
		clients := make([]leasepool.Interface, attempts)
		for i := range clients {
			clients[i] = newPool(t)
		}
		winners := make(chan leasepool.Lease, attempts)
		start := make(chan struct{})
		var wg sync.WaitGroup
		for _, p := range clients {
			wg.Go(func() {
				<-start
				l, err := p.TryAcquire(t.Context(), t.Name())
				if err == nil {
					winners <- l
				} else if !errors.Is(err, leasepool.ErrBusy) {
					t.Errorf("acquire: %v", err)
				}
			})
		}
		close(start)
		wg.Wait()
		close(winners)
		count := 0
		for l := range winners {
			count++
			if err := l.Release(t.Context()); err != nil {
				t.Error(err)
			}
		}
		if count != 1 {
			t.Fatalf("winners: got %d, want 1", count)
		}
	})
	t.Run("parent_cancellation", func(t *testing.T) {
		p := newPool(t)
		ctx, cancel := context.WithCancel(t.Context())
		l, err := p.TryAcquire(ctx, t.Name())
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		<-l.Context().Done()
		if !errors.Is(context.Cause(l.Context()), context.Canceled) {
			t.Fatal(context.Cause(l.Context()))
		}
		// Cancellation stops renewal, but must not release while work is stopping.
		if _, err := p.TryAcquire(t.Context(), t.Name()); !errors.Is(err, leasepool.ErrBusy) {
			t.Fatalf("canceled owner was released early: %v", err)
		}
		if err := l.Release(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := p.TryAcquire(ctx, t.Name()); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled acquisition: %v", err)
		}
	})
	t.Run("run", func(t *testing.T) {
		p := newPool(t)
		want := errors.New("work failed")
		if err := leasepool.Run(t.Context(), p, t.Name(), func(context.Context) error { return want }); !errors.Is(err, want) {
			t.Fatalf("got %v, want work error", err)
		}
		l, err := p.TryAcquire(t.Context(), t.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := l.Release(t.Context()); err != nil {
				t.Error(err)
			}
		}()
		if err := leasepool.Run(t.Context(), p, t.Name(), func(context.Context) error { t.Error("busy callback ran"); return nil }); !errors.Is(err, leasepool.ErrBusy) {
			t.Fatalf("got %v, want busy", err)
		}
	})
	t.Run("invalid_name", func(t *testing.T) {
		if _, err := newPool(t).TryAcquire(t.Context(), ""); err == nil {
			t.Fatal("empty name accepted")
		}
	})
}
