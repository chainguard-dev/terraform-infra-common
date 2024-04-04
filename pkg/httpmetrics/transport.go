package httpmetrics

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const CeTypeHeader string = "ce-type"

var (
	mReqCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_client_request_count",
			Help: "The total number of HTTP requests",
		},
		[]string{"code", "method", "host", "service_name", "configuration_name", "revision_name", "ce_type"},
	)
	mReqInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_client_request_in_flight",
			Help: "The number of outgoing HTTP requests currently inflight",
		},
		[]string{"method", "host", "service_name", "configuration_name", "revision_name", "ce_type"},
	)
	mReqDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_client_request_duration_seconds",
			Help:    "The duration of HTTP requests",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60},
		},
		[]string{"code", "method", "host", "service_name", "configuration_name", "revision_name", "ce_type"},
	)
	seenHostMap = make(map[string]int)
)

var buckets = map[string]string{}
var bucketSuffixes = map[string]string{}

func SetBuckets(b map[string]string)         { buckets = b }
func SetBucketSuffixes(bs map[string]string) { bucketSuffixes = bs }

// Transport is an http.RoundTripper that records metrics for each request.
var Transport = WrapTransport(http.DefaultTransport)

// WrapTransport wraps an http.RoundTripper with instrumentation.
func WrapTransport(t http.RoundTripper) http.RoundTripper {
	return instrumentRoundTripperCounter(
		instrumentRoundTripperInFlight(
			instrumentRoundTripperDuration(
				instrumentGitHubRateLimits(
					otelhttp.NewTransport(t)))))
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

func instrumentRoundTripperCounter(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		resp, err := next.RoundTrip(r)
		if err == nil {
			mReqCount.With(prometheus.Labels{
				"code":               fmt.Sprintf("%d", resp.StatusCode),
				"method":             r.Method,
				"host":               bucketize(r.URL.Host),
				"service_name":       env.KnativeServiceName,
				"configuration_name": env.KnativeConfigurationName,
				"revision_name":      env.KnativeRevisionName,
				"ce_type":            r.Header.Get(CeTypeHeader),
			}).Inc()
		} else {
			mReqCount.With(prometheus.Labels{
				"code":               mapErrorToLabel(err),
				"method":             r.Method,
				"host":               bucketize(r.URL.Host),
				"service_name":       env.KnativeServiceName,
				"configuration_name": env.KnativeConfigurationName,
				"revision_name":      env.KnativeRevisionName,
				"ce_type":            r.Header.Get(CeTypeHeader),
			}).Inc()
		}
		return resp, err
	}
}

func instrumentRoundTripperInFlight(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		g := mReqInFlight.With(prometheus.Labels{
			"method":             r.Method,
			"host":               bucketize(r.URL.Host),
			"service_name":       env.KnativeServiceName,
			"configuration_name": env.KnativeConfigurationName,
			"revision_name":      env.KnativeRevisionName,
			"ce_type":            r.Header.Get(CeTypeHeader),
		})
		g.Inc()
		defer g.Dec()
		return next.RoundTrip(r)
	}
}

func instrumentRoundTripperDuration(next http.RoundTripper) promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		start := time.Now()
		resp, err := next.RoundTrip(r)
		if err == nil {
			mReqDuration.With(prometheus.Labels{
				"code":               fmt.Sprintf("%d", resp.StatusCode),
				"method":             r.Method,
				"host":               bucketize(r.URL.Host),
				"service_name":       env.KnativeServiceName,
				"configuration_name": env.KnativeConfigurationName,
				"revision_name":      env.KnativeRevisionName,
				"ce_type":            r.Header.Get(CeTypeHeader),
			}).Observe(time.Since(start).Seconds())
		}
		return resp, err
	}
}

func bucketize(host string) string {
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
	if math.Mod(float64(seenHostMap[host]), 10) == 0 {
		seenHostMap[host]++
		slog.Warn(`bucketing host as "other", use httpmetrics.SetBucket{Suffixe}s`, "host", host, "seen", seenHostMap[host])
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
