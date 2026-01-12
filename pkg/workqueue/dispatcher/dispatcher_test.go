/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- Mocks ---

type mockKey struct {
	name     string
	orphaned bool
	startErr error
	attempts int
	requeue  int
	dead     int
	complete int
	mu       sync.Mutex
}

// Implement Priority() to satisfy workqueue.QueuedKey.
func (m *mockKey) Priority() int64 {
	return 0
}

func (m *mockKey) Name() string     { return m.name }
func (m *mockKey) IsOrphaned() bool { return m.orphaned }
func (m *mockKey) Start(context.Context) (workqueue.OwnedInProgressKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.startErr != nil {
		return nil, m.startErr
	}
	return &mockInProgressKey{mockKey: m}, nil
}
func (m *mockKey) Requeue(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requeue++
	return nil
}

func (m *mockKey) RequeueWithOptions(context.Context, workqueue.Options) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requeue++
	return nil
}

type mockInProgressKey struct {
	*mockKey
}

// Ensure mockInProgressKey implements workqueue.OwnedInProgressKey.
var _ workqueue.OwnedInProgressKey = (*mockInProgressKey)(nil)

func (m *mockInProgressKey) Context() context.Context { return context.Background() }
func (m *mockInProgressKey) Name() string             { return m.name }
func (m *mockInProgressKey) Priority() int64          { return 0 }
func (m *mockInProgressKey) GetAttempts() int         { return m.attempts }
func (m *mockInProgressKey) Complete(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.complete++
	return nil
}
func (m *mockInProgressKey) Deadletter(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dead++
	return nil
}

type mockQueue struct {
	wip  []workqueue.ObservedInProgressKey
	next []workqueue.QueuedKey
	err  error
}

func (m *mockQueue) Enumerate(context.Context) ([]workqueue.ObservedInProgressKey, []workqueue.QueuedKey, []workqueue.DeadLetteredKey, error) {
	return m.wip, m.next, nil, m.err
}

func (m *mockQueue) Queue(_ context.Context, key string, _ workqueue.Options) error {
	m.next = append(m.next, &mockKey{name: key})
	return nil
}

func (m *mockQueue) Get(_ context.Context, key string) (*workqueue.KeyState, error) {
	return nil, status.Errorf(codes.NotFound, "key %q not found", key)
}

// --- Tests ---

func TestHandleAsync_EnumerateError(t *testing.T) {
	q := &mockQueue{err: errors.New("fail")}
	future := HandleAsync(context.Background(), q, 1, 0, func(context.Context, string, workqueue.Options) error { return nil }, 0)
	if err := future(); err == nil || err.Error() != "enumerate() = fail" {
		t.Errorf("expected enumerate error, got %v", err)
	}
}

func TestHandleAsync_OrphanedWorkIsRequeued(t *testing.T) {
	orphan := &mockKey{name: "orphan", orphaned: true}
	q := &mockQueue{wip: []workqueue.ObservedInProgressKey{&mockInProgressKey{mockKey: orphan}}}
	called := false
	future := HandleAsync(context.Background(), q, 1, 0, func(context.Context, string, workqueue.Options) error {
		called = true
		return nil
	}, 0)
	if err := future(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orphan.requeue != 1 {
		t.Errorf("expected orphaned key to be requeued")
	}
	if called {
		t.Errorf("callback should not be called for orphaned key")
	}
}

func TestHandleAsync_NoOpenSlots(t *testing.T) {
	active := &mockKey{name: "active"}
	q := &mockQueue{
		wip:  []workqueue.ObservedInProgressKey{active},
		next: []workqueue.QueuedKey{&mockKey{name: "next"}},
	}
	called := false
	future := HandleAsync(context.Background(), q, 1, 0, func(context.Context, string, workqueue.Options) error {
		called = true
		return nil
	}, 0)
	if err := future(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Errorf("callback should not be called when no open slots")
	}
}

func TestHandleAsync_LaunchesNewWork(t *testing.T) {
	next := &mockKey{name: "next"}
	q := &mockQueue{next: []workqueue.QueuedKey{next}}
	var called bool
	future := HandleAsync(context.Background(), q, 1, 0, func(_ context.Context, key string, _ workqueue.Options) error {
		called = true
		if key != "next" {
			t.Errorf("expected key 'next', got %q", key)
		}
		return nil
	}, 0)
	if err := future(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Errorf("callback was not called")
	}
	if next.complete != 1 {
		t.Errorf("expected Complete to be called")
	}
}

func TestHandleAsync_CallbackFails_Requeue(t *testing.T) {
	next := &mockKey{name: "fail"}
	q := &mockQueue{next: []workqueue.QueuedKey{next}}
	future := HandleAsync(context.Background(), q, 1, 0, func(context.Context, string, workqueue.Options) error {
		return errors.New("fail")
	}, 0)
	if err := future(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.requeue != 1 {
		t.Errorf("expected Requeue to be called")
	}
}

func TestHandleAsync_CallbackFails_DeadletterOnMaxRetry(t *testing.T) {
	next := &mockKey{name: "fail", attempts: 3}
	q := &mockQueue{next: []workqueue.QueuedKey{next}}
	maxRetry := 3
	future := HandleAsync(context.Background(), q, 1, 0, func(context.Context, string, workqueue.Options) error {
		return errors.New("fail")
	}, maxRetry)
	if err := future(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.dead != 1 {
		t.Errorf("expected Deadletter to be called")
	}
}

func TestHandleAsync_CallbackFails_NonRetriable(t *testing.T) {
	next := &mockKey{name: "fail"}
	q := &mockQueue{next: []workqueue.QueuedKey{next}}
	nonRetriable := workqueue.NonRetriableError(errors.New("non-retriable"), "no retry")
	future := HandleAsync(context.Background(), q, 1, 0, func(context.Context, string, workqueue.Options) error {
		return nonRetriable
	}, 0)
	if err := future(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.complete != 1 {
		t.Errorf("expected Complete to be called for non-retriable error")
	}
}

func TestHandleAsync_RespectsBatchSize(t *testing.T) {
	keys := []*mockKey{
		{name: "k1"},
		{name: "k2"},
		{name: "k3"},
	}

	next := make([]workqueue.QueuedKey, len(keys))
	for i := range keys {
		next[i] = keys[i]
	}

	q := &mockQueue{next: next}

	future := HandleAsync(context.Background(), q, 3, 2, func(context.Context, string, workqueue.Options) error {
		return nil
	}, 0)

	if err := future(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var launched int
	for _, k := range keys {
		launched += k.complete
	}

	if launched != 2 {
		t.Fatalf("expected to launch 2 keys, got %d", launched)
	}
}
