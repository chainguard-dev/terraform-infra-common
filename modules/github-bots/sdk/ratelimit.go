/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
)

var (
	// secondaryRateLimitTriggered tracks when secondary rate limits are detected
	secondaryRateLimitTriggered = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_secondary_rate_limit_triggered_total",
			Help: "Total number of times GitHub secondary rate limit was triggered",
		},
		[]string{"status_code", "reason"},
	)

	// secondaryRateLimitWaitSeconds tracks duration of rate limit pauses
	secondaryRateLimitWaitSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "github_secondary_rate_limit_wait_seconds",
			Help:    "Duration of secondary rate limit pauses in seconds",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		},
		[]string{"reason"},
	)

	// secondaryRateLimitRetries tracks automatic retries after rate limits
	secondaryRateLimitRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_secondary_rate_limit_retries_total",
			Help: "Total number of automatic retries after secondary rate limit",
		},
		[]string{"outcome"},
	)

	// secondaryRateLimitPausedRequests tracks current number of paused requests
	secondaryRateLimitPausedRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "github_secondary_rate_limit_paused_requests",
			Help: "Current number of requests paused due to secondary rate limit",
		},
	)

	// secondaryRateLimitHeaderErrors tracks header parsing failures
	secondaryRateLimitHeaderErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_secondary_rate_limit_header_errors_total",
			Help: "Total number of errors parsing rate limit headers",
		},
		[]string{"header"},
	)
)

// SecondaryRateLimitWaiter
type SecondaryRateLimitWaiter struct {
	base              http.RoundTripper
	limiter           *limiter
	defaultRetryAfter time.Duration
}

func NewSecondaryRateLimitWaiterClient(base http.RoundTripper) *http.Client {
	if base == nil {
		base = http.DefaultTransport
	}

	return &http.Client{
		Transport: &SecondaryRateLimitWaiter{
			base: base,
			limiter: &limiter{
				base: rate.NewLimiter(rate.Inf, 100),
				mu:   sync.Mutex{},
			},
			defaultRetryAfter: 1 * time.Minute,
		},
	}
}

// maxRateLimitRetries caps how many times a single request is re-sent
// after rate-limit pauses. Retrying used to recurse without a bound, so a
// response that never stopped classifying as a rate limit turned one API
// call into an infinite pause-and-retry loop that only ended when the
// caller's deadline killed it. Genuine rate limits clear within a few
// waits; anything still limited after that is returned to the caller.
const maxRateLimitRetries = 3

func (w *SecondaryRateLimitWaiter) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	for attempt := 0; ; attempt++ {
		if err := w.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		resp, err := w.base.RoundTrip(req)
		if err != nil {
			if attempt > 0 {
				secondaryRateLimitRetries.WithLabelValues("error").Inc()
			}
			return resp, err
		}

		if !w.processLimit(ctx, resp) {
			if attempt > 0 {
				secondaryRateLimitRetries.WithLabelValues("ok").Inc()
			}
			return resp, nil
		}

		if attempt >= maxRateLimitRetries {
			secondaryRateLimitRetries.WithLabelValues("exhausted").Inc()
			clog.WarnContextf(ctx, "still rate limited after %d retries, returning the rate-limited response", maxRateLimitRetries)
			return resp, nil
		}

		// The rate-limited response is discarded before re-sending; drain
		// and close it so the underlying connection can be reused.
		if resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}
}

