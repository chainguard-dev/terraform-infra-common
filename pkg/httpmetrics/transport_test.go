package httpmetrics

import (
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
