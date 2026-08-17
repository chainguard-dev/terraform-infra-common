/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.opentelemetry.io/otel"
)

func TestServerMetrics(t *testing.T) {
	handler := "test"
	http.Handle("/", Handler(handler, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	srv := httptest.NewServer(http.DefaultServeMux)

	resp, err := srv.Client().Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want OK, got %s", resp.Status)
	}

	// Sample a metric to make sure labels are being properly applied.
	if got := testutil.ToFloat64(counter.MustCurryWith(prometheus.Labels{
		"handler": handler,
		"method":  "get",
		"code":    "200",
	})); got != 1 {
		t.Errorf("want metric count = 1, got %f", got)
	}
}

type nonFlushingResponseWriter struct {
	header http.Header
}

func (w *nonFlushingResponseWriter) Header() http.Header {
	return w.header
}

func (*nonFlushingResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (*nonFlushingResponseWriter) WriteHeader(int) {}

type flushErrorResponseWriter struct {
	*nonFlushingResponseWriter
	flushed *bool
	err     error
}

func (w *flushErrorResponseWriter) FlushError() error {
	*w.flushed = true
	return w.err
}

type flushingResponseWriter struct {
	*nonFlushingResponseWriter
	flushed *bool
}

func (w *flushingResponseWriter) Flush() {
	*w.flushed = true
}

type unwrappingResponseWriter struct {
	http.ResponseWriter
}

func (w *unwrappingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func TestHandlerDoesNotAdvertiseUnsupportedFlush(t *testing.T) {
	var advertised bool
	var flushErr error
	h := Handler("non-flushing-test", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, advertised = w.(http.Flusher)
		flushErr = http.NewResponseController(w).Flush()
	}))

	h.ServeHTTP(
		&nonFlushingResponseWriter{header: make(http.Header)},
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if advertised {
		t.Error("handler advertised flush support for a non-flushing response writer")
	}
	if !errors.Is(flushErr, http.ErrNotSupported) {
		t.Errorf("flush error = %v, want http.ErrNotSupported", flushErr)
	}
}

func TestHandlerPreservesFlusher(t *testing.T) {
	var advertised bool
	var flushed bool
	h := Handler("flushing-test", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		advertised = ok
		if ok {
			flusher.Flush()
		}
	}))

	h.ServeHTTP(
		&flushingResponseWriter{
			nonFlushingResponseWriter: &nonFlushingResponseWriter{header: make(http.Header)},
			flushed:                   &flushed,
		},
		httptest.NewRequest(http.MethodGet, "/", nil),
	)

	if !advertised {
		t.Error("handler did not preserve flush support")
	}
	if !flushed {
		t.Error("flush did not reach the underlying response writer")
	}
}

func TestHandlerPreservesResponseControllerFlush(t *testing.T) {
	flushFailed := errors.New("flush failed")
	for _, tc := range []struct {
		name    string
		writer  func(*bool) http.ResponseWriter
		wantErr error
	}{
		{
			name: "FlushError",
			writer: func(flushed *bool) http.ResponseWriter {
				return &flushErrorResponseWriter{
					nonFlushingResponseWriter: &nonFlushingResponseWriter{header: make(http.Header)},
					flushed:                   flushed,
					err:                       flushFailed,
				}
			},
			wantErr: flushFailed,
		},
		{
			name: "Unwrap",
			writer: func(flushed *bool) http.ResponseWriter {
				return &unwrappingResponseWriter{ResponseWriter: &flushingResponseWriter{
					nonFlushingResponseWriter: &nonFlushingResponseWriter{header: make(http.Header)},
					flushed:                   flushed,
				}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var flushed bool
			var flushErr error
			h := Handler("response-controller-test", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				flushErr = http.NewResponseController(w).Flush()
			}))

			h.ServeHTTP(tc.writer(&flushed), httptest.NewRequest(http.MethodGet, "/", nil))

			if !errors.Is(flushErr, tc.wantErr) {
				t.Errorf("flush error = %v, want %v", flushErr, tc.wantErr)
			}
			if !flushed {
				t.Error("flush did not reach the underlying response writer")
			}
		})
	}
}

