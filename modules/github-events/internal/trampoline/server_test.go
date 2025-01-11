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
	"github.com/google/go-github/v61/github"
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
	impl := NewServer(client, [][]byte{
		[]byte("badsecret"), // This secret should be ignored
		secret,
	}, nil, nil)
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
			Subject:         github.String("org/repo"),
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
	srv := httptest.NewServer(NewServer(&fakeClient{}, nil, nil, nil))
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
	srv := httptest.NewServer(NewServer(&fakeClient{}, [][]byte{secret}, []string{"doesnotmatch"}, nil))
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
	srv := httptest.NewServer(NewServer(&fakeClient{}, [][]byte{secret}, nil, []string{"1234"}))
	defer srv.Close()

	// Send an event with the requested action
	resp, err := sendevent(t, srv.Client(), srv.URL, "check_run", map[string]interface{}{
		"action": "requested",
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
