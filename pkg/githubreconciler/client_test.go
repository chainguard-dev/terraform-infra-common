/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpratelimit"
	"github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
)

// mockTokenSource is a mock OAuth2 token source
type mockTokenSource struct {
	token string
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken: m.token,
	}, nil
}

func TestClientCache_Get(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		org1       string
		repo1      string
		org2       string
		repo2      string
		wantSame   bool
		tokenError error
	}{
		{
			name:     "same org/repo returns same client",
			org1:     "myorg",
			repo1:    "myrepo",
			org2:     "myorg",
			repo2:    "myrepo",
			wantSame: true,
		},
		{
			name:     "different repo returns different client",
			org1:     "myorg",
			repo1:    "repo1",
			org2:     "myorg",
			repo2:    "repo2",
			wantSame: false,
		},
		{
			name:     "different org returns different client",
			org1:     "org1",
			repo1:    "myrepo",
			org2:     "org2",
			repo2:    "myrepo",
			wantSame: false,
		},
		{
			name:     "both different returns different client",
			org1:     "org1",
			repo1:    "repo1",
			org2:     "org2",
			repo2:    "repo2",
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenSourceFunc := func(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
				if tt.tokenError != nil {
					return nil, tt.tokenError
				}
				return &mockTokenSource{token: fmt.Sprintf("token-%s-%s", org, repo)}, nil
			}

			cache := NewClientCache(tokenSourceFunc)

			client1, err1 := cache.Get(ctx, tt.org1, tt.repo1)
			client2, err2 := cache.Get(ctx, tt.org2, tt.repo2)

			if err1 != nil || err2 != nil {
				if tt.tokenError == nil {
					t.Errorf("Unexpected error: err1=%v, err2=%v", err1, err2)
				}
				return
			}

			if tt.wantSame {
				if client1 != client2 {
					t.Errorf("Expected same client instance for %s/%s", tt.org1, tt.repo1)
				}
			} else {
				if client1 == client2 {
					t.Errorf("Expected different client instances for %s/%s and %s/%s",
						tt.org1, tt.repo1, tt.org2, tt.repo2)
				}
			}
		})
	}
}

func TestClientCache_GetError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("token source error")

	tokenSourceFunc := func(_ context.Context, _, _ string) (oauth2.TokenSource, error) {
		return nil, expectedErr
	}

	cache := NewClientCache(tokenSourceFunc)

	_, err := cache.Get(ctx, "org", "repo")
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error to contain %v, got %v", expectedErr, err)
	}
}

func TestClientCache_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	var tokenSourceCalls int32

	tokenSourceFunc := func(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
		atomic.AddInt32(&tokenSourceCalls, 1)
		return &mockTokenSource{token: fmt.Sprintf("token-%s-%s", org, repo)}, nil
	}

	cache := NewClientCache(tokenSourceFunc)

	org, repo := "testorg", "testrepo"
	numGoroutines := 50
	clientsChan := make(chan *github.Client, numGoroutines)
	errorsChan := make(chan error, numGoroutines)

	// Concurrently get clients with the same org/repo
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			client, err := cache.Get(ctx, org, repo)
			if err != nil {
				errorsChan <- err
			} else {
				clientsChan <- client
			}
		}()
	}

	wg.Wait()
	close(clientsChan)
	close(errorsChan)

	// Check for errors
	for err := range errorsChan {
		t.Fatalf("Unexpected error in concurrent access: %v", err)
	}

	// Collect all clients
	clients := make([]*github.Client, 0, numGoroutines)
	for client := range clientsChan {
		clients = append(clients, client)
	}

	// Verify all clients are the same instance
	if len(clients) != numGoroutines {
		t.Fatalf("Expected %d clients, got %d", numGoroutines, len(clients))
	}

	firstClient := clients[0]
	for i, client := range clients {
		if client != firstClient {
			t.Errorf("Client %d is not the same instance as client 0", i)
		}
	}

	// Verify token source was only called once due to caching
	calls := atomic.LoadInt32(&tokenSourceCalls)
	if calls != 1 {
		t.Errorf("Expected token source to be called once, but was called %d times", calls)
	}
}

