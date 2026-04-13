/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

const (
	CeTypeHeader          string = "ce-type"
	GoogClientTraceHeader string = "googclient_traceparent"
	OriginalTraceHeader   string = "original-traceparent"
)

// contextKey is an unexported type for context keys in this package, preventing
// collisions with keys defined in other packages.
type contextKey string

const (
	githubAppIDKey          contextKey = "github_app_id"
	githubInstallationIDKey contextKey = "github_installation_id"
)

// WithGitHubAppID returns a copy of ctx with the GitHub App ID attached.
// The transport reads this value to label rate limit metrics with the app
// that made the request, enabling per-app visibility into quota consumption.
func WithGitHubAppID(ctx context.Context, appID int64) context.Context {
	return context.WithValue(ctx, githubAppIDKey, strconv.FormatInt(appID, 10))
}

// WithGitHubInstallationID returns a copy of ctx with the GitHub installation
// ID attached. The transport reads this value to label rate limit metrics with
// the installation that made the request. GitHub enforces rate limits per
// installation, so this label identifies which (app, org) pair is consuming quota.
func WithGitHubInstallationID(ctx context.Context, installationID int64) context.Context {
	return context.WithValue(ctx, githubInstallationIDKey, strconv.FormatInt(installationID, 10))
}

var (
	mReqCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_request_count",
			Help: "The total number of HTTP requests",
		},
		[]string{"code", "method", "host", "service_name", "revision_name", "ce_type", "path"},
	)
	mReqInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_client_request_in_flight",
			Help: "The number of outgoing HTTP requests currently inflight",
		},
		[]string{"method", "host", "service_name", "revision_name", "ce_type", "path"},
	)
	mReqDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_client_request_duration_seconds",
			Help:    "The duration of HTTP requests",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60},
		},
		[]string{"code", "method", "host", "service_name", "revision_name", "ce_type", "path"},
	)
	seenHostMap = sync.Map{}
)

// bucketConfig holds the host-to-label mappings used by bucketize().
// Written once at startup via SetBuckets/SetBucketSuffixes, read on
// every HTTP round trip. Using atomic.Pointer avoids data races without
// adding mutex overhead on the hot path.
type bucketConfig struct {
	exact    map[string]string
	suffixes map[string]string
}

var activeBucketConfig atomic.Pointer[bucketConfig]

func init() {
	activeBucketConfig.Store(&bucketConfig{
		exact:    map[string]string{},
		suffixes: map[string]string{},
	})
}

// SetBuckets configures exact host-to-label mappings. Must be called
// before the first HTTP request for the mappings to take effect.
func SetBuckets(b map[string]string) {
	cfg := *activeBucketConfig.Load()
	cfg.exact = b
	activeBucketConfig.Store(&cfg)
}

// SetBucketSuffixes configures suffix-based host-to-label mappings.
// Must be called before the first HTTP request for the mappings to take effect.
func SetBucketSuffixes(bs map[string]string) {
	cfg := *activeBucketConfig.Load()
	cfg.suffixes = bs
	activeBucketConfig.Store(&cfg)
}

// Transport is an http.RoundTripper that records metrics for each request.
var Transport = WrapTransport(http.DefaultTransport)

type MetricsTransport struct {
	http.RoundTripper

	inner http.RoundTripper
}

type metricsTransportOptions struct {
	skipBucketize bool
}

type TransportOption func(*metricsTransportOptions)

// WithSkipBucketize is a TransportOption that skips the bucketization of the host.
// This is useful for transports that talk to an unbounded number of hosts,
// where bucketization would cause excessive metric cardinality.
// If true, the host label will be set to "unbucketized".
func WithSkipBucketize(skip bool) TransportOption {
	return func(opts *metricsTransportOptions) {
		opts.skipBucketize = skip
	}
}

// WrapTransport wraps an http.RoundTripper with instrumentation.
func WrapTransport(t http.RoundTripper, opts ...TransportOption) http.RoundTripper {
	topts := &metricsTransportOptions{}
	for _, opt := range opts {
		opt(topts)
	}

	return &MetricsTransport{
		RoundTripper: useGoogClientTraceparent(
			instrumentRequest(
				instrumentGitHubRateLimits(
					instrumentDockerHubRateLimit(
						otelhttp.NewTransport(
							newPreserveTraceparentTransport(t),
						),
					),
				), topts.skipBucketize,
			),
		),
		inner: t,
	}
}

