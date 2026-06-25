/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test_authorizedRequest_correctToken verifies that authorizedRequest accepts
// a token that exactly matches the secret.
func Test_authorizedRequest_correctToken(t *testing.T) {
	const secret = "super-secret-token"
	if !authorizedRequest(secret, secret) {
		t.Errorf("authorizedRequest(%q, %q): got = false, want = true", secret, secret)
	}
}

// Test_authorizedRequest_wrongToken verifies that authorizedRequest rejects
// tokens that do not match the secret, including prefixes, suffixes, and
// completely different values. Several cases use tokens of the same length as
// the secret to exercise the full byte-loop of subtle.ConstantTimeCompare
// (equal-length inputs skip the length check and reach the byte comparison).
func Test_authorizedRequest_wrongToken(t *testing.T) {
	const secret = "super-secret-token" // 18 bytes

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "wrong token",
			token: "wrong-token",
		},
		{
			name:  "prefix of secret",
			token: "super-secret",
		},
		{
			name:  "secret with extra suffix",
			token: secret + "x",
		},
		// The following cases are the same length as the secret (18 bytes) so
		// they reach the byte-comparison loop inside subtle.ConstantTimeCompare.
		// A regression to a short-circuiting comparison would still reject these,
		// but a timing oracle would be re-introduced for equal-length inputs.
		{
			name:  "single differing byte at end",
			token: "super-secret-toke0", // last byte differs
		},
		{
			name:  "single differing byte at start",
			token: "Xuper-secret-token", // first byte differs
		},
		{
			name:  "single differing byte in middle",
			token: "super-Xecret-token", // middle byte differs
		},
		{
			name:  "all bytes differ same length",
			token: "XXXXXXXXXXXXXXXXXX", // all 18 bytes differ
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if authorizedRequest(tc.token, secret) {
				t.Errorf("authorizedRequest(%q, %q): got = true, want = false", tc.token, secret)
			}
		})
	}
}

// Test_authorizedRequest_emptySecret verifies that authorizedRequest always
// returns false when the secret is empty, preventing accidental open access
// when AUTHORIZATION is not configured.
func Test_authorizedRequest_emptySecret(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token with empty secret",
			token: "",
		},
		{
			name:  "non-empty token with empty secret",
			token: "anything",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if authorizedRequest(tc.token, "") {
				t.Errorf("authorizedRequest(%q, %q): got = true, want = false; empty secret must never grant access", tc.token, "")
			}
		})
	}
}

// Test_probeHandler_unauthorized verifies that the real probeHandler rejects
// requests with missing or incorrect Authorization headers.
func Test_probeHandler_unauthorized(t *testing.T) {
	const secret = "test-handler-secret"
	cfg := &config{
		Authorization: secret,
		// TargetURL is intentionally empty: every case returns 401 before
		// checkHTTP runs, so the URL is never contacted.
		TargetURL: "",
		Port:      "8080",
	}
	handler := probeHandler(cfg)

	tests := []struct {
		name       string
		authHeader string
		setHeader  bool
	}{
		{
			name:      "missing authorization header",
			setHeader: false,
		},
		{
			name:       "empty authorization header",
			authHeader: "",
			setHeader:  true,
		},
		{
			name:       "wrong authorization header",
			authHeader: "not-the-secret",
			setHeader:  true,
		},
		{
			name:       "prefix of secret",
			authHeader: "test-handler",
			setHeader:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.setHeader {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("status: got = %d, want = %d", rr.Code, http.StatusUnauthorized)
			}
		})
	}
}

// Test_probeHandler_authorized verifies that the real probeHandler accepts
// requests with the correct Authorization header and returns the expected
// status code. An httptest.Server is used as the target so the test is
// hermetic and does not make real outbound network dials.
func Test_probeHandler_authorized(t *testing.T) {
	const secret = "test-handler-secret"

	tests := []struct {
		name              string
		targetStatus      int
		wantHandlerStatus int
	}{
		{
			name:              "healthy target returns 200",
			targetStatus:      http.StatusOK,
			wantHandlerStatus: http.StatusOK,
		},
		{
			name:              "unhealthy target returns 503",
			targetStatus:      http.StatusInternalServerError,
			wantHandlerStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Start a local test server that returns the configured status.
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.targetStatus)
			}))
			defer target.Close()

			cfg := &config{
				Authorization: secret,
				TargetURL:     target.URL,
				Port:          "8080",
			}
			handler := probeHandler(cfg)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", secret)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantHandlerStatus {
				t.Errorf("status: got = %d, want = %d", rr.Code, tc.wantHandlerStatus)
			}
		})
	}
}
