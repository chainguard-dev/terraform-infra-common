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
				Transport: NewTransport(
					trt,
					WithDefaultRetryAfter(defaultRetryAfter),
					WithMaxRequestsPerSecond(100), // High rate for tests
				),
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
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	transport, ok := client.Transport.(*Transport)
	if !ok {
		t.Fatal("expected transport to be *Transport")
	}

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

func TestTransport_VelocityBasedRateLimiting(t *testing.T) {
	// Test that velocity-based rate limiting actually enforces the RPS limit
	maxRPS := 10.0 // 10 requests per second
	requestCount := 25

	// Mock transport that always returns 200 OK
	mockRT := &testRT{
		responses: make([]*http.Response, requestCount),
	}
	for i := 0; i < requestCount; i++ {
		mockRT.responses[i] = &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
		}
	}

	client := &http.Client{
		Transport: NewTransport(mockRT, WithMaxRequestsPerSecond(maxRPS)),
	}

	ctx := context.Background()
	startTime := time.Now()

	// Make requests and measure time
	for i := 0; i < requestCount; i++ {
		req, err := http.NewRequest(http.MethodGet, "https://api.github.com/test", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req = req.WithContext(ctx)

		_, err = client.Do(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
	}

	elapsed := time.Since(startTime)

	// Calculate expected minimum time based on rate limit
	// For 25 requests at 10 RPS, minimum time should be ~2.5 seconds
	// We account for burst (2x) so first ~20 requests can be fast, then throttled
	// Expected: (requestCount - burst) / maxRPS
	burst := int(maxRPS * BurstMultiplier)
	throttledRequests := requestCount - burst
	if throttledRequests < 0 {
		throttledRequests = 0
	}
	expectedMinTime := time.Duration(float64(throttledRequests)/maxRPS*1000) * time.Millisecond

	// Allow 20% tolerance for timing variations
	tolerance := expectedMinTime / 5
	minAcceptable := expectedMinTime - tolerance

	if elapsed < minAcceptable {
		t.Errorf("Rate limiting not enforced: expected at least %v, got %v (tolerance: %v)",
			expectedMinTime, elapsed, tolerance)
	}

	t.Logf("Rate limiting working: %d requests took %v (expected minimum: %v)",
		requestCount, elapsed, expectedMinTime)
}