// TransportUnwrapper is implemented by RoundTripper wrappers that can expose
// their underlying transport. This follows the same convention as errors.Unwrap.
type TransportUnwrapper interface {
	Unwrap() http.RoundTripper
}

// maxUnwrapDepth guards against infinite loops from buggy Unwrap implementations.
const maxUnwrapDepth = 10

// ExtractInnerTransport recursively unwraps layers of RoundTripper wrapping
// (MetricsTransport and any TransportUnwrapper) to find the base transport.
// Stops after maxUnwrapDepth iterations to prevent infinite loops.
func ExtractInnerTransport(rt http.RoundTripper) http.RoundTripper {
	for range maxUnwrapDepth {
		switch t := rt.(type) {
		case *MetricsTransport:
			rt = t.inner
		case TransportUnwrapper:
			rt = t.Unwrap()
		default:
			return rt
		}
	}
	return rt
}

func mapErrorToLabel(err error) string {
	msg := err.Error()
	if strings.Contains(msg, "no route to host") {
		return "no-route-to-host"
	}
	if strings.Contains(msg, "i/o timeout") {
		return "io-timeout"
	}
	if strings.Contains(msg, "TLS handshake timeout") {
		return "tls-handshake-timeout"
	}
	if strings.Contains(msg, "TLS handshake error") {
		return "tls-handshake-error"
	}
	if strings.Contains(msg, "unexpected EOF") {
		return "unexpected-eof"
	}
	return "unknown-error"
}

// These instrument methods based on promhttp, with bucketized host and Knative labels added:
// https://pkg.go.dev/github.com/prometheus/client_golang/prometheus/promhttp

func useGoogClientTraceparent(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		restoreTraceparentHeader(r)
		return next.RoundTrip(r)
	}
}

func instrumentRequest(next http.RoundTripper, skipBucketize bool) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		start := time.Now()

		tracer := otel.Tracer("httpmetrics")
		host := bucketize(r.Context(), r.URL.Host, skipBucketize)
		ctx, span := tracer.Start(r.Context(), fmt.Sprintf("http-%s-%s", r.Method, host))
		// Ensure that outgoing requests are nested under this span.
		r = r.WithContext(ctx)
		defer span.End()

		path := ""
		if r.URL.Host == "api.github.com" {
			path = bucketizeGitHubPath(r.URL.Path)
		}

		baseLabels := prometheus.Labels{
			"method":        r.Method,
			"host":          host,
			"service_name":  env.KnativeServiceName,
			"revision_name": env.KnativeRevisionName,
			"ce_type":       r.Header.Get(CeTypeHeader),
			"path":          path,
		}

		g := mReqInFlight.With(baseLabels)
		g.Inc()
		defer g.Dec()

		labels := maps.Clone(baseLabels)
		resp, err := next.RoundTrip(r)
		if err == nil {
			labels["code"] = fmt.Sprintf("%d", resp.StatusCode)

			// We only record the duration if we got a response.
			mReqDuration.With(labels).Observe(time.Since(start).Seconds())
		} else {
			labels["code"] = mapErrorToLabel(err)
		}
		mReqCount.With(labels).Inc()

		return resp, err
	}
}

var setupWarning sync.Once

func bucketize(ctx context.Context, host string, skip bool) string {
	if skip {
		return "unbucketized"
	}

	cfg := activeBucketConfig.Load()
	if len(cfg.exact) == 0 && len(cfg.suffixes) == 0 {
		setupWarning.Do(func() {
			clog.WarnContext(ctx, "no buckets configured, use httpmetrics.SetBuckets or SetBucketSuffixes")
		})
		return "other"
	}

	// Check the exact matches first.
	if b, ok := cfg.exact[host]; ok {
		return b
	}
	// Then check the suffixes.
	for k, v := range cfg.suffixes {
		if strings.HasSuffix(host, "."+k) {
			return v
		}
	}

	// Only log every 10th request to avoid flooding the logs.
	v, _ := seenHostMap.LoadOrStore(host, &atomic.Int64{})
	vInt := v.(*atomic.Int64)
	if seen := vInt.Add(1); (seen-1)%10 == 0 {
		clog.WarnContext(ctx, `bucketing host as "other", use httpmetrics.SetBucket{Suffixe}s`, "host", host, "seen", seen)
	}

	return "other"
}

