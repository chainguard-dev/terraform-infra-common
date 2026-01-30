/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package hyperqueue

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"sync/atomic"
	"testing"

	"google.golang.org/grpc"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

type mockClient struct {
	workqueue.WorkqueueServiceClient
	processCount  atomic.Int64
	keyStateCount atomic.Int64
}

func (m *mockClient) Process(_ context.Context, _ *workqueue.ProcessRequest, _ ...grpc.CallOption) (*workqueue.ProcessResponse, error) {
	m.processCount.Add(1)
	return &workqueue.ProcessResponse{}, nil
}

func (m *mockClient) GetKeyState(_ context.Context, req *workqueue.GetKeyStateRequest, _ ...grpc.CallOption) (*workqueue.KeyState, error) {
	m.keyStateCount.Add(1)
	return &workqueue.KeyState{
		Key:    req.GetKey(),
		Status: workqueue.KeyState_QUEUED,
	}, nil
}

// failingClient returns errors for all operations.
type failingClient struct {
	workqueue.WorkqueueServiceClient
	processErr  error
	keyStateErr error
}

func (f *failingClient) Process(context.Context, *workqueue.ProcessRequest, ...grpc.CallOption) (*workqueue.ProcessResponse, error) {
	return nil, f.processErr
}

func (f *failingClient) GetKeyState(context.Context, *workqueue.GetKeyStateRequest, ...grpc.CallOption) (*workqueue.KeyState, error) {
	return nil, f.keyStateErr
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		backends []workqueue.WorkqueueServiceClient
		wantErr  bool
	}{{
		name:     "no backends returns error",
		backends: nil,
		wantErr:  true,
	}, {
		name:     "empty backends returns error",
		backends: []workqueue.WorkqueueServiceClient{},
		wantErr:  true,
	}, {
		name:     "single backend succeeds",
		backends: []workqueue.WorkqueueServiceClient{&mockClient{}},
		wantErr:  false,
	}, {
		name:     "multiple backends succeeds",
		backends: []workqueue.WorkqueueServiceClient{&mockClient{}, &mockClient{}, &mockClient{}},
		wantErr:  false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.backends)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcess_routesToSingleBackend(t *testing.T) {
	mocks := make([]*mockClient, 3)
	backends := make([]workqueue.WorkqueueServiceClient, 3)
	for i := range 3 {
		mocks[i] = &mockClient{}
		backends[i] = mocks[i]
	}

	srv, err := New(backends)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	key := fmt.Sprintf("test-key-%d", rand.Int64())

	if _, err := srv.Process(ctx, &workqueue.ProcessRequest{Key: key}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	var totalCalls int64
	for _, m := range mocks {
		totalCalls += m.processCount.Load()
	}
	if totalCalls != 1 {
		t.Errorf("Process() called %d backends, wanted 1", totalCalls)
	}
}

func TestGetKeyState_routesToSingleBackend(t *testing.T) {
	mocks := make([]*mockClient, 3)
	backends := make([]workqueue.WorkqueueServiceClient, 3)
	for i := range 3 {
		mocks[i] = &mockClient{}
		backends[i] = mocks[i]
	}

	srv, err := New(backends)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	key := fmt.Sprintf("test-key-%d", rand.Int64())

	state, err := srv.GetKeyState(ctx, &workqueue.GetKeyStateRequest{Key: key})
	if err != nil {
		t.Fatalf("GetKeyState() error = %v", err)
	}

	if state.GetKey() != key {
		t.Errorf("GetKeyState() key: got = %q, wanted = %q", state.GetKey(), key)
	}

	var totalCalls int64
	for _, m := range mocks {
		totalCalls += m.keyStateCount.Load()
	}
	if totalCalls != 1 {
		t.Errorf("GetKeyState() called %d backends, wanted 1", totalCalls)
	}
}

func TestConsistentRouting(t *testing.T) {
	mocks := make([]*mockClient, 5)
	backends := make([]workqueue.WorkqueueServiceClient, 5)
	for i := range 5 {
		mocks[i] = &mockClient{}
		backends[i] = mocks[i]
	}

	srv, err := New(backends)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()

	for range 100 {
		key := fmt.Sprintf("test-key-%d", rand.Int64())

		for _, m := range mocks {
			m.processCount.Store(0)
		}

		for range 2 {
			if _, err := srv.Process(ctx, &workqueue.ProcessRequest{Key: key}); err != nil {
				t.Fatalf("Process() error = %v", err)
			}
		}

		callsPerBackend := make([]int64, 0, len(mocks))
		for _, m := range mocks {
			callsPerBackend = append(callsPerBackend, m.processCount.Load())
		}

		foundTwo := false
		for _, calls := range callsPerBackend {
			if calls == 2 {
				foundTwo = true
			} else if calls != 0 {
				t.Errorf("Inconsistent routing for key %q: calls per backend = %v", key, callsPerBackend)
				break
			}
		}
		if !foundTwo {
			t.Errorf("Inconsistent routing for key %q: no backend received both calls, calls = %v", key, callsPerBackend)
		}
	}
}

func TestDistribution(t *testing.T) {
	const numShards = 4
	const numKeys = 10000
	const tolerance = 0.3

	mocks := make([]*mockClient, numShards)
	backends := make([]workqueue.WorkqueueServiceClient, numShards)
	for i := range numShards {
		mocks[i] = &mockClient{}
		backends[i] = mocks[i]
	}

	srv, err := New(backends)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	expectedPerShard := float64(numKeys) / float64(numShards)

	for range numKeys {
		key := fmt.Sprintf("key-%d", rand.Int64())
		if _, err := srv.Process(ctx, &workqueue.ProcessRequest{Key: key}); err != nil {
			t.Fatalf("Process() error = %v", err)
		}
	}

	for i, m := range mocks {
		count := float64(m.processCount.Load())
		deviation := (count - expectedPerShard) / expectedPerShard
		if deviation > tolerance || deviation < -tolerance {
			t.Errorf("shard %d: got = %d requests, wanted ~%d (deviation: %.2f%%)",
				i, int64(count), int64(expectedPerShard), deviation*100)
		}
	}
}

func TestProcess_surfacesBackendErrors(t *testing.T) {
	backendErr := errors.New("backend unavailable")
	failing := &failingClient{processErr: backendErr}

	srv, err := New([]workqueue.WorkqueueServiceClient{failing})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = srv.Process(context.Background(), &workqueue.ProcessRequest{Key: "test-key"})
	if err == nil {
		t.Fatal("Process() error = nil, wanted error")
	}
	if !errors.Is(err, backendErr) {
		t.Errorf("Process() error = %v, wanted %v", err, backendErr)
	}
}

func TestGetKeyState_surfacesBackendErrors(t *testing.T) {
	backendErr := errors.New("backend unavailable")
	failing := &failingClient{keyStateErr: backendErr}

	srv, err := New([]workqueue.WorkqueueServiceClient{failing})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = srv.GetKeyState(context.Background(), &workqueue.GetKeyStateRequest{Key: "test-key"})
	if err == nil {
		t.Fatal("GetKeyState() error = nil, wanted error")
	}
	if !errors.Is(err, backendErr) {
		t.Errorf("GetKeyState() error = %v, wanted %v", err, backendErr)
	}
}

func TestProcess_routesErrorToCorrectShard(t *testing.T) {
	// Create a mix of working and failing backends to verify errors come from the right shard
	backendErr := errors.New("shard-1-error")
	backends := []workqueue.WorkqueueServiceClient{
		&mockClient{},
		&failingClient{processErr: backendErr},
		&mockClient{},
	}

	srv, err := New(backends)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()

	// Find a key that routes to shard 1 (the failing one)
	var keyForShard1 string
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		// Use the same hash logic as shardFor
		h := fnv.New32a()
		h.Write([]byte(key))
		if int(h.Sum32())%3 == 1 {
			keyForShard1 = key
			break
		}
	}
	if keyForShard1 == "" {
		t.Fatal("could not find a key that routes to shard 1")
	}

	_, err = srv.Process(ctx, &workqueue.ProcessRequest{Key: keyForShard1})
	if err == nil {
		t.Fatal("Process() error = nil, wanted error for shard 1")
	}
	if !errors.Is(err, backendErr) {
		t.Errorf("Process() error = %v, wanted %v", err, backendErr)
	}
}

func TestGetKeyState_routesErrorToCorrectShard(t *testing.T) {
	// Create a mix of working and failing backends to verify errors come from the right shard
	backendErr := errors.New("shard-2-error")
	backends := []workqueue.WorkqueueServiceClient{
		&mockClient{},
		&mockClient{},
		&failingClient{keyStateErr: backendErr},
	}

	srv, err := New(backends)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()

	// Find a key that routes to shard 2 (the failing one)
	var keyForShard2 string
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		h := fnv.New32a()
		h.Write([]byte(key))
		if int(h.Sum32())%3 == 2 {
			keyForShard2 = key
			break
		}
	}
	if keyForShard2 == "" {
		t.Fatal("could not find a key that routes to shard 2")
	}

	_, err = srv.GetKeyState(ctx, &workqueue.GetKeyStateRequest{Key: keyForShard2})
	if err == nil {
		t.Fatal("GetKeyState() error = nil, wanted error for shard 2")
	}
	if !errors.Is(err, backendErr) {
		t.Errorf("GetKeyState() error = %v, wanted %v", err, backendErr)
	}
}
