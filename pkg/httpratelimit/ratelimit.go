/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpratelimit

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/chainguard-dev/clog"
	"golang.org/x/time/rate"
)

// GitHub rate limit header names
// https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api#checking-the-status-of-your-rate-limit
// NOTE: Use the Go canonical form (capitals) for these headers, even though they are lowercase in the docs.
const (
	// HeaderRetryAfter indicates how many seconds to wait before retrying
	HeaderRetryAfter = "Retry-After"
	// HeaderXRateLimitReset is the time at which the current rate limit window resets, in UTC epoch seconds
	HeaderXRateLimitReset = "X-Ratelimit-Reset"
	// HeaderXRateLimitRemaining is the number of requests remaining in the current rate limit window
	HeaderXRateLimitRemaining = "X-Ratelimit-Remaining"
)

// Transport wraps an http.RoundTripper to provide proactive rate limiting
// for GitHub API requests. It monitors rate limit headers and automatically throttles
// requests to prevent hitting rate limits.
type Transport struct {
	base              http.RoundTripper
	limiter           *limiter
	defaultRetryAfter time.Duration
}

// NewTransport creates a new rate limiting transport wrapper.
// The defaultRetryAfter specifies how long to wait when rate limited but no
// retry-after header is provided by GitHub (defaults to 1 minute).
func NewTransport(base http.RoundTripper, defaultRetryAfter time.Duration) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}
	if defaultRetryAfter == 0 {
		defaultRetryAfter = time.Minute
	}

	return &Transport{
		base: base,
		limiter: &limiter{
			base: rate.NewLimiter(rate.Inf, 100),
			mu:   sync.Mutex{},
		},
		defaultRetryAfter: defaultRetryAfter,
	}
}

// NewClient creates a new HTTP client with rate limiting enabled.
// This is a convenience function that wraps the given base transport.
func NewClient(base http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: NewTransport(base, time.Minute),
	}
}

// RoundTrip implements http.RoundTripper and adds rate limiting logic.
func (rt *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Wait if we're currently paused due to rate limiting
	if err := rt.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Execute the request
	resp, err := rt.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Check if we hit a rate limit and need to retry
	if rt.processRateLimit(ctx, resp) {
		// Recursively retry the request after waiting
		return rt.RoundTrip(req)
	}

	return resp, nil
}

// processRateLimit checks if the response indicates rate limiting and pauses future requests.
// Returns true if the request should be retried after the pause.
//
// GitHub rate limit documentation:
// https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api#exceeding-the-rate-limit
func (rt *Transport) processRateLimit(ctx context.Context, resp *http.Response) bool {
	log := clog.FromContext(ctx)

	// Check for rate limit status codes
	if resp.StatusCode != http.StatusForbidden &&
		resp.StatusCode != http.StatusTooManyRequests {
		return false
	}

	var (
		retryAfter time.Duration
		reset      time.Time
		remaining  int
	)

	// Parse retry-after header (in seconds)
	if v := resp.Header.Get(HeaderRetryAfter); v != "" {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			log.Warnf("Failed to parse retry-after header: %v", err)
		} else {
			retryAfter = time.Duration(seconds) * time.Second
		}
	}

	// Parse x-ratelimit-remaining header
	if v := resp.Header.Get(HeaderXRateLimitRemaining); v != "" {
		r, err := strconv.Atoi(v)
		if err != nil {
			log.Warnf("Failed to parse x-ratelimit-remaining header: %v", err)
		} else {
			remaining = r
		}
	}

	// Parse x-ratelimit-reset header (Unix timestamp)
	if v := resp.Header.Get(HeaderXRateLimitReset); v != "" {
		seconds, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Warnf("Failed to parse x-ratelimit-reset header: %v", err)
		} else {
			reset = time.Unix(seconds, 0)
		}
	}

	// Determine pause duration based on available headers
	if retryAfter > 0 {
		log.With("retry_after", retryAfter).
			Warn("GitHub rate limit hit, pausing requests")
		rt.limiter.PauseFor(retryAfter)
		return true
	}

	// If remaining is 0 and reset time is provided, wait until reset
	if remaining == 0 && !reset.IsZero() {
		retryAfter = time.Until(reset)
		if retryAfter > 0 {
			log.With("reset_at", reset, "retry_after", retryAfter).
				Warn("GitHub rate limit exhausted, pausing until reset")
			rt.limiter.PauseFor(retryAfter)
			return true
		}
	}

	// Default fallback if we got a rate limit status but no helpful headers
	log.With("retry_after", rt.defaultRetryAfter).
		Warn("GitHub rate limit hit (no headers), using default pause")
	rt.limiter.PauseFor(rt.defaultRetryAfter)
	return true
}

// limiter provides a pausable rate limiter that can temporarily block all requests.
type limiter struct {
	base       *rate.Limiter
	mu         sync.Mutex
	pauseUntil time.Time
	pauseCh    chan struct{}
}

// Wait blocks until the limiter allows a request to proceed.
// It respects both the underlying rate limiter and any active pause.
func (l *limiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	pauseCh := l.pauseCh
	l.mu.Unlock()

	// If we're paused, wait for the pause to end
	if pauseCh != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pauseCh:
		}
	}

	// Wait for rate limiter to allow the request
	return l.base.Wait(ctx)
}

// PauseFor pauses all requests for the specified duration.
// If already paused, extends the pause only if the new duration is longer.
func (l *limiter) PauseFor(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	until := time.Now().Add(d)

	// Only update if this extends the current pause
	if until.After(l.pauseUntil) {
		l.pauseUntil = until

		// Close existing pause channel if any
		if l.pauseCh != nil {
			close(l.pauseCh)
		}
		l.pauseCh = make(chan struct{})

		// Start goroutine to end the pause after duration
		go func(ch chan struct{}) {
			timer := time.NewTimer(d)
			defer timer.Stop()

			<-timer.C

			l.mu.Lock()
			// Only clear if this is still the active pause channel
			if ch == l.pauseCh {
				close(ch)
				l.pauseCh = nil
				l.pauseUntil = time.Time{}
			}
			l.mu.Unlock()
		}(l.pauseCh)
	}
}
