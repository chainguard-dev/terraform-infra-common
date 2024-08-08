package httpmetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestTransport(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Log("got request")
	}))
	defer s.Close()

	resp, err := (&http.Client{Transport: Transport}).Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want OK, got %s", resp.Status)
	}

	// Sample a metric to make sure labels are being properly applied.
	if got := testutil.ToFloat64(mReqCount.MustCurryWith(prometheus.Labels{
		"method": "get",
		"code":   "200",
		"host":   "other",
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
