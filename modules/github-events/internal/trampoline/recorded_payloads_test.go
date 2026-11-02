/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package trampoline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// The fixtures under testdata are payloads GitHub delivered for a real pull
// request, recorded from a webhook delivery. They are kept verbatim so the
// extension logic is exercised against the shape GitHub actually sends.
func TestRecordedPayloadExtensions(t *testing.T) {
	const prURL = "https://github.com/example-owner/example-repo/pull/1"

	tests := []struct {
		name      string
		eventType string
		fixture   string
		// want holds the extensions under test. A key that is absent from want
		// must also be absent from the event.
		want map[string]any
	}{{
		name:      "pull_request opened has a head branch",
		eventType: "pull_request",
		fixture:   "pull_request_opened.json",
		want: map[string]any{
			"pullrequesturl": prURL,
			"headbranch":     "test/issue-comment-events",
			"actortype":      "User",
		},
	}, {
		name:      "pull_request_review has a head branch",
		eventType: "pull_request_review",
		fixture:   "pull_request_review_submitted.json",
		want: map[string]any{
			"pullrequesturl": prURL,
			"headbranch":     "test/issue-comment-events",
			"actortype":      "User",
		},
	}, {
		name:      "pull_request_review_comment has a head branch",
		eventType: "pull_request_review_comment",
		fixture:   "pull_request_review_comment_created.json",
		want: map[string]any{
			"pullrequesturl": prURL,
			"headbranch":     "test/issue-comment-events",
			"actortype":      "User",
		},
	}, {
		name:      "issue_comment on a pull request has no head branch",
		eventType: "issue_comment",
		fixture:   "issue_comment_on_pull_request_created.json",
		want: map[string]any{
			"pullrequesturl": prURL,
			"actortype":      "User",
		},
	}}

	extensions := []string{"pullrequesturl", "headbranch", "issueurl", "actortype"}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := os.ReadFile(filepath.Join("testdata", tc.fixture))
			if err != nil {
				t.Fatalf("reading fixture: %v", err)
			}

			client := &fakeClient{}
			secret := []byte("hunter2")
			srv := httptest.NewServer(NewServer(client, ServerOptions{Secrets: [][]byte{secret}}))
			defer srv.Close()

			resp, err := sendevent(t, srv.Client(), srv.URL, tc.eventType, json.RawMessage(payload), secret)
			if err != nil {
				t.Fatalf("sending event: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status: got = %v, want = %v", resp.StatusCode, http.StatusOK)
			}
			if len(client.events) != 1 {
				t.Fatalf("events: got = %d, want = 1", len(client.events))
			}

			got := make(map[string]any, len(extensions))
			for _, name := range extensions {
				if v, ok := client.events[0].Extensions()[name]; ok {
					got[name] = v
				}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("extensions (-want, +got):\n%s", diff)
			}
		})
	}
}
