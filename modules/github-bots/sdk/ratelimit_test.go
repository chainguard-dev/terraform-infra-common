package sdk

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpratelimit"
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
				Transport: httpratelimit.NewTransport(
					trt,
					httpratelimit.WithDefaultRetryAfter(defaultRetryAfter),
					httpratelimit.WithMaxRequestsPerSecond(100), // High rate for tests
				),
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
