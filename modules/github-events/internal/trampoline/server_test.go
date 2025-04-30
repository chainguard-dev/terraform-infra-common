package trampoline

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v71/github"
	"github.com/jonboulle/clockwork"
)

type fakeClient struct {
	cloudevents.Client

	events []cloudevents.Event
}

func (f *fakeClient) Send(_ context.Context, event cloudevents.Event) cloudevents.Result {
	fmt.Println("send!", event)
	f.events = append(f.events, event)
	return nil
}

func TestTrampoline(t *testing.T) {
	client := &fakeClient{}

	secret := []byte("hunter2")
	clock := clockwork.NewFakeClock()
	opts := ServerOptions{
		Secrets: [][]byte{
			[]byte("badsecret"), // This secret should be ignored
			secret,
		},
	}
	impl := NewServer(client, opts)
	impl.clock = clock

	srv := httptest.NewServer(impl)
	defer srv.Close()

	body := map[string]interface{}{
		"action": "push",
		"repository": map[string]interface{}{
			"full_name": "org/repo",
		},
		"foo": "bar",
	}
	resp, err := sendevent(t, srv.Client(), srv.URL, "push", body, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}

	// Generate expected event body
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("error encoding body: %v", err)
	}
	enc, err := json.Marshal(eventData{
		When: clock.Now(),
		Headers: &eventHeaders{
			HookID:     "1234",
			DeliveryID: "5678",
			UserAgent:  t.Name(),
			Event:      "push",
		},
		Body: json.RawMessage(b),
	})
	if err != nil {
		t.Fatalf("error encoding body: %v", err)
	}

	want := []cloudevents.Event{{
		Context: cloudevents.EventContextV1{
			Type:            "dev.chainguard.github.push",
			Source:          *types.ParseURIRef("localhost"),
			ID:              "5678",
			DataContentType: cloudevents.StringOfApplicationJSON(),
			Subject:         github.Ptr("org/repo"),
			Extensions: map[string]interface{}{
				"action":     "push",
				"githubhook": "1234",
			},
		}.AsV1(),
		DataEncoded: enc,
	}}
	if diff := cmp.Diff(want, client.events); diff != "" {
		t.Error(diff)
	}
}

func sendevent(t *testing.T, client *http.Client, url string, eventType string, payload interface{}, secret []byte) (*http.Response, error) {
	t.Helper()

	b := new(bytes.Buffer)
	if err := json.NewEncoder(b).Encode(payload); err != nil {
		t.Fatalf("error encoding payload: %v", err)
	}

	// Compute the signature
	mac := hmac.New(sha256.New, secret)
	mac.Write(b.Bytes())
	sig := fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))

	r, err := http.NewRequest(http.MethodPost, url, b)
	if err != nil {
		return nil, err
	}
	r.Host = "localhost"
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add(github.SHA256SignatureHeader, sig)
	r.Header.Add(github.EventTypeHeader, eventType)
	r.Header.Add("X-Github-Hook-ID", "1234")
	r.Header.Add(github.DeliveryIDHeader, "5678")
	r.Header.Set("User-Agent", t.Name())

	return client.Do(r)
}

func TestForbidden(t *testing.T) {
	srv := httptest.NewServer(NewServer(&fakeClient{}, ServerOptions{}))
	defer srv.Close()

	// Doesn't really matter what we send, we just want to ensure we get a forbidden response
	resp, err := sendevent(t, srv.Client(), srv.URL, "push", nil, nil)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected status: %v", resp.Status)
	}
}

func TestWebhookIDFilter(t *testing.T) {
	secret := []byte("hunter2")
	opts := ServerOptions{
		Secrets:   [][]byte{secret},
		WebhookID: []string{"doesnotmatch"},
	}
	srv := httptest.NewServer(NewServer(&fakeClient{}, opts))
	defer srv.Close()

	// Send an event with the requested action
	resp, err := sendevent(t, srv.Client(), srv.URL, "check_run", nil, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected status: %v", resp.Status)
	}
}

func TestRequestedOnlyWebhook(t *testing.T) {
	secret := []byte("hunter2")
	opts := ServerOptions{
		Secrets:              [][]byte{secret},
		RequestedOnlyWebhook: []string{"1234"},
	}
	srv := httptest.NewServer(NewServer(&fakeClient{}, opts))
	defer srv.Close()

	// Send an event with the requested action
	resp, err := sendevent(t, srv.Client(), srv.URL, "check_run", map[string]interface{}{
		"action": "requested",
		"repository": map[string]interface{}{
			"full_name": "org/repo",
		},
		"organization": map[string]interface{}{
			"login": "org",
		},
	}, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}

	// Send the same event again, but without the requested action
	resp, err = sendevent(t, srv.Client(), srv.URL, "check_run", nil, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected status: %v", resp.Status)
	}
}

