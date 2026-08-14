/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestWithTracePropagation(t *testing.T) {
	// otelhttp injects with the global propagator and records with the global
	// tracer provider; install real ones (recording into sr) so we can check
	// both what goes on the wire and what gets recorded, then restore.
	sr := tracetest.NewSpanRecorder()
	prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})

	var wire string
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		wire = r.Header.Get("traceparent")
	}))
	t.Cleanup(srv.Close)

	get := func(ctx context.Context, rt http.RoundTripper) {
		wire = ""
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		if err != nil {
			t.Fatalf("NewRequestWithContext: %v", err)
		}
		resp, err := (&http.Client{Transport: rt}).Do(req)
		if err != nil {
			t.Fatalf("Do: %v", err)
		}
		_ = resp.Body.Close() // otelhttp ends the client span on body close
	}

	// Default: the traceparent is injected onto the wire.
	get(t.Context(), WrapTransport(http.DefaultTransport))
	if wire == "" {
		t.Errorf("default traceparent: got = %q, want non-empty", wire)
	}

	// Disabled: the client span is still recorded in the caller's trace, but
	// nothing is injected onto the wire.
	sr.Reset()
	ctx, caller := otel.Tracer("test").Start(t.Context(), "caller")
	get(ctx, WrapTransport(http.DefaultTransport, WithTracePropagation(false)))
	caller.End()

	if wire != "" {
		t.Errorf("disabled traceparent: got = %q, want empty", wire)
	}

	var client sdktrace.ReadOnlySpan
	for _, s := range sr.Ended() {
		if s.SpanKind() == trace.SpanKindClient {
			client = s
			break
		}
	}
	if client == nil {
		t.Fatal("recorded client span: got none, want one")
	}
	if got, want := client.SpanContext().TraceID(), caller.SpanContext().TraceID(); got != want {
		t.Errorf("client span trace: got = %s, want = %s", got, want)
	}
}
