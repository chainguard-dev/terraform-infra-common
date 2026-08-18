/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package valkey_test

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/pkg/valkey"
	"github.com/redis/go-redis/v9"
)

func ExampleResolve() {
	ctx := context.Background()
	// Resolve looks up the PSC connect address and managed CA for a
	// Memorystore for Valkey instance. Pass the full resource name once at
	// boot and reuse the Endpoint for all NewClient calls.
	ep, err := valkey.Resolve(ctx, "projects/my-project/locations/us-central1/instances/my-instance")
	if err != nil {
		fmt.Println("resolve failed:", err)
		return
	}
	_ = ep
}

func ExampleNewClient() {
	ctx := context.Background()
	ep, err := valkey.Resolve(ctx, "projects/my-project/locations/us-central1/instances/my-instance")
	if err != nil {
		fmt.Println("resolve failed:", err)
		return
	}
	client, err := valkey.NewClient(ctx, ep)
	if err != nil {
		fmt.Println("dial failed:", err)
		return
	}
	defer client.Close()
	_ = client
}

func ExampleNewClient_withOption() {
	ctx := context.Background()
	ep, err := valkey.Resolve(ctx, "projects/my-project/locations/us-central1/instances/my-instance")
	if err != nil {
		fmt.Println("resolve failed:", err)
		return
	}
	// Options adjust pool sizing or timeouts before the client is built.
	client, err := valkey.NewClient(ctx, ep, func(uo *redis.UniversalOptions) {
		uo.PoolSize = 10
	})
	if err != nil {
		fmt.Println("dial failed:", err)
		return
	}
	defer client.Close()
	_ = client
}