// processLimit reports whether resp is a rate-limit response, pausing the
// limiter accordingly when it is. Only responses carrying an actual
// rate-limit signal count: a Retry-After header, an exhausted quota
// (X-Ratelimit-Remaining: 0 with a reset time), or a 429 status. A 403
// without any of those is an authorization failure, not a rate limit —
// GitHub reports permission-denied ("resource not accessible by
// integration") as 403 with a nonzero remaining quota, and pausing on
// those turns a hard failure the caller must see into an endless wait.
// https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#exceeding-the-rate-limit
func (w *SecondaryRateLimitWaiter) processLimit(ctx context.Context, resp *http.Response) bool {
	if resp.StatusCode != http.StatusForbidden &&
		resp.StatusCode != http.StatusTooManyRequests {
		return false
	}

	var (
		retryAfter     time.Duration
		reset          time.Time
		remaining      int
		remainingKnown bool
	)

	// if "retry-after" is present, set the duration to wait before our retry
	if v := resp.Header.Get(HeaderRetryAfter); v != "" {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			clog.WarnContextf(ctx, "failed to parse retry-after header: %v", err)
			secondaryRateLimitHeaderErrors.WithLabelValues("Retry-After").Inc()
		} else {
			retryAfter = time.Duration(seconds) * time.Second
		}
	}

	// if "x-ratelimit-remaining" is present, don't retry until after the time specified
	if v := resp.Header.Get(HeaderXRateLimitRemaining); v != "" {
		r, err := strconv.Atoi(v)
		if err != nil {
			clog.WarnContextf(ctx, "failed to parse x-ratelimit-remaining header: %v", err)
			secondaryRateLimitHeaderErrors.WithLabelValues("X-Ratelimit-Remaining").Inc()
		} else {
			remaining = r
			remainingKnown = true
		}
	}

	// if "x-ratelimit-reset" is present, don't retry until after the time specified
	if v := resp.Header.Get(HeaderXRateLimitReset); v != "" {
		seconds, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			clog.WarnContextf(ctx, "failed to parse x-ratelimit-reset header: %v", err)
			secondaryRateLimitHeaderErrors.WithLabelValues("X-Ratelimit-Reset").Inc()
		} else {
			reset = time.Unix(seconds, 0)
		}
	}

	statusCode := strconv.Itoa(resp.StatusCode)

	if retryAfter > 0 {
		secondaryRateLimitTriggered.WithLabelValues(statusCode, "retry_after").Inc()
		secondaryRateLimitWaitSeconds.WithLabelValues("retry_after").Observe(retryAfter.Seconds())
		w.limiter.PauseFor(retryAfter)
		return true
	}

	// A reported-exhausted quota: wait until the reset time. The header
	// must actually be present and zero — an absent header is not an
	// exhausted quota.
	if remainingKnown && remaining == 0 && !reset.IsZero() {
		retryAfter = time.Until(reset)
		secondaryRateLimitTriggered.WithLabelValues(statusCode, "remaining_zero").Inc()
		secondaryRateLimitWaitSeconds.WithLabelValues("remaining_zero").Observe(retryAfter.Seconds())
		w.limiter.PauseFor(retryAfter)
		return true
	}

	// 429 is a rate-limit signal on its own, even without usable headers.
	if resp.StatusCode == http.StatusTooManyRequests {
		secondaryRateLimitTriggered.WithLabelValues(statusCode, "default_fallback").Inc()
		secondaryRateLimitWaitSeconds.WithLabelValues("default_fallback").Observe(w.defaultRetryAfter.Seconds())
		w.limiter.PauseFor(w.defaultRetryAfter)
		return true
	}

	// A 403 with no rate-limit signal: permission or authorization
	// failure. Surface it to the caller instead of pausing.
	return false
}

// https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#checking-the-status-of-your-rate-limit
// NOTE: Use the go canonical form (capitals) for these headers, even though they are lowercase in the docs.
const (
	HeaderRetryAfter = "Retry-After"
	// The time at which the current rate limit window resets, in UTC epoch seconds
	HeaderXRateLimitReset = "X-Ratelimit-Reset"
	// The number of requests remaining in the current rate limit window
	HeaderXRateLimitRemaining = "X-Ratelimit-Remaining"
)

type limiter struct {
	base       *rate.Limiter
	mu         sync.Mutex
	pauseUntil time.Time
	pauseCh    chan struct{}
}

func (l *limiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	pauseCh := l.pauseCh
	l.mu.Unlock()

	if pauseCh != nil {
		secondaryRateLimitPausedRequests.Inc()
		defer secondaryRateLimitPausedRequests.Dec()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pauseCh:
		}
	}

	return l.base.Wait(ctx)
}

func (l *limiter) PauseFor(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	until := time.Now().Add(d)

	if until.After(l.pauseUntil) {
		l.pauseUntil = until
		if l.pauseCh != nil {
			close(l.pauseCh)
		}
		l.pauseCh = make(chan struct{})

		go func(ch chan struct{}) {
			timer := time.NewTimer(d)
			defer timer.Stop()

			<-timer.C
			l.mu.Lock()
			if ch == l.pauseCh {
				close(ch)
				l.pauseCh = nil
				l.pauseUntil = time.Time{}
			}
			l.mu.Unlock()
		}(l.pauseCh)
	}
}
