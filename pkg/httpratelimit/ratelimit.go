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

// Rate limiting configuration constants
const (
	// DefaultMaxRequestsPerSecond is a conservative default to prevent GitHub secondary rate limits
	// Secondary rate limits are triggered by request velocity, not quota, and don't provide warning headers.
	DefaultMaxRequestsPerSecond = 15.0

	// DefaultRetryAfter is the default wait time when rate limited but no retry-after header is provided
	DefaultRetryAfter = time.Minute

	// BurstMultiplier determines the burst capacity as a multiple of the rate limit
	// This allows some flexibility for occasional bursts while maintaining average rate
	BurstMultiplier = 2
)

// Transport wraps an http.RoundTripper to provide proactive rate limiting
// for GitHub API requests. It prevents both primary rate limits (quota-based)
// and secondary rate limits (velocity-based) by enforcing a maximum requests-per-second.
type Transport struct {
	base              http.RoundTripper
	limiter           *limiter
	defaultRetryAfter time.Duration
	maxRequestsPerSec float64
}

// TransportOption is a functional option for configuring Transport.
type TransportOption func(*Transport)

// WithDefaultRetryAfter sets the default retry duration when rate limited but no retry-after header is provided.
func WithDefaultRetryAfter(d time.Duration) TransportOption {
	return func(t *Transport) {
		t.defaultRetryAfter = d
	}
}

// WithMaxRequestsPerSecond sets the maximum requests per second to prevent secondary rate limits.
// Secondary rate limits are triggered by request velocity, not total quota, and don't provide warning headers.
func WithMaxRequestsPerSecond(rps float64) TransportOption {
	return func(t *Transport) {
		t.maxRequestsPerSec = rps
	}
}

// NewTransport creates a new rate limiting transport wrapper.
//
// Parameters:
//   - base: The underlying http.RoundTripper to wrap (uses http.DefaultTransport if nil)
//   - opts: Optional configuration options (WithDefaultRetryAfter, WithMaxRequestsPerSecond)
//
// By default, uses DefaultRetryAfter (1 minute) and DefaultMaxRequestsPerSecond (15 RPS).
// The maxRequestsPerSec limit prevents GitHub's secondary rate limits which are triggered
// by request velocity, not total quota.
func NewTransport(base http.RoundTripper, opts ...TransportOption) *Transport {
	if base == nil {
		base = http.DefaultTransport
	}

	t := &Transport{
		base:              base,
		defaultRetryAfter: DefaultRetryAfter,
		maxRequestsPerSec: DefaultMaxRequestsPerSecond,
	}

	// Apply options
	for _, opt := range opts {
		opt(t)
	}

	// Create rate limiter with specified RPS
	// Burst allows some flexibility for occasional bursts while maintaining average rate
	burst := int(t.maxRequestsPerSec * BurstMultiplier)
	if burst < 1 {
		burst = 1
	}

	t.limiter = &limiter{
		base: rate.NewLimiter(rate.Limit(t.maxRequestsPerSec), burst),
		mu:   sync.Mutex{},
	}

	return t
}

// NewClient creates a new HTTP client with rate limiting enabled using default settings.
// This is a convenience function that wraps the given base transport with default constants.
func NewClient(base http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: NewTransport(base),
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
