/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"net/http"
	"testing"

	"google.golang.org/api/option"
)

func TestNewIDTokenClient(t *testing.T) {
	aud := "https://example.com"
	t.Run("with regular transport", func(t *testing.T) {
		_, err := NewIDTokenClient(t.Context(), aud, option.WithAuthCredentialsFile(option.ExternalAccount, "testdata/creds.json"))
		if err != nil {
			t.Fatalf("NewIDTokenClient() = %v", err)
		}
	})
	t.Run("with wrapped transport", func(t *testing.T) {
		prev := http.DefaultTransport
		http.DefaultTransport = WrapTransport(http.DefaultTransport)
		defer func() {
			http.DefaultTransport = prev
		}()
		_, err := NewIDTokenClient(t.Context(), aud, option.WithAuthCredentialsFile(option.ExternalAccount, "testdata/creds.json"))
		if err != nil {
			t.Fatalf("NewIDTokenClient() = %v", err)
		}
	})
	// Reproduces the prober panic: MetricsTransport wrapping a custom
	// RoundTripper that wraps the real *http.Transport.
	t.Run("with unwrappable transport inside metrics transport", func(t *testing.T) {
		prev := http.DefaultTransport
		http.DefaultTransport = WrapTransport(&unwrappableTransport{
			inner: http.DefaultTransport,
		})
		defer func() {
			http.DefaultTransport = prev
		}()
		_, err := NewIDTokenClient(t.Context(), aud, option.WithAuthCredentialsFile(option.ExternalAccount, "testdata/creds.json"))
		if err != nil {
			t.Fatalf("NewIDTokenClient() = %v", err)
		}
	})
	// Verifies fallback when the wrapper doesn't implement Unwrap.
	t.Run("with opaque transport inside metrics transport", func(t *testing.T) {
		prev := http.DefaultTransport
		http.DefaultTransport = WrapTransport(&opaqueTestTransport{
			inner: http.DefaultTransport,
		})
		defer func() {
			http.DefaultTransport = prev
		}()
		_, err := NewIDTokenClient(t.Context(), aud, option.WithAuthCredentialsFile(option.ExternalAccount, "testdata/creds.json"))
		if err != nil {
			t.Fatalf("NewIDTokenClient() = %v", err)
		}
	})
}
