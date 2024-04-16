/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chainguard-dev/clog"
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
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"code", "method", "host", "service_name", "configuration_name", "revision_name", "ce_type"},
	)
	seenHostMap = make(map[string]int)
)

// Transport is an http.RoundTripper that records metrics for each request.
var Transport = WrapTransport(http.DefaultTransport)

// WrapTransport wraps an http.RoundTripper with instrumentation.
func WrapTransport(t http.RoundTripper) http.RoundTripper {
	return instrumentRoundTripperCounter(
		instrumentRoundTripperInFlight(
			instrumentRoundTripperDuration(
				instrumentDockerHubRateLimit(
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
				"host":               bucketize(r.Context(), r.URL.Host),
				"service_name":       env.KnativeServiceName,
				"configuration_name": env.KnativeConfigurationName,
				"revision_name":      env.KnativeRevisionName,
				"ce_type":            r.Header.Get(CeTypeHeader),
			}).Inc()
		} else {
			mReqCount.With(prometheus.Labels{
				"code":               mapErrorToLabel(err),
				"method":             r.Method,
				"host":               bucketize(r.Context(), r.URL.Host),
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
			"host":               bucketize(r.Context(), r.URL.Host),
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
				"host":               bucketize(r.Context(), r.URL.Host),
				"service_name":       env.KnativeServiceName,
				"configuration_name": env.KnativeConfigurationName,
				"revision_name":      env.KnativeRevisionName,
				"ce_type":            r.Header.Get(CeTypeHeader),
			}).Observe(time.Since(start).Seconds())
		}
		return resp, err
	}
}

var buckets = map[string]string{
	"api.github.com":                       "GH API",
	"cgr.dev":                              "cgr.dev",
	"distroless.dev":                       "distroless.dev",
	"ingress.eventing-system.svc":          "eventing",
	"fulcio.sigstore.dev":                  "Fulcio",
	"gcr.io":                               "GCR",
	"ghcr.io":                              "GHCR",
	"gke.gcr.io":                           "gke.gcr.io",
	"index.docker.io":                      "Dockerhub",
	"issuer.chainops.dev":                  "issuer.chainops.dev",
	"issuer.enforce.dev":                   "issuer.enforce.dev",
	"pkg-containers.githubusercontent.com": "GHCR blob",
	"quay.io":                              "Quay",
	"registry.k8s.io":                      "registry.k8s.io",
	"rekor.sigstore.dev":                   "Rekor",
	"storage.googleapis.com":               "GCS",
	"registry.gitlab.com":                  "registry.gitlab.com",
	"gitlab.com":                           "GitLab",
	"github.com":                           "GitHub",
	"169.254.169.254":                      "metadata server",
	"all-broker-ingress.eventing-system.svc.cluster.local": "all-broker-ingress",
}

var bucketSuffixes = map[string]string{
	"googleapis.com":           "Google API",
	"amazonaws.com":            "AWS",
	"gcr.io":                   "GCR",
	"r2.cloudflarestorage.com": "R2",
	"a.run.app":                "Cloud Run",
}

func bucketize(ctx context.Context, host string) string {
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
		clog.FromContext(ctx).Infof("bucketing host [%s] as \"other\"; seen %d times, update list at api-internal/pkg/metrics", host, seenHostMap[host])
	}
	return "other"
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
