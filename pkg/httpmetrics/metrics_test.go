package httpmetrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestServerMetrics(t *testing.T) {
	handler := "test"
	http.Handle("/", Handler(handler, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if got := bucketize(c.host); got != c.bucket {
			t.Errorf("bucketize(%q) = %q, want %q", c.host, got, c.bucket)
		}
	}
}