var (
	mGitHubRateLimitRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit_remaining",
			Help: "The number of requests remaining in the current rate limit window",
		},
		[]string{"resource", "organization", "app_id", "installation_id"},
	)
	mGitHubRateLimit = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit",
			Help: "The number of requests allowed during the rate limit window",
		},
		[]string{"resource", "organization", "app_id", "installation_id"},
	)
	mGitHubRateLimitReset = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit_reset",
			Help: "The timestamp at which the current rate limit window resets",
		},
		[]string{"resource", "organization", "app_id", "installation_id"},
	)
	mGitHubRateLimitUsed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit_used",
			Help: "The fraction of the rate limit window used",
		},
		[]string{"resource", "organization", "app_id", "installation_id"},
	)
	mGitHubRateLimitTimeToReset = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit_time_to_reset",
			Help: "The number of minutes until the current rate limit window resets",
		},
		[]string{"resource", "organization", "app_id", "installation_id"},
	)
	mGitHubRateLimitErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_rate_limit_errors_total",
			Help: "GitHub API requests rejected due to rate limiting (403/429 with rate limit headers)",
		},
		[]string{"resource", "organization", "app_id", "installation_id", "service_name", "code", "rate_limit_type"},
	)
	mGitHubRetryAfterSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "github_retry_after_seconds",
			Help:    "Retry-After values from GitHub rate limit responses",
			Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 900, 1800, 3600},
		},
		[]string{"organization", "app_id", "installation_id", "service_name", "rate_limit_type"},
	)
)

// extractOrgFromGitHubURL extracts the organization from a GitHub API URL path.
// GitHub API URLs typically have the format: /repos/{org}/{repo}/... or /orgs/{org}/...
func extractOrgFromGitHubURL(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) < 2 {
		return "unknown"
	}

	// Handle /repos/{org}/{repo}/... URLs
	if parts[0] == "repos" && len(parts) >= 2 {
		return parts[1]
	}

	// Handle /orgs/{org}/... URLs
	if parts[0] == "orgs" && len(parts) >= 2 {
		return parts[1]
	}

	return "unknown"
}

