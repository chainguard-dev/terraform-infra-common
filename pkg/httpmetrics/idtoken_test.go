/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"context"
	"net/http"
	"testing"

	"google.golang.org/api/option"
)

func TestNewIDTokenClient(t *testing.T) {
	aud := "https://example.com"
	ctx := context.Background()
	t.Run("with regular transport", func(t *testing.T) {
		_, err := NewIDTokenClient(ctx, aud, option.WithCredentialsFile("testdata/creds.json"))
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
		_, err := NewIDTokenClient(ctx, aud, option.WithCredentialsFile("testdata/creds.json"))
		if err != nil {
			t.Fatalf("NewIdTokenClient() = %v", err)
		}
	})
}
