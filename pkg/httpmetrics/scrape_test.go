/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
)

func TestHTTPClientMetricsScrape(t *testing.T) {
	// A fresh process preserves the package-init collectors without inheriting
	// instruments or global meter providers from other tests.
	if os.Getenv("HTTPMETRICS_SCRAPE_TEST_CHILD") != "1" {
		binary, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), binary, "-test.run=^TestHTTPClientMetricsScrape$")
		cmd.Env = append(os.Environ(), "HTTPMETRICS_SCRAPE_TEST_CHILD=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("scrape subprocess: %v\n%s", err, output)
		}
		return
	}

	cleanup := SetupMetrics(t.Context())
	t.Cleanup(cleanup)
	counter, err := otel.Meter("scrape-regression").Int64Counter("scrape_regression_work")
	if err != nil {
		t.Fatal(err)
	}
	counter.Add(t.Context(), 1)
	handler := promhttp.Handler()
	scrape := func() {
		t.Helper()
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("scrape status = %d: %s", recorder.Code, recorder.Body.String())
		}
	}
	scrape()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	client := &http.Client{Transport: WrapTransport(server.Client().Transport)}
	t.Cleanup(client.CloseIdleConnections)
	for i := range 2 {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_, readErr := io.Copy(io.Discard, response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("response body: read=%v close=%v", readErr, closeErr)
		}
		scrape()

		families, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			t.Fatal(err)
		}
		counts := make(map[string]uint64, 2)
		var unrelated float64
		for _, family := range families {
			switch family.GetName() {
			case "http_client_request_duration_seconds", "otel_http_client_request_duration_seconds":
				for _, metric := range family.GetMetric() {
					counts[family.GetName()] += metric.GetHistogram().GetSampleCount()
				}
			case "scrape_regression_work_total":
				for _, metric := range family.GetMetric() {
					unrelated += metric.GetCounter().GetValue()
				}
			}
		}
		for _, name := range []string{"http_client_request_duration_seconds", "otel_http_client_request_duration_seconds"} {
			if got, want := counts[name], uint64(i+1); got != want {
				t.Errorf("%s sample count = %d, want %d", name, got, want)
			}
		}
		if unrelated != 1 {
			t.Errorf("unrelated counter = %v, want 1", unrelated)
		}
	}
}