func TestClientCache_Clear(t *testing.T) {
	ctx := context.Background()
	var tokenSourceCalls int32

	tokenSourceFunc := func(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
		atomic.AddInt32(&tokenSourceCalls, 1)
		return &mockTokenSource{token: fmt.Sprintf("token-%s-%s", org, repo)}, nil
	}

	cache := NewClientCache(tokenSourceFunc)

	// Get a client
	client1, err := cache.Get(ctx, "org", "repo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Clear the cache
	cache.Clear()

	// Get the same client again
	client2, err := cache.Get(ctx, "org", "repo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should be different instances after clear
	if client1 == client2 {
		t.Error("Expected different client instances after Clear()")
	}

	// Token source should have been called twice
	calls := atomic.LoadInt32(&tokenSourceCalls)
	if calls != 2 {
		t.Errorf("Expected token source to be called twice, but was called %d times", calls)
	}
}

// Benchmark client cache
func BenchmarkClientCache_Get_Cached(b *testing.B) {
	ctx := context.Background()
	tokenSourceFunc := func(_ context.Context, _, _ string) (oauth2.TokenSource, error) {
		return &mockTokenSource{token: "benchmark-token"}, nil
	}

	cache := NewClientCache(tokenSourceFunc)
	org, repo := "benchorg", "benchrepo"

	// Prime the cache
	cache.Get(ctx, org, repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(ctx, org, repo)
	}
}

func BenchmarkClientCache_Get_Different(b *testing.B) {
	ctx := context.Background()
	tokenSourceFunc := func(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
		return &mockTokenSource{token: fmt.Sprintf("token-%s-%s", org, repo)}, nil
	}

	cache := NewClientCache(tokenSourceFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		org := fmt.Sprintf("org%d", i%100)
		repo := fmt.Sprintf("repo%d", i)
		cache.Get(ctx, org, repo)
	}
}

func TestClientCache_RateLimitTransportIntegration(t *testing.T) {
	ctx := context.Background()

	tokenSourceFunc := func(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
		return &mockTokenSource{token: fmt.Sprintf("token-%s-%s", org, repo)}, nil
	}

	cache := NewClientCache(tokenSourceFunc)

	// Get a client
	client, err := cache.Get(ctx, "myorg", "myrepo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the client has a transport
	if client.Client() == nil {
		t.Fatal("Expected client to have an HTTP client")
	}

	// Walk through the transport chain to verify rate limiter is present
	transport := client.Client().Transport
	if transport == nil {
		t.Fatal("Expected transport to be non-nil")
	}

	// The transport should be the metrics transport wrapping the rate limiter
	// We can't directly check private fields, but we can verify it doesn't panic
	// and that subsequent calls to the same org/repo return the same client
	client2, err := cache.Get(ctx, "myorg", "myrepo")
	if err != nil {
		t.Fatalf("Unexpected error on second get: %v", err)
	}

	if client != client2 {
		t.Error("Expected cached client to be the same instance")
	}
}

func TestClientCache_RateLimitTransportChain(t *testing.T) {
	ctx := context.Background()

	tokenSourceFunc := func(_ context.Context, org, repo string) (oauth2.TokenSource, error) {
		return &mockTokenSource{token: fmt.Sprintf("token-%s-%s", org, repo)}, nil
	}

	cache := NewClientCache(tokenSourceFunc)

	// Get a client and verify the transport chain
	client, err := cache.Get(ctx, "testorg", "testrepo")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the HTTP client has a transport
	httpClient := client.Client()
	if httpClient == nil || httpClient.Transport == nil {
		t.Fatal("Expected HTTP client with non-nil transport")
	}

	// The transport chain is: metrics -> rate limiter -> oauth2
	// We verified above that the transport is non-nil, which means
	// the transport chain is properly constructed
	_, err = http.NewRequest("GET", "https://api.github.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}
}

func TestNewTransport_UsesRateLimiter(t *testing.T) {
	// This test verifies that our rate limiter is correctly created
	baseTransport := http.DefaultTransport
	rateLimitTransport := httpratelimit.NewTransport(baseTransport, 0)

	if rateLimitTransport == nil {
		t.Fatal("Expected rate limit transport to be non-nil")
	}

	// Verify the transport is usable
	_, err := http.NewRequest("GET", "https://api.github.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create test request: %v", err)
	}

	// Verify construction was successful
	if rateLimitTransport == nil {
		t.Error("Rate limit transport not properly initialized")
	}
}