// TestHandlerStreamingFlush checks that a handler behind Handler can flush
// each write to the client before it returns. grpc-gateway flushes after every
// streamed message through http.ResponseController and aborts the stream when
// the flush is unsupported, so a metrics wrapper that hides flush support from
// http.ResponseController breaks every server-streaming endpoint behind it.
func TestHandlerStreamingFlush(t *testing.T) {
	flushErr := make(chan error, 1)
	release := make(chan struct{})

	srv := httptest.NewServer(Handler("stream-test", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, "first\n"); err != nil {
			flushErr <- fmt.Errorf("write first line: %w", err)
			return
		}

		err := http.NewResponseController(w).Flush()
		flushErr <- err
		if err != nil {
			return
		}

		// Hold the response open until the test has read the flushed line, so
		// its arrival proves the flush delivered it.
		<-release
		_, _ = io.WriteString(w, "second\n")
	})))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if err := <-flushErr; err != nil {
		t.Fatalf("flush inside a wrapped handler failed: %v", err)
	}

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read flushed line: %v", err)
	}
	if line != "first\n" {
		t.Fatalf("want flushed line %q, got %q", "first\n", line)
	}

	close(release)
	rest, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read rest of response: %v", err)
	}
	if string(rest) != "second\n" {
		t.Fatalf("want remainder %q, got %q", "second\n", string(rest))
	}
}

func TestResolveServiceName(t *testing.T) {
	for _, tc := range []struct {
		name        string
		kService    string
		cloudRunJob string
		want        string
	}{{
		name:     "service sets K_SERVICE",
		kService: "my-service",
		want:     "my-service",
	}, {
		name:        "job falls back to CLOUD_RUN_JOB",
		cloudRunJob: "my-job",
		want:        "my-job",
	}, {
		name:        "K_SERVICE wins over CLOUD_RUN_JOB",
		kService:    "my-service",
		cloudRunJob: "my-job",
		want:        "my-service",
	}, {
		name: "neither set yields unknown",
		want: "unknown",
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveServiceName(tc.kService, tc.cloudRunJob); got != tc.want {
				t.Errorf("resolveServiceName(%q, %q) = %q, want %q", tc.kService, tc.cloudRunJob, got, tc.want)
			}
		})
	}
}

func TestBucketize(t *testing.T) {
	SetBuckets(map[string]string{
		"api.github.com":                       "GH API",
		"cgr.dev":                              "cgr.dev",
		"fulcio.sigstore.dev":                  "Fulcio",
		"gcr.io":                               "GCR",
		"ghcr.io":                              "GHCR",
		"gke.gcr.io":                           "gke.gcr.io",
		"index.docker.io":                      "Dockerhub",
		"issuer.enforce.dev":                   "issuer.enforce.dev",
		"pkg-containers.githubusercontent.com": "GHCR blob",
		"quay.io":                              "Quay",
		"registry.k8s.io":                      "registry.k8s.io",
		"rekor.sigstore.dev":                   "Rekor",
		"storage.googleapis.com":               "GCS",
		"registry.gitlab.com":                  "registry.gitlab.com",
		"gitlab.com":                           "GitLab",
		"github.com":                           "GitHub",
	})
	SetBucketSuffixes(map[string]string{
		"googleapis.com":           "Google API",
		"amazonaws.com":            "AWS",
		"gcr.io":                   "GCR",
		"r2.cloudflarestorage.com": "R2",
	})
	for _, c := range []struct{ host, bucket string }{
		{"gcr.io", "GCR"},
		{"us.gcr.io", "GCR"},
		{"notgcr.io", "other"},
		{"notamazonaws.com", "other"},
		{"foo.us-east-1.amazonaws.com", "AWS"},
		{"compute.googleapis.com", "Google API"},
		{"storage.googleapis.com", "GCS"},
		{"amazonaws.com", "other"},  // only as a prefix
		{"googleapis.com", "other"}, // only as a prefix
		{"ghcr.io", "GHCR"},
		{"api.github.com", "GH API"},
		{"index.docker.io", "Dockerhub"},
		{"fulcio.sigstore.dev", "Fulcio"},
		{"rekor.sigstore.dev", "Rekor"},
		{"issuer.enforce.dev", "issuer.enforce.dev"},
	} {
		if got := bucketize(t.Context(), c.host, false); got != c.bucket {
			t.Errorf("bucketize(%q) = %q, want %q", c.host, got, c.bucket)
		}
	}
}

func Test_SetupMetrics(t *testing.T) {
	ctx := context.Background()

	cleanup := SetupMetrics(ctx)
	if cleanup == nil {
		t.Fatal("SetupMetrics() returned nil cleanup function")
	}

	provider := otel.GetMeterProvider()
	if provider == nil {
		t.Fatal("global meter provider was not set")
	}

	meter := provider.Meter("test-meter")
	if meter == nil {
		t.Fatal("failed to create meter from provider")
	}

	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("failed to create counter: %v", err)
	}
	if counter == nil {
		t.Fatal("counter is nil")
	}

	cleanup()
}

func Test_SetupMetricsCleanup(t *testing.T) {
	ctx := context.Background()

	cleanup := SetupMetrics(ctx)
	if cleanup == nil {
		t.Fatal("SetupMetrics() returned nil cleanup function")
	}

	cleanup()
	cleanup()
}
