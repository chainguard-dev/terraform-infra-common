package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v75/github"
)

func TestGitHubClientConfiguration(t *testing.T) {
	tests := []struct {
		name string
		opts []GitHubClientOption
		want func(*testing.T, GitHubClient)
	}{
		{
			name: "default configuration without WithClient",
			opts: nil,
			want: func(t *testing.T, client GitHubClient) {
				t.Helper()

				// Verify a default client was created
				if got := client.Client(); got == nil {
					t.Error("expected default client to be created, got nil")
				}

				// Verify default buffer size
				if got := client.bufSize; got != 1024*1024 {
					t.Errorf("default bufSize = %v, want %v", got, 1024*1024)
				}
			},
		},
		{
			name: "with custom HTTP client",
			opts: []GitHubClientOption{
				WithClient(github.NewClient(&http.Client{
					Transport: &http.Transport{},
				})),
			},
			want: func(t *testing.T, client GitHubClient) {
				t.Helper()

				// Verify client was set
				if got := client.Client(); got == nil {
					t.Error("expected client to be set, got nil")
				}
			},
		},
		{
			name: "with test server",
			opts: nil,
			want: func(t *testing.T, _ GitHubClient) {
				t.Helper()

				// Create test server
				ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"status": "ok"}`))
				}))
				defer ts.Close()

				// Parse the test server URL
				baseURL, err := url.Parse(ts.URL + "/")
				if err != nil {
					t.Fatalf("failed to parse test server URL: %v", err)
				}

				// Create a client pointing to test server
				httpClient := &http.Client{Transport: &http.Transport{}}
				testClient := github.NewClient(httpClient)
				testClient.BaseURL = baseURL

				customClient := NewGitHubClient(
					context.Background(),
					"test-org",
					"test-repo",
					"test-policy",
					WithSecondaryRateLimitWaiter(),
					WithClient(testClient),
				)

				// Client should now be configured to use the test server
				if got := customClient.Client().BaseURL.String(); got != baseURL.String() {
					t.Errorf("baseURL = %v, want %v", got, baseURL)
				}
			},
		},
		{
			// `WithSecondaryRateLimitWaiter` ought to change the underlying transport to SecondaryRateLimitWaiter.
			// At the moment, it does not.
			// We *could* fix it, but at the time of writing we are trying to get a better handle
			// on how we relate to GitHub rate-limiting.
			// Rather than enable a new rate-limiting feature that's not been production tested,
			// we've documented that the option doesn't do anything, and will leave this test here,
			// but with the assertion inverted, to indicate that the no-op behavior is intentional.
			//
			// See https://github.com/chainguard-dev/terraform-infra-common/pull/1211/
			// and https://chainguard-dev.slack.com/archives/C05SJTTHE79/p1763040569561639
			name: "WithSecondaryRateLimitWaiter modifies transport (xfail)",
			opts: []GitHubClientOption{
				WithClient(github.NewClient(&http.Client{
					Transport: &http.Transport{},
				})),
				WithSecondaryRateLimitWaiter(),
			},
			want: func(t *testing.T, client GitHubClient) {
				t.Helper()

				// Transport was wrapped with SecondaryRateLimitWaiter
				transport := client.Client().Client().Transport
				if _, ok := transport.(*SecondaryRateLimitWaiter); ok {
					t.Errorf("transport unexpectedly changed to SecondaryRateLimitWaiter")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewGitHubClient(
				context.Background(),
				"test-org",
				"test-repo",
				"test-policy",
				tt.opts...,
			)

			// Verify org and repo are set correctly in all cases
			if got := client.org; got != "test-org" {
				t.Errorf("client.org = %v, want test-org", got)
			}
			if got := client.repo; got != "test-repo" {
				t.Errorf("client.repo = %v, want test-repo", got)
			}

			// Run the specific test case assertions
			tt.want(t, client)
		})
	}
}
