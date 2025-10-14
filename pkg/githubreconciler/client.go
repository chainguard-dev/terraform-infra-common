/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/google/go-github/v75/github"
	"golang.org/x/oauth2"
)

// TokenSourceFunc is a function that creates an OAuth2 token source for a given org/repo.
type TokenSourceFunc func(ctx context.Context, org, repo string) (oauth2.TokenSource, error)

// ClientCache manages GitHub clients for multiple org/repo combinations.
type ClientCache struct {
	tokenSourceFunc TokenSourceFunc
	mu              sync.RWMutex
	clients         map[string]*github.Client
}

// NewClientCache creates a new client cache with the provided token source function.
func NewClientCache(tokenSourceFunc TokenSourceFunc) *ClientCache {
	return &ClientCache{
		tokenSourceFunc: tokenSourceFunc,
		clients:         make(map[string]*github.Client),
	}
}

// getKey returns the cache key for an org/repo combination.
func (cc *ClientCache) getKey(org, repo string) string {
	return fmt.Sprintf("%s/%s", org, repo)
}

// Get returns a GitHub client for the given org/repo, creating one if needed.
func (cc *ClientCache) Get(ctx context.Context, org, repo string) (*github.Client, error) {
	key := cc.getKey(org, repo)

	// Try to get existing client
	cc.mu.RLock()
	client, exists := cc.clients[key]
	cc.mu.RUnlock()

	if exists {
		clog.FromContext(ctx).With(
			"org", org,
			"repo", repo,
		).Debug("Using cached GitHub client")
		return client, nil
	}

	// Create new client
	cc.mu.Lock()
	defer cc.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := cc.clients[key]; exists {
		return client, nil
	}

	// Create token source for this org/repo
	// Use context.Background() instead of ctx because the token source will be cached
	// and reused across multiple requests. If we use the request context, it could be
	// cancelled while the token source is still in the cache, causing future token
	// refresh attempts to fail with "context cancelled". Alternatively, using a context
	// without cancel here will cause future requests to inherit a leaky/dirty context
	// based on previous requests.
	tokenSource, err := cc.tokenSourceFunc(context.Background(), org, repo)
	if err != nil {
		return nil, fmt.Errorf("creating token source: %w", err)
	}

	// Create OAuth2 client with the token source
	oauthClient := oauth2.NewClient(ctx, tokenSource)

	// Wrap the transport with metrics instrumentation for GitHub API monitoring
	httpClient := &http.Client{
		Transport: httpmetrics.WrapTransport(oauthClient.Transport),
	}

	client = github.NewClient(httpClient)

	// Cache the client
	cc.clients[key] = client

	clog.FromContext(ctx).With(
		"org", org,
		"repo", repo,
	).Info("Created new GitHub client for repository")

	return client, nil
}

// Clear removes all cached clients.
func (cc *ClientCache) Clear() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.clients = make(map[string]*github.Client)
}
