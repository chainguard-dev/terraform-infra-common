/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"golang.org/x/sync/errgroup"
)

func TestTransport(t *testing.T) {
	var mux sync.Mutex
	requestSeen := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(requestSeen)
		mux.Lock()
		defer mux.Unlock()
		t.Log("got request")
	}))
	defer s.Close()

	// Cause the request to "hang" for a bit to ensure we can observe in-flight metrics.
	mux.Lock()

	grp := errgroup.Group{}
	grp.Go(func() error {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.URL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set(CeTypeHeader, "testce")
		resp, err := (&http.Client{Transport: Transport}).Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("want OK, got %s", resp.Status)
		}
		return nil
	})

	// Wait for the request to enter the server handler.
	// This ensures that the in-flight metric is incremented before we check it.
	<-requestSeen
	if got := testutil.ToFloat64(mReqInFlight.With(prometheus.Labels{
		"method":        http.MethodGet,
		"host":          "other",
		"service_name":  "unknown",
		"revision_name": "unknown",
		"ce_type":       "testce",
		"path":          "",
	})); got != 1 {
		t.Errorf("want metric in-flight = 1, got %f", got)
	}

	// Release the lock to allow the request to complete.
	mux.Unlock()

	// Wait for the request to finish.
	if err := grp.Wait(); err != nil {
		t.Fatal(err)
	}

	if got := testutil.ToFloat64(mReqCount.With(prometheus.Labels{
		"method":        http.MethodGet,
		"code":          "200",
		"host":          "other",
		"service_name":  "unknown",
		"revision_name": "unknown",
		"ce_type":       "testce",
		"path":          "",
	})); got != 1 {
		t.Errorf("want metric count = 1, got %f", got)
	}
	if got := testutil.ToFloat64(mReqInFlight.With(prometheus.Labels{
		"method":        http.MethodGet,
		"host":          "other",
		"service_name":  "unknown",
		"revision_name": "unknown",
		"ce_type":       "testce",
		"path":          "",
	})); got != 0 {
		t.Errorf("want metric in-flight = 0, got %f", got)
	}
}

func TestTransport_SkipBucketize(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Log("got request")
	}))
	defer s.Close()

	resp, err := (&http.Client{Transport: WrapTransport(http.DefaultTransport, WithSkipBucketize(true))}).Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want OK, got %s", resp.Status)
	}

	// Sample a metric to make sure labels are being properly applied.
	if got := testutil.ToFloat64(mReqCount.With(prometheus.Labels{
		"method":        http.MethodGet,
		"code":          "200",
		"host":          "unbucketized",
		"service_name":  "unknown",
		"revision_name": "unknown",
		"ce_type":       "",
		"path":          "",
	})); got != 1 {
		t.Errorf("want metric count = 1, got %f", got)
	}
}

func TestDockerHubRateLimitParsing(t *testing.T) {
	// Exercise the actual instrumentDockerHubRateLimit round tripper with a
	// test server that returns Docker Hub rate limit headers. Before the fix,
	// strings.Cut arguments were reversed so gauges were never set.
	for _, tt := range []struct {
		name          string
		host          string
		limit         string
		remaining     string
		wantLimit     float64
		wantRemaining float64
	}{
		{
			name:          "standard headers",
			host:          "index.docker.io",
			limit:         "100;w=21600",
			remaining:     "98;w=21600",
			wantLimit:     100,
			wantRemaining: 98,
		},
		{
			name:          "non-docker host ignored",
			host:          "ghcr.io",
			limit:         "100;w=21600",
			remaining:     "98;w=21600",
			wantLimit:     0,
			wantRemaining: 0,
		},
		{
			name:          "empty headers",
			host:          "index.docker.io",
			limit:         "",
			remaining:     "",
			wantLimit:     0,
			wantRemaining: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mDockerRateLimit.Set(0)
			mDockerRateLimitRemaining.Set(0)
			mDockerRateLimitUsed.Set(0)

			// Stub transport that returns the configured headers.
			stub := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{},
					Body:       http.NoBody,
				}
				if tt.limit != "" {
					resp.Header.Set("RateLimit-Limit", tt.limit)
				}
				if tt.remaining != "" {
					resp.Header.Set("RateLimit-Remaining", tt.remaining)
				}
				return resp, nil
			})

			transport := instrumentDockerHubRateLimit(stub)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+tt.host+"/v2/library/alpine/manifests/latest", nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := transport.RoundTrip(req); err != nil {
				t.Fatal(err)
			}

			if got := testutil.ToFloat64(mDockerRateLimit); got != tt.wantLimit {
				t.Errorf("docker_rate_limit: got %v, want %v", got, tt.wantLimit)
			}
			if got := testutil.ToFloat64(mDockerRateLimitRemaining); got != tt.wantRemaining {
				t.Errorf("docker_rate_limit_remaining: got %v, want %v", got, tt.wantRemaining)
			}
		})
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestMapErrorToLabel(t *testing.T) {
	for _, tt := range []struct {
		err  string
		want string
	}{
		{"dial tcp: no route to host", "no-route-to-host"},
		{"context deadline exceeded (i/o timeout)", "io-timeout"},
		{"remote error: TLS handshake timeout", "tls-handshake-timeout"},
		{"remote error: TLS handshake error", "tls-handshake-error"},
		{"unexpected EOF", "unexpected-eof"},
		{"something else entirely", "unknown-error"},
	} {
		t.Run(tt.err, func(t *testing.T) {
			got := mapErrorToLabel(errors.New(tt.err))
			if got != tt.want {
				t.Errorf("mapErrorToLabel(%q): got %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestExtractInnerTransport(t *testing.T) {
	t.Run("not wrapped", func(t *testing.T) {
		tr := &http.Transport{}
		if got := ExtractInnerTransport(tr); got != tr {
			t.Errorf("want %v, got %v", tr, got)
		}
	})

	t.Run("wrapped", func(t *testing.T) {
		inner := &http.Transport{}
		var tr = WrapTransport(inner)
		if got := ExtractInnerTransport(tr); got != inner {
			t.Errorf("want %v, got %v", inner, got)
		}
	})
}
