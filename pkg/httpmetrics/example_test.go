/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics_test

import (
	"context"
	"net/http"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func ExampleSetupMetrics() {
	ctx := context.Background()
	cleanup := httpmetrics.SetupMetrics(ctx)
	defer cleanup()
}

func ExampleSetupTracer() {
	ctx := context.Background()
	cleanup := httpmetrics.SetupTracer(ctx)
	defer cleanup()
}

func ExampleHandler() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := httpmetrics.Handler("my-handler", inner)
	_ = h
}

func ExampleHandlerFunc() {
	h := httpmetrics.HandlerFunc("my-handler", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	_ = h
}

func ExampleSetBuckets() {
	httpmetrics.SetBuckets(map[string]string{
		"api.example.com": "example-api",
	})
}

func ExampleSetBucketSuffixes() {
	httpmetrics.SetBucketSuffixes(map[string]string{
		"example.com": "example",
	})
}

func ExampleWrapTransport() {
	t := httpmetrics.WrapTransport(http.DefaultTransport)
	_ = t
}

func ExampleWrapTransport_skipBucketize() {
	t := httpmetrics.WrapTransport(http.DefaultTransport, httpmetrics.WithSkipBucketize(true))
	_ = t
}

func ExampleExtractInnerTransport() {
	wrapped := httpmetrics.WrapTransport(http.DefaultTransport)
	inner := httpmetrics.ExtractInnerTransport(wrapped)
	_ = inner
}

func ExampleGenerateRelatedTraceID() {
	// Re-root work as its own trace whose export decision matches the
	// caller's: plant the related trace id (no span id) and the SDK adopts
	// it as a new root. Real servers receive the caller span context via
	// inbound propagation; the example plants one. See pkg/tracesplit for
	// the packaged version of this pattern.
	ctx := trace.ContextWithRemoteSpanContext(context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
			TraceFlags: trace.FlagsSampled,
			Remote:     true,
		}))

	// Derive and link only when there is a real caller trace.
	var opts []trace.SpanStartOption
	if caller := trace.SpanContextFromContext(ctx); caller.IsValid() {
		ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: httpmetrics.GenerateRelatedTraceID(ctx),
		}))
		opts = append(opts, trace.WithLinks(trace.Link{SpanContext: caller}))
	}
	_, span := otel.Tracer("example").Start(ctx, "build", opts...)
	defer span.End()
}
