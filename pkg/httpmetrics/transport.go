package httpmetrics

import (
	"context"
	"fmt"
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

var (
	mReqCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_request_count",
			Help: "The total number of HTTP requests",
		},
		[]string{"code", "method", "host", "service_name", "revision_name", "ce_type"},
	)
	mReqInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_client_request_in_flight",
			Help: "The number of outgoing HTTP requests currently inflight",
		},
		[]string{"method", "host", "service_name", "revision_name", "ce_type"},
	)
	mReqDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_client_request_duration_seconds",
			Help:    "The duration of HTTP requests",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60},
		},
		[]string{"code", "method", "host", "service_name", "revision_name", "ce_type"},
	)
	seenHostMap = sync.Map{}
)

var buckets = map[string]string{}
var bucketSuffixes = map[string]string{}

func SetBuckets(b map[string]string)         { buckets = b }
func SetBucketSuffixes(bs map[string]string) { bucketSuffixes = bs }

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

func ExtractInnerTransport(rt http.RoundTripper) http.RoundTripper {
	if mt, ok := rt.(*MetricsTransport); ok {
		return mt.inner
	}
	return rt
}

func mapErrorToLabel(err error) string {
	if strings.Contains(err.Error(), "no route to host") {
		return "no-route-to_host"
	}
	if strings.Contains(err.Error(), "i/o timeout") {
		return "io-timeout"
	}
	if strings.Contains(err.Error(), "TLS handshake timeout") {
		return "tls-handshake-timeout"
	}
	if strings.Contains(err.Error(), "TLS handshake error") {
		return "tls-handshake-error"
	}
	if strings.Contains(err.Error(), "unexpected EOF") {
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

		baseLabels := prometheus.Labels{
			"method":        r.Method,
			"host":          host,
			"service_name":  env.KnativeServiceName,
			"revision_name": env.KnativeRevisionName,
			"ce_type":       r.Header.Get(CeTypeHeader),
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

	if buckets == nil && bucketSuffixes == nil {
		setupWarning.Do(func() {
			clog.WarnContext(ctx, "no buckets configured, use httpmetrics.SetBuckets or SetBucketSuffixes")
		})
		return "other"
	}

	// Check the exact matches first.
	if b, ok := buckets[host]; ok {
		return b
	}
	// Then check the suffixes.
	for k, v := range bucketSuffixes {
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
		[]string{"resource"},
	)
	mGitHubRateLimit = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit",
			Help: "The number of requests allowed during the rate limit window",
		},
		[]string{"resource"},
	)
	mGitHubRateLimitReset = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit_reset",
			Help: "The timestamp at which the current rate limit window resets",
		},
		[]string{"resource"},
	)
	mGitHubRateLimitUsed = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit_used",
			Help: "The fraction of the rate limit window used",
		},
		[]string{"resource"},
	)
	mGitHubRateLimitTimeToReset = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "github_rate_limit_time_to_reset",
			Help: "The number of minutes until the current rate limit window resets",
		},
		[]string{"resource"},
	)
)

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
			remaining := val("X-RateLimit-Remaining")
			mGitHubRateLimitRemaining.With(prometheus.Labels{"resource": resource}).Set(remaining)

			limit := val("X-RateLimit-Limit")
			mGitHubRateLimit.With(prometheus.Labels{"resource": resource}).Set(limit)

			reset := val("X-RateLimit-Reset")
			mGitHubRateLimitReset.With(prometheus.Labels{"resource": resource}).Set(reset)

			if limit > 0 {
				used := (limit - remaining) / limit
				mGitHubRateLimitUsed.With(prometheus.Labels{"resource": resource}).Set(used)
			}

			if reset > 0 {
				timeToReset := time.Until(time.Unix(int64(reset), 0)).Minutes()
				mGitHubRateLimitTimeToReset.With(prometheus.Labels{"resource": resource}).Set(timeToReset)
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
				val, _, ok := strings.Cut(";", val)
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
