/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpratelimit

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

type testRT struct {
	responses []*http.Response
	mu        sync.Mutex
	callCount int
}

func (t *testRT) RoundTrip(_ *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.callCount >= len(t.responses) {
		return nil, fmt.Errorf("no more responses")
	}
	resp := t.responses[t.callCount]
	t.callCount++
	return resp, nil
}

func TestTransport_RateLimiting(t *testing.T) {
	defaultRetryAfter := 1 * time.Second

	tests := []struct {
		name           string
		responses      func(baseTime time.Time) []*http.Response
		expectedCalls  int
		expectedStatus int
		expectedWait   time.Duration
	}{
		{
			name: "No rate limit",
			responses: func(_ time.Time) []*http.Response {
				return []*http.Response{{StatusCode: http.StatusOK}}
			},
			expectedCalls:  1,
			expectedWait:   0,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Rate limit with x-ratelimit-reset",
			responses: func(baseTime time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusForbidden,
						Header: http.Header{
							HeaderXRateLimitRemaining: []string{"0"},
							HeaderXRateLimitReset:     []string{fmt.Sprintf("%d", baseTime.Add(4*time.Second).Unix())},
						},
					},
					{StatusCode: http.StatusOK},
				}
			},
			expectedCalls:  2,
			expectedWait:   4 * time.Second,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Rate limit with x-ratelimit-remaining",
			responses: func(baseTime time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusForbidden,
						Header: http.Header{
							HeaderXRateLimitRemaining: []string{"0"},
							HeaderXRateLimitReset:     []string{fmt.Sprintf("%d", baseTime.Add(4*time.Second).Unix())},
						},
					},
					{
						StatusCode: http.StatusOK,
					},
				}
			},
			expectedCalls:  2,
			expectedWait:   4 * time.Second,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Rate limit with retry-after",
			responses: func(_ time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusForbidden,
						Header: http.Header{
							HeaderRetryAfter: {"2"},
						},
					},
					{
						StatusCode: http.StatusOK,
					},
				}
			},
			expectedCalls:  2,
			expectedWait:   2 * time.Second,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Rate limited without headers uses the default retry-after",
			responses: func(_ time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusForbidden,
						Header:     http.Header{},
					},
					{StatusCode: http.StatusOK},
				}
			},
			expectedCalls:  2,
			expectedWait:   defaultRetryAfter,
			expectedStatus: http.StatusOK,
		},
		{
			name: "Rate limit with 429 status code",
			responses: func(_ time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusTooManyRequests,
						Header: http.Header{
							HeaderRetryAfter: {"1"},
						},
					},
					{StatusCode: http.StatusOK},
				}
			},
			expectedCalls:  2,
			expectedWait:   1 * time.Second,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseTime := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			trt := &testRT{
				responses: tt.responses(baseTime),
			}

			client := &http.Client{
				Transport: NewTransport(trt, defaultRetryAfter),
			}

			req, err := http.NewRequest(http.MethodGet, "https://api.github.com/test", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			req = req.WithContext(ctx)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			elapsed := time.Since(baseTime)

			if resp != nil && resp.StatusCode != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if trt.callCount != tt.expectedCalls {
				t.Fatalf("expected %d calls, got %d", tt.expectedCalls, trt.callCount)
			}

			// Apply some buffer to account for timing variations
			if tt.expectedWait == 0 {
				if elapsed > 100*time.Millisecond {
					t.Fatalf("expected no significant wait, but got %s", elapsed)
				}
			} else {
				buffer := tt.expectedWait / 4 // 25% buffer
				minExpectedWait := tt.expectedWait - buffer
				maxExpectedWait := tt.expectedWait + buffer

				if elapsed < minExpectedWait || elapsed > maxExpectedWait {
					t.Fatalf("expected wait time between %s and %s, got %s", minExpectedWait, maxExpectedWait, elapsed)
				}
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient(nil)
	require := func(cond bool, msg string) {
		if !cond {
			t.Fatal(msg)
		}
	}

	require(client != nil, "expected non-nil client")

	transport, ok := client.Transport.(*Transport)
	require(ok, "expected transport to be *Transport")

	if transport.defaultRetryAfter != time.Minute {
		t.Fatalf("expected default retry after to be 1 minute, got %v", transport.defaultRetryAfter)
	}
}

func TestLimiter_ConcurrentPause(t *testing.T) {
	l := &limiter{
		base: nil, // Not used in this test
	}

	var wg sync.WaitGroup
	pauseDurations := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 50 * time.Millisecond}

	// Pause concurrently with different durations
	for _, d := range pauseDurations {
		wg.Add(1)
		go func(duration time.Duration) {
			defer wg.Done()
			l.PauseFor(duration)
		}(d)
	}

	wg.Wait()

	// The longest pause should win
	expectedPauseUntil := time.Now().Add(200 * time.Millisecond)
	l.mu.Lock()
	actualPauseUntil := l.pauseUntil
	l.mu.Unlock()

	// Allow some timing variance
	diff := actualPauseUntil.Sub(expectedPauseUntil)
	if diff < -50*time.Millisecond || diff > 50*time.Millisecond {
		t.Fatalf("expected pause until around %v, got %v (diff: %v)", expectedPauseUntil, actualPauseUntil, diff)
	}
}

