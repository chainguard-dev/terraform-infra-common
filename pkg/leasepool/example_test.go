/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package leasepool_test

import (
	"context"
	"fmt"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/memory"
)

func ExampleRun() {
	pool, err := memory.New(leasepool.Config{})
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err := leasepool.Run(ctx, pool, "maintenance", func(owned context.Context) error {
		if err := owned.Err(); err != nil {
			return err
		}
		fmt.Println("protected work")
		return nil
	}); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("released")
	// Output:
	// protected work
	// released
}

func ExampleInterface() {
	pool, err := memory.New(leasepool.Config{})
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	lease, err := pool.TryAcquire(ctx, "maintenance")
	if err != nil {
		fmt.Println(err)
		return
	}
	// Protected work uses lease.Context() and must stop before Release.
	fmt.Println(lease.Context().Err())
	cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer stop()
	if err := lease.Release(cleanup); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(context.Cause(lease.Context()))
	// Output:
	// <nil>
	// lease released
}
