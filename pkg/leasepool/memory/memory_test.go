/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memory_test

import (
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/internal/contract"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/memory"
)

func TestContract(t *testing.T) {
	p, err := memory.New(leasepool.Config{})
	if err != nil {
		t.Fatal(err)
	}
	contract.Test(t, func(*testing.T) leasepool.Interface { return p })
}

func TestRenewal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, err := memory.New(leasepool.Config{})
		if err != nil {
			t.Fatal(err)
		}
		l, err := p.TryAcquire(t.Context(), "renew")
		if err != nil {
			t.Fatal(err)
		}
		defer l.Release(t.Context())
		synctest.Wait()
		// Virtual time advances through multiple original TTLs and real renewals.
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		if err := l.Context().Err(); err != nil {
			t.Fatal(err)
		}
		if _, err := p.TryAcquire(t.Context(), "renew"); !errors.Is(err, leasepool.ErrBusy) {
			t.Fatalf("got %v, want busy", err)
		}
	})
}
