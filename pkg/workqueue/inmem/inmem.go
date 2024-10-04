/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package inmem

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

// NewWorkQueue creates a new in-memory workqueue.
// This is intended for testing, and is not suitable for production use.
func NewWorkQueue(limit int) workqueue.Interface {
	return &wq{
		limit: limit,

		wip:   make(map[string]struct{}, limit),
		queue: make(map[string]queueItem, 10),
	}
}

type wq struct {
	limit int

	// rw guards the key sets.
	rw    sync.RWMutex
	wip   map[string]struct{}
	queue map[string]queueItem
}

type queueItem struct {
	queued   time.Time
	priority int64
}

var _ workqueue.Interface = (*wq)(nil)

// Queue implements workqueue.Interface.
func (w *wq) Queue(_ context.Context, key string, opts workqueue.Options) error {
	w.rw.Lock()
	defer w.rw.Unlock()
	if qi, ok := w.queue[key]; !ok {
		w.queue[key] = queueItem{
			queued:   time.Now().UTC(),
			priority: opts.Priority,
		}
	} else if qi.priority < opts.Priority {
		// Raise the priority of the queued item.
		qi.priority = opts.Priority
		w.queue[key] = qi
	}
	return nil
}

// Enumerate implements workqueue.Interface.
func (w *wq) Enumerate(_ context.Context) ([]workqueue.ObservedInProgressKey, []workqueue.QueuedKey, error) {
	w.rw.RLock()
	defer w.rw.RUnlock()
	wip := make([]workqueue.ObservedInProgressKey, 0, len(w.wip))
	qd := make([]struct {
		key      string
		priority int64
		ts       time.Time
	}, 0, w.limit+1)

	for k := range w.wip {
		wip = append(wip, &inProgressKey{
			wq:  w,
			key: k,
		})
	}

	// Collect the top "limit" queued keys.
	for k, ts := range w.queue {
		qd = append(qd, struct {
			key      string
			priority int64
			ts       time.Time
		}{
			key:      k,
			priority: ts.priority,
			ts:       ts.queued,
		})
		sort.Slice(qd, func(i, j int) bool {
			if qd[i].priority == qd[j].priority {
				return qd[i].ts.Before(qd[j].ts)
			}
			return qd[i].priority > qd[j].priority
		})
		if len(qd) > w.limit {
			qd = qd[:w.limit]
		}
	}

	qk := make([]workqueue.QueuedKey, 0, len(qd))
	for _, q := range qd {
		qk = append(qk, &queuedKey{
			wq:       w,
			key:      q.key,
			priority: q.priority,
		})
	}
	return wip, qk, nil
}

type inProgressKey struct {
	wq       *wq
	key      string
	priority int64

	ownerCtx    context.Context
	ownerCancel context.CancelFunc
}

var _ workqueue.ObservedInProgressKey = (*inProgressKey)(nil)
var _ workqueue.OwnedInProgressKey = (*inProgressKey)(nil)

// Name implements workqueue.Key.
func (o *inProgressKey) Name() string {
	return o.key
}

// Priority implements workqueue.Key.
func (o *inProgressKey) Priority() int64 {
	return o.priority
}

// Requeue implements workqueue.InProgressKey.
func (o *inProgressKey) Requeue(_ context.Context) error {
	if o.ownerCancel != nil {
		o.ownerCancel()
	}
	o.wq.rw.Lock()
	defer o.wq.rw.Unlock()
	if qi, ok := o.wq.queue[o.key]; !ok {
		o.wq.queue[o.key] = queueItem{
			queued:   time.Now().UTC(),
			priority: o.Priority(),
		}
	} else if qi.priority < o.Priority() {
		// Raise the priority of the queued item.
		qi.priority = o.Priority()
		o.wq.queue[o.key] = qi
	}
	delete(o.wq.wip, o.key)
	return nil
}

// IsOrphaned implements workqueue.ObservedInProgressKey.
func (o *inProgressKey) IsOrphaned() bool {
	return false
}

// Complete implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) Complete(_ context.Context) error {
	o.ownerCancel()
	o.wq.rw.Lock()
	defer o.wq.rw.Unlock()
	delete(o.wq.wip, o.key)
	return nil
}

// Context implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) Context() context.Context {
	return o.ownerCtx
}

type queuedKey struct {
	wq       *wq
	key      string
	priority int64
}

var _ workqueue.QueuedKey = (*queuedKey)(nil)

// Name implements workqueue.Key.
func (q *queuedKey) Name() string {
	return q.key
}

// Priority implements workqueue.Key.
func (q *queuedKey) Priority() int64 {
	return q.priority
}

// Start implements workqueue.QueuedKey.
func (q *queuedKey) Start(ctx context.Context) (workqueue.OwnedInProgressKey, error) {
	q.wq.rw.Lock()
	defer q.wq.rw.Unlock()
	if _, ok := q.wq.wip[q.key]; ok {
		// This should never happen, unless we have bad locking.
		return nil, fmt.Errorf("key %q already in progress", q.key)
	}
	if _, ok := q.wq.queue[q.key]; !ok {
		// This should never happen, unless we have bad locking.
		return nil, fmt.Errorf("key %q has disappeared from the backlog", q.key)
	}

	delete(q.wq.queue, q.key)
	q.wq.wip[q.key] = struct{}{}

	ctx, cancel := context.WithCancel(ctx)
	return &inProgressKey{
		wq:          q.wq,
		key:         q.key,
		priority:    q.Priority(),
		ownerCtx:    ctx,
		ownerCancel: cancel,
	}, nil
}
