/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package valkey_test

import (
	"crypto/x509"
	"testing"

	"github.com/chainguard-dev/terraform-infra-common/pkg/valkey"
)

func TestNewClientRejectsUnresolvedEndpoint(t *testing.T) {
	// Fail before minting a token so a missing Resolve surfaces at startup,
	// not as a TLS or connection error.
	tests := []struct {
		name     string
		endpoint valkey.Endpoint
	}{
		{name: "zero endpoint"},
		{name: "address without roots", endpoint: valkey.Endpoint{Addr: "10.0.0.2:6379"}},
		{name: "roots without address", endpoint: valkey.Endpoint{Roots: x509.NewCertPool()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := valkey.NewClient(t.Context(), tt.endpoint); err == nil {
				t.Fatal("NewClient accepted an unresolved endpoint, want error")
			}
		})
	}
}
