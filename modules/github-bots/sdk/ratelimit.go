package sdk

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/chainguard-dev/clog"
	"golang.org/x/time/rate"
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

func (w *SecondaryRateLimitWaiter) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	if err := w.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := w.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if w.processLimit(ctx, resp) {
		return w.RoundTrip(req)
	}

	return resp, nil
}

// processLimit processes a response and returns a secondaryLimit if the response is a secondary limit
// https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28#exceeding-the-rate-limit
func (w *SecondaryRateLimitWaiter) processLimit(ctx context.Context, resp *http.Response) bool {
	log := clog.FromContext(ctx)

	if resp.StatusCode != http.StatusForbidden &&
		resp.StatusCode != http.StatusTooManyRequests {
		return false
	}

	var (
		retryAfter time.Duration
		reset      time.Time
		remaining  int
	)

	// if "retry-after" is present, set the duration to wait before our retry
	if v := resp.Header.Get(HeaderRetryAfter); v != "" {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			log.Warnf("failed to parse retry-after header: %v", err)
		} else {
			retryAfter = time.Duration(seconds) * time.Second
		}
	}

	// if "x-ratelimit-remaining" is present, don't retry until after the time specified
	if v := resp.Header.Get(HeaderXRateLimitRemaining); v != "" {
		r, err := strconv.Atoi(v)
		if err != nil {
			log.Warnf("failed to parse x-ratelimit-remaining header: %v", err)
		} else {
			remaining = r
		}
	}

	// if "x-ratelimit-reset" is present, don't retry until after the time specified
	if v := resp.Header.Get(HeaderXRateLimitReset); v != "" {
		seconds, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			log.Warnf("failed to parse x-ratelimit-reset header: %v", err)
		} else {
			reset = time.Unix(seconds, 0)
		}
	}

	if retryAfter > 0 {
		w.limiter.PauseFor(retryAfter)
		return true
	}

	// If remaining is 0 and reset is not zero, wait until reset time
	if remaining == 0 && !reset.IsZero() {
		retryAfter = time.Until(reset)
		w.limiter.PauseFor(retryAfter)
		return true
	}

	// Default fallback if no rate-limit headers are present
	w.limiter.PauseFor(w.defaultRetryAfter)
	return true
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
