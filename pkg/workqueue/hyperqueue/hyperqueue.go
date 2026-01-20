/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package hyperqueue

import (
	"context"
	"errors"
	"hash/fnv"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

type server struct {
	workqueue.UnimplementedWorkqueueServiceServer
	backends []workqueue.WorkqueueServiceClient
}

// New creates a sharded WorkqueueServiceServer from the provided backend clients.
// The shard count is determined by len(backends).
// Keys are consistently routed to shards using hash(key) % len(backends).
func New(backends []workqueue.WorkqueueServiceClient) (workqueue.WorkqueueServiceServer, error) {
	if len(backends) == 0 {
		return nil, errors.New("at least one backend is required")
	}
	return &server{backends: backends}, nil
}

func (s *server) shardFor(key string) workqueue.WorkqueueServiceClient {
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.backends[int(h.Sum32())%len(s.backends)]
}

// Process routes the request to the appropriate shard based on the key.
func (s *server) Process(ctx context.Context, req *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
	return s.shardFor(req.GetKey()).Process(ctx, req)
}

// GetKeyState routes the request to the appropriate shard based on the key.
func (s *server) GetKeyState(ctx context.Context, req *workqueue.GetKeyStateRequest) (*workqueue.KeyState, error) {
	return s.shardFor(req.GetKey()).GetKeyState(ctx, req)
}
