/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
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

func TestSecondaryRateLimitWaiter(t *testing.T) {
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
			name: "Rate limit with `x-ratelimit-reset`",
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
			name: "Rate limit with `x-ratelimit-remaining`",
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
			name: "Rate limit with `retry-after`",
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
			name: "429 without headers uses the default retry-after",
			responses: func(_ time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusTooManyRequests,
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
			// Regression: a 403 with no rate-limit signal is an
			// authorization failure, not a rate limit. Treating it as one
			// put a permission-denied API call into an endless
			// pause-and-retry loop in production.
			name: "403 without rate limit signals is returned immediately",
			responses: func(_ time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusForbidden,
						Header:     http.Header{},
					},
				}
			},
			expectedCalls:  1,
			expectedWait:   0,
			expectedStatus: http.StatusForbidden,
		},
		{
			// The exact shape GitHub uses for "resource not accessible by
			// integration": 403 with plenty of quota remaining.
			name: "403 with remaining quota is a permission failure, not a rate limit",
			responses: func(baseTime time.Time) []*http.Response {
				return []*http.Response{
					{
						StatusCode: http.StatusForbidden,
						Header: http.Header{
							HeaderXRateLimitRemaining: []string{"4999"},
							HeaderXRateLimitReset:     []string{fmt.Sprintf("%d", baseTime.Add(time.Hour).Unix())},
						},
					},
				}
			},
			expectedCalls:  1,
			expectedWait:   0,
			expectedStatus: http.StatusForbidden,
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
				Transport: &SecondaryRateLimitWaiter{
					base: trt,
					limiter: &limiter{
						base: rate.NewLimiter(rate.Inf, 100),
						mu:   sync.Mutex{},
					},
					defaultRetryAfter: defaultRetryAfter,
				},
			}

			req, err := http.NewRequest(http.MethodGet, "https://foobear.com", nil)
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

			// Apply some buffer to account for these bad tests and the fact that we're not mocking the clock
			if tt.expectedWait == 0 {
				if elapsed > 100*time.Millisecond {
					t.Fatalf("expected no significant wait, but got %s", elapsed)
				}
			} else {
				buffer := tt.expectedWait / 4 // 10% of expected wait
				minExpectedWait := tt.expectedWait - buffer
				maxExpectedWait := tt.expectedWait + buffer

				if elapsed < minExpectedWait || elapsed > maxExpectedWait {
					t.Fatalf("expected wait time between %s and %s, got %s", minExpectedWait, maxExpectedWait, elapsed)
				}
			}
		})
	}
}

// Test_SecondaryRateLimitWaiter_retryCap: a response that keeps
// classifying as a rate limit must stop being retried after
// maxRateLimitRetries and be returned to the caller, rather than looping
// until the caller's deadline kills it.
func Test_SecondaryRateLimitWaiter_retryCap(t *testing.T) {
	limited := func() *http.Response {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{},
		}
	}
	// More rate-limited responses than the waiter is allowed to consume.
	responses := make([]*http.Response, 0, maxRateLimitRetries+5)
	for range maxRateLimitRetries + 5 {
		responses = append(responses, limited())
	}
	trt := &testRT{responses: responses}

	client := &http.Client{
		Transport: &SecondaryRateLimitWaiter{
			base: trt,
			limiter: &limiter{
				base: rate.NewLimiter(rate.Inf, 100),
				mu:   sync.Mutex{},
			},
			defaultRetryAfter: 10 * time.Millisecond,
		},
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://foobear.com", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected the rate-limited response to be surfaced, got %d", resp.StatusCode)
	}
	if want := 1 + maxRateLimitRetries; trt.callCount != want {
		t.Fatalf("expected %d calls (initial + %d retries), got %d", want, maxRateLimitRetries, trt.callCount)
	}
}
