/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"context"
	"sync"

	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool"
	"github.com/chainguard-dev/terraform-infra-common/pkg/leasepool/internal/engine"
)

// New creates a pool with no external dependencies.
func New(cfg leasepool.Config) (leasepool.Interface, error) {
	return engine.New(&store{entries: make(map[string]entry)}, cfg)
}

type entry struct {
	record engine.Record
	gen    int64
}
type store struct {
	mu      sync.Mutex
	entries map[string]entry
	next    int64
}

func (s *store) Read(ctx context.Context, name string) (engine.Record, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return engine.Record{}, 0, err
	}
	e := s.entries[name]
	return e.record, e.gen, nil
}

func (s *store) Write(ctx context.Context, name string, rec engine.Record, gen int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s.entries[name].gen != gen {
		return 0, engine.ErrConflict
	}
	s.next++
	s.entries[name] = entry{record: rec, gen: s.next}
	return s.next, nil
}

var _ engine.Store = (*store)(nil)
