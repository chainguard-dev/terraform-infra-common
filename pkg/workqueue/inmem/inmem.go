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
	workqueue.Options
	attempts int
	queued   time.Time
}

var _ workqueue.Interface = (*wq)(nil)

// Queue implements workqueue.Interface.
func (w *wq) Queue(_ context.Context, key string, opts workqueue.Options) error {
	w.rw.Lock()
	defer w.rw.Unlock()
	if qi, ok := w.queue[key]; !ok {
		w.queue[key] = queueItem{
			Options: opts,
			queued:  time.Now().UTC(),
		}
	} else if qi.Priority < opts.Priority {
		// Raise the priority of the queued item.
		qi.Priority = opts.Priority
		w.queue[key] = qi
	} else if qi.NotBefore.Before(opts.NotBefore) {
		// Update the NotBefore time.
		qi.NotBefore = opts.NotBefore
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
		workqueue.Options
		key      string
		attempts int
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
		if time.Now().UTC().Before(ts.NotBefore) {
			// Skip keys that are not ready to be processed.
			continue
		}
		qd = append(qd, struct {
			workqueue.Options
			key      string
			attempts int
			ts       time.Time
		}{
			Options:  ts.Options,
			key:      k,
			attempts: ts.attempts,
			ts:       ts.queued,
		})
		sort.Slice(qd, func(i, j int) bool {
			if qd[i].Priority == qd[j].Priority {
				return qd[i].ts.Before(qd[j].ts)
			}
			return qd[i].Priority > qd[j].Priority
		})
		if len(qd) > w.limit {
			qd = qd[:w.limit]
		}
	}

	qk := make([]workqueue.QueuedKey, 0, len(qd))
	for _, q := range qd {
		qk = append(qk, &queuedKey{
			Options:  q.Options,
			wq:       w,
			key:      q.key,
			attempts: q.attempts,
		})
	}
	return wip, qk, nil
}

type inProgressKey struct {
	workqueue.Options

	attempts int

	wq  *wq
	key string

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
	return o.Options.Priority
}

// Requeue implements workqueue.InProgressKey.
func (o *inProgressKey) Requeue(ctx context.Context) error {
	// Use RequeueWithOptions with an empty options struct to get default behavior
	return o.RequeueWithOptions(ctx, workqueue.Options{})
}

// RequeueWithOptions implements workqueue.InProgressKey.
func (o *inProgressKey) RequeueWithOptions(_ context.Context, opts workqueue.Options) error {
	if o.ownerCancel != nil {
		o.ownerCancel()
	}

	// If no priority specified in opts, use the current priority
	if opts.Priority == 0 {
		opts.Priority = o.Priority()
	}

	// Handle custom delay if specified
	if opts.Delay > 0 {
		opts.NotBefore = time.Now().UTC().Add(opts.Delay)
	} else if opts.Priority > 0 {
		// If no custom delay and priority is set, use the standard backoff
		backoffDelay := time.Duration(o.attempts * int(workqueue.BackoffPeriod))
		if backoffDelay > workqueue.MaximumBackoffPeriod {
			backoffDelay = workqueue.MaximumBackoffPeriod
		}
		opts.NotBefore = time.Now().UTC().Add(backoffDelay)
	}

	o.wq.rw.Lock()
	defer o.wq.rw.Unlock()
	if qi, ok := o.wq.queue[o.key]; !ok {
		o.wq.queue[o.key] = queueItem{
			Options:  opts,
			attempts: o.attempts,
			queued:   time.Now().UTC(),
		}
	} else {
		if qi.Priority < opts.Priority {
			// Raise the priority of the queued item.
			qi.Priority = opts.Priority
		}
		if opts.NotBefore.After(qi.NotBefore) {
			// Update the NotBefore time if the new one is later.
			qi.NotBefore = opts.NotBefore
		}
		o.wq.queue[o.key] = qi
	}

	delete(o.wq.wip, o.key)
	return nil
}

// GetAttempts implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) GetAttempts() int {
	return o.attempts
}

// Deadletter implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) Deadletter(_ context.Context) error {
	if o.ownerCancel != nil {
		o.ownerCancel()
	}

	// For in-memory implementation, we simply remove the task without requeueing it
	o.wq.rw.Lock()
	defer o.wq.rw.Unlock()
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
	workqueue.Options

	attempts int

	wq  *wq
	key string
}

var _ workqueue.QueuedKey = (*queuedKey)(nil)

// Name implements workqueue.Key.
func (q *queuedKey) Name() string {
	return q.key
}

// Priority implements workqueue.Key.
func (q *queuedKey) Priority() int64 {
	return q.Options.Priority
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
		Options:     q.Options,
		wq:          q.wq,
		key:         q.key,
		attempts:    q.attempts + 1,
		ownerCtx:    ctx,
		ownerCancel: cancel,
	}, nil
}
