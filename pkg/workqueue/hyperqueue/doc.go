/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package hyperqueue provides a sharded WorkqueueService implementation that
// consistently distributes keys across N backend workqueue services using
// consistent hashing.
//
// Hyperqueue acts as a transparent router that implements the WorkqueueService
// gRPC interface. It takes a slice of backend WorkqueueServiceClients and routes
// incoming requests to the appropriate shard based on the request key.
//
// Keys are assigned to shards using FNV-1a hashing:
//
//	hash(key) % len(backends) -> shard index
//
// This ensures:
//   - Deterministic routing: the same key always routes to the same shard
//   - Even distribution: keys are roughly evenly distributed across shards
//   - Fast computation: FNV-1a is a fast non-cryptographic hash
package hyperqueue