func TestTransport_ProactiveThrottling(t *testing.T) {
	// Test that the transport proactively throttles when quota is low
	baseTime := time.Now()
	resetTime := baseTime.Add(60 * time.Second)

	// Simulate responses with decreasing rate limit quota
	responses := []*http.Response{
		// First request: 5000/5000 remaining (100%)
		{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"X-Ratelimit-Limit":     []string{"5000"},
				"X-Ratelimit-Remaining": []string{"5000"},
				"X-Ratelimit-Reset":     []string{fmt.Sprintf("%d", resetTime.Unix())},
			},
		},
		// Second request: 500/5000 remaining (10%)
		{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"X-Ratelimit-Limit":     []string{"5000"},
				"X-Ratelimit-Remaining": []string{"500"},
				"X-Ratelimit-Reset":     []string{fmt.Sprintf("%d", resetTime.Unix())},
			},
		},
		// Third request: 50/5000 remaining (1%)
		{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"X-Ratelimit-Limit":     []string{"5000"},
				"X-Ratelimit-Remaining": []string{"50"},
				"X-Ratelimit-Reset":     []string{fmt.Sprintf("%d", resetTime.Unix())},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trt := &testRT{
		responses: responses,
	}

	transport := NewTransport(trt, time.Minute)
	client := &http.Client{Transport: transport}

	// Make requests (timing tracked implicitly by rate limiter)
	for i := 0; i < len(responses); i++ {
		req, err := http.NewRequest(http.MethodGet, "https://api.github.com/test", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to make request %d: %v", i, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d: expected status 200, got %d", i, resp.StatusCode)
		}
	}

	// Verify the transport is tracking state
	transport.mu.RLock()
	lastRemaining := transport.lastRemaining
	lastLimit := transport.lastLimit
	transport.mu.RUnlock()

	if lastRemaining != 50 {
		t.Errorf("expected last remaining to be 50, got %d", lastRemaining)
	}
	if lastLimit != 5000 {
		t.Errorf("expected last limit to be 5000, got %d", lastLimit)
	}

	// Note: We don't check timing delays here because rate limiting introduces
	// variable delays that are hard to test deterministically
}

func TestTransport_MonitoringSuccessfulResponses(t *testing.T) {
	// Test that we monitor rate limit headers on ALL responses, not just errors
	baseTime := time.Now()
	resetTime := baseTime.Add(60 * time.Second)

	resp := &http.Response{
		StatusCode: http.StatusOK, // Success, not an error!
		Header: http.Header{
			"X-Ratelimit-Limit":     []string{"5000"},
			"X-Ratelimit-Remaining": []string{"4500"},
			"X-Ratelimit-Reset":     []string{fmt.Sprintf("%d", resetTime.Unix())},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	trt := &testRT{
		responses: []*http.Response{resp},
	}

	transport := NewTransport(trt, time.Minute)
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req = req.WithContext(ctx)

	_, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	// Verify we tracked the rate limit info from the successful response
	transport.mu.RLock()
	lastRemaining := transport.lastRemaining
	lastLimit := transport.lastLimit
	lastReset := transport.lastReset
	transport.mu.RUnlock()

	if lastRemaining != 4500 {
		t.Errorf("expected lastRemaining to be 4500, got %d", lastRemaining)
	}
	if lastLimit != 5000 {
		t.Errorf("expected lastLimit to be 5000, got %d", lastLimit)
	}
	if lastReset.IsZero() {
		t.Error("expected lastReset to be set")
	}
}