func TestExtractPullRequestInfo(t *testing.T) {
	testCases := []struct {
		name      string
		eventType string
		payload   map[string]interface{}
		expected  string
	}{
		{
			name:      "pull_request event with valid data",
			eventType: "pull_request",
			payload: map[string]interface{}{
				"number": float64(123),
				"repository": map[string]interface{}{
					"full_name": "foo/bar",
				},
			},
			expected: "foo/bar#123",
		},
		{
			name:      "not a pull_request event",
			eventType: "push",
			payload: map[string]interface{}{
				"number": float64(123),
				"repository": map[string]interface{}{
					"full_name": "foo/bar",
				},
			},
			expected: "",
		},
		{
			name:      "pull_request event with missing number",
			eventType: "pull_request",
			payload: map[string]interface{}{
				"repository": map[string]interface{}{
					"full_name": "foo/bar",
				},
			},
			expected: "",
		},
		{
			name:      "pull_request event with missing repo",
			eventType: "pull_request",
			payload: map[string]interface{}{
				"number": float64(123),
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function directly with map
			result := extractPullRequestInfo(tc.eventType, tc.payload)

			// Check the result
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestPullRequestExtension(t *testing.T) {
	client := &fakeClient{}
	secret := []byte("hunter2")
	clock := clockwork.NewFakeClock()
	opts := ServerOptions{
		Secrets: [][]byte{secret},
	}
	impl := NewServer(client, opts)
	impl.clock = clock

	srv := httptest.NewServer(impl)
	defer srv.Close()

	// Send a pull_request event
	prPayload := map[string]interface{}{
		"action": "opened",
		"number": 123,
		"repository": map[string]interface{}{
			"full_name": "foo/bar",
		},
		"organization": map[string]interface{}{
			"login": "foo",
		},
	}

	resp, err := sendevent(t, srv.Client(), srv.URL, "pull_request", prPayload, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}

	// Check that the pullrequest extension was added
	if len(client.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(client.events))
	}

	pullrequest, ok := client.events[0].Extensions()["pullrequest"]
	if !ok {
		t.Fatal("pullrequest extension not found")
	}
	if pullrequest != "foo/bar#123" {
		t.Errorf("unexpected pullrequest value: %v", pullrequest)
	}

	// Reset client events
	client.events = nil

	// Send a non-pull_request event
	nonPrPayload := map[string]interface{}{
		"action": "push",
		"repository": map[string]interface{}{
			"full_name": "foo/bar",
		},
		"organization": map[string]interface{}{
			"login": "foo",
		},
	}

	resp, err = sendevent(t, srv.Client(), srv.URL, "push", nonPrPayload, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}

	// Check that no pullrequest extension was added
	if len(client.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(client.events))
	}

	_, exists := client.events[0].Extensions()["pullrequest"]
	if exists {
		t.Fatal("pullrequest extension should not be present for non-PR events")
	}
}

func TestIsPullRequestMerged(t *testing.T) {
	testCases := []struct {
		name      string
		eventType string
		payload   map[string]interface{}
		expected  bool
	}{
		{
			name:      "merged pull request",
			eventType: "pull_request",
			payload: map[string]interface{}{
				"action": "closed",
				"pull_request": map[string]interface{}{
					"merged": true,
				},
			},
			expected: true,
		},
		{
			name:      "closed but not merged pull request",
			eventType: "pull_request",
			payload: map[string]interface{}{
				"action": "closed",
				"pull_request": map[string]interface{}{
					"merged": false,
				},
			},
			expected: false,
		},
		{
			name:      "open pull request",
			eventType: "pull_request",
			payload: map[string]interface{}{
				"action": "opened",
				"pull_request": map[string]interface{}{
					"merged": false,
				},
			},
			expected: false,
		},
		{
			name:      "not a pull request event",
			eventType: "push",
			payload: map[string]interface{}{
				"action": "closed",
				"pull_request": map[string]interface{}{
					"merged": true,
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function directly
			result := isPullRequestMerged(tc.eventType, tc.payload)

			// Check the result
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestOrgFilter(t *testing.T) {
	secret := []byte("hunter2")
	opts := ServerOptions{
		Secrets:   [][]byte{secret},
		OrgFilter: []string{"org"},
	}
	srv := httptest.NewServer(NewServer(&fakeClient{}, opts))
	defer srv.Close()

	// Send an event with the requested action
	resp, err := sendevent(t, srv.Client(), srv.URL, "pull_request", map[string]interface{}{
		"action": "opened",
	}, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected status: %v", resp.Status)
	}

	resp, err = sendevent(t, srv.Client(), srv.URL, "pull_request", map[string]interface{}{
		"action": "opened",
		"repository": map[string]interface{}{
			"full_name": "org/repo",
		},
		"organization": map[string]interface{}{
			"login": "org",
		},
	}, secret)
	if err != nil {
		t.Fatalf("error sending event: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %v", resp.Status)
	}
}