// instrumentGitHubRateLimits is a promhttp.RoundTripperFunc that records GitHub rate limit metrics.
// See https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api?apiVersion=2022-11-28
func instrumentGitHubRateLimits(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := next.RoundTrip(r)
		if err != nil {
			return resp, err
		}
		if r.URL.Host == "api.github.com" {
			resource := resp.Header.Get("X-RateLimit-Resource")
			if resource == "" {
				resource = "unknown"
			}

			// Extract organization from the request URL
			organization := extractOrgFromGitHubURL(r.URL.Path)

			// Read caller-supplied app/installation IDs from the request context.
			// These are set by callers via WithGitHubAppID and WithGitHubInstallationID.
			// Empty string when not set — callers that do not set these values produce
			// time series with app_id="" and installation_id="".
			appID, _ := r.Context().Value(githubAppIDKey).(string)
			installationID, _ := r.Context().Value(githubInstallationIDKey).(string)

			val := func(key string) float64 {
				val := resp.Header.Get(key)
				if val == "" {
					return 0
				}
				i, err := strconv.Atoi(val)
				if err != nil {
					return 0
				}
				return float64(i)
			}
			labels := prometheus.Labels{
				"resource":        resource,
				"organization":    organization,
				"app_id":          appID,
				"installation_id": installationID,
			}

			remaining := val("X-RateLimit-Remaining")
			mGitHubRateLimitRemaining.With(labels).Set(remaining)

			limit := val("X-RateLimit-Limit")
			mGitHubRateLimit.With(labels).Set(limit)

			reset := val("X-RateLimit-Reset")
			mGitHubRateLimitReset.With(labels).Set(reset)

			if limit > 0 {
				used := (limit - remaining) / limit
				mGitHubRateLimitUsed.With(labels).Set(used)
			}

			if reset > 0 {
				timeToReset := time.Until(time.Unix(int64(reset), 0)).Minutes()
				mGitHubRateLimitTimeToReset.With(labels).Set(timeToReset)
			}

			// Compute timeToReset only when the header is present.
			var timeToReset float64
			if reset > 0 {
				timeToReset = time.Until(time.Unix(int64(reset), 0)).Minutes()
			}

			// Proactive warning when approaching rate limit exhaustion (<10% remaining).
			// Only log at round-number boundaries to avoid flooding on every request.
			if limit > 0 && remaining > 0 && remaining/limit < 0.1 {
				rem := int64(remaining)
				if rem%1000 == 0 || (rem < 1000 && rem%100 == 0) {
					clog.WarnContextf(r.Context(), "GitHub rate limit <10%%: org=%s, resource=%s, remaining=%.0f/%.0f, reset_in=%.1fm",
						organization, resource, remaining, limit, timeToReset)
				}
			}

			// Detect rate limit errors (403/429).
			// Only classify as rate-limit if rate limit headers are actually present,
			// otherwise a permission 403 would be miscounted.
			hasRateLimitHeaders := resp.Header.Get("X-RateLimit-Remaining") != ""
			if (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests) && hasRateLimitHeaders {
				rateLimitType := "unknown"
				if remaining == 0 {
					rateLimitType = "primary"
				} else if resp.Body != nil {
					// Check for secondary rate limit by inspecting the response body
					// for documentation_url containing abuse or secondary rate limit references.
					// Re-populate the body afterward so downstream code can still read it,
					// matching the pattern used by go-github's CheckResponse.
					data, readErr := io.ReadAll(resp.Body)
					resp.Body.Close()
					if readErr == nil {
						body := string(data)
						if strings.Contains(body, "#abuse-rate-limits") || strings.Contains(body, "secondary-rate-limits") {
							rateLimitType = "secondary"
						}
					}
					resp.Body = io.NopCloser(bytes.NewBuffer(data))
				}

				mGitHubRateLimitErrors.With(prometheus.Labels{
					"resource":        resource,
					"organization":    organization,
					"app_id":          appID,
					"installation_id": installationID,
					"service_name":    env.KnativeServiceName,
					"code":            strconv.Itoa(resp.StatusCode),
					"rate_limit_type": rateLimitType,
				}).Inc()

				// Parse Retry-After header when present.
				if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
					if seconds, parseErr := strconv.Atoi(retryAfter); parseErr == nil {
						mGitHubRetryAfterSeconds.With(prometheus.Labels{
							"organization":    organization,
							"app_id":          appID,
							"installation_id": installationID,
							"service_name":    env.KnativeServiceName,
							"rate_limit_type": rateLimitType,
						}).Observe(float64(seconds))
					}
				}

				clog.WarnContextf(r.Context(), "GitHub rate limit hit: %s %s (org=%s, resource=%s, type=%s, remaining=%.0f, reset_in=%.1fm)",
					r.Method, r.URL.Path, organization, resource, rateLimitType, remaining, timeToReset)
			}
		}
		return resp, err
	}
}

var (
	mDockerRateLimitRemaining = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "docker_rate_limit_remaining",
			Help: "The number of requests remaining in the current rate limit window",
		},
	)
	mDockerRateLimit = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "docker_rate_limit",
			Help: "The number of requests allowed during the rate limit window",
		},
	)
	mDockerRateLimitUsed = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "docker_rate_limit_used",
			Help: "The fraction of the rate limit window used",
		},
	)
)

func instrumentDockerHubRateLimit(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := next.RoundTrip(r)
		if err != nil {
			return resp, err
		}
		if r.URL.Host == "index.docker.io" {
			// https://www.docker.com/blog/checking-your-current-docker-pull-rate-limits-and-status/
			// Values are like:
			// "Ratelimit-Limit: 100;w=21600" indicating "100 requests per 21600 seconds" (6 hours)
			// "Ratelimit-Remaining: 98;w=21600" indicating "98 requests remaining in the current 6 hour window"
			val := func(key string) float64 {
				val := resp.Header.Get(key)
				if val == "" {
					return 0
				}
				val, _, ok := strings.Cut(val, ";")
				if !ok {
					return 0
				}
				i, err := strconv.Atoi(val)
				if err != nil {
					return 0
				}
				return float64(i)
			}

			remaining := val("RateLimit-Remaining")
			if remaining > 0 {
				mDockerRateLimitRemaining.Set(remaining)
			}

			limit := val("RateLimit-Limit")

			if limit > 0 {
				mDockerRateLimit.Set(limit)
				used := (limit - remaining) / limit
				mDockerRateLimitUsed.Set(used)
			}
		}
		return resp, err
	}
}
