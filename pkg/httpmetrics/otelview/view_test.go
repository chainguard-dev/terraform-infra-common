/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package otelview

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestClientRequestDuration(t *testing.T) {
	type result struct {
		Stream sdkmetric.Stream
		OK     bool
	}

	tests := []struct {
		name       string
		instrument sdkmetric.Instrument
		want       result
	}{{
		name: "otelhttp",
		instrument: sdkmetric.Instrument{
			Name:        "http.client.request.duration",
			Description: "Duration of HTTP client requests.",
			Kind:        sdkmetric.InstrumentKindHistogram,
			Unit:        "s",
			Scope:       instrumentation.Scope{Name: otelhttp.ScopeName},
		},
		want: result{
			Stream: sdkmetric.Stream{
				Name:        "otel.http.client.request.duration",
				Description: "Duration of HTTP client requests.",
				Unit:        "s",
			},
			OK: true,
		},
	}, {
		name: "other HTTP client instrumentation",
		instrument: sdkmetric.Instrument{
			Name:        "http.client.request.duration",
			Description: "Duration of HTTP client requests.",
			Kind:        sdkmetric.InstrumentKindHistogram,
			Unit:        "s",
			Scope:       instrumentation.Scope{Name: "example.com/client"},
		},
		want: result{
			Stream: sdkmetric.Stream{
				Name:        "otel.http.client.request.duration",
				Description: "Duration of HTTP client requests.",
				Unit:        "s",
			},
			OK: true,
		},
	}, {
		name: "another instrument",
		instrument: sdkmetric.Instrument{
			Name:  "http.server.request.duration",
			Kind:  sdkmetric.InstrumentKindHistogram,
			Unit:  "s",
			Scope: instrumentation.Scope{Name: otelhttp.ScopeName},
		},
		want: result{},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, ok := ClientRequestDuration(tt.instrument)
			if diff := cmp.Diff(tt.want, result{Stream: stream, OK: ok}); diff != "" {
				t.Errorf("ClientRequestDuration (-want +got):\n%s", diff)
			}
		})
	}
}
