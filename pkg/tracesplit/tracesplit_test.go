/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package tracesplit

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	trace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// newTestProvider installs a synchronous in-memory provider globally, which
// is where Split resolves its tracer from.
func newTestProvider(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
	})
	return exporter
}

func TestSplit(t *testing.T) {
	exporter := newTestProvider(t)
	tr := otel.Tracer("test")

	cctx, caller := tr.Start(t.Context(), "reconcile")
	callerSC := caller.SpanContext()

	sctx, span := Split(cctx, "apko-build")
	splitSC := oteltrace.SpanContextFromContext(sctx)
	span.End()
	caller.End()

	spans := exporter.GetSpans()
	if len(spans) != 3 {
		t.Fatalf("exported spans: got = %d, want = 3 (root, handoff, caller)", len(spans))
	}
	var root, handoff tracetest.SpanStub
	for _, s := range spans {
		switch {
		case s.SpanContext.TraceID() != callerSC.TraceID():
			root = s
		case s.Name == "apko-build":
			handoff = s
		}
	}
	if root.Parent.IsValid() {
		t.Errorf("split root has parent %s; want a true root", root.Parent.SpanID())
	}
	if root.SpanContext.TraceID() == callerSC.TraceID() {
		t.Error("split root shares the caller's trace; want its own")
	}
	if !splitSC.Equal(root.SpanContext) {
		t.Errorf("returned context carries %v; want the split root", splitSC)
	}
	if handoff.Parent.SpanID() != callerSC.SpanID() {
		t.Errorf("handoff parent: got = %s, want caller %s", handoff.Parent.SpanID(), callerSC.SpanID())
	}
	if len(root.Links) != 1 || !root.Links[0].SpanContext.Equal(handoff.SpanContext) {
		t.Errorf("root links: got = %+v, want one link to the handoff", root.Links)
	}
	if len(handoff.Links) != 1 || !handoff.Links[0].SpanContext.Equal(root.SpanContext) {
		t.Errorf("handoff links: got = %+v, want one link to the root", handoff.Links)
	}
}

func TestSplit_RelatedTraceID(t *testing.T) {
	newTestProvider(t)
	tr := otel.Tracer("test")

	cctx, caller := tr.Start(t.Context(), "reconcile")
	callerTID := caller.SpanContext().TraceID()

	sctx, span := Split(cctx, "apko-build", WithRelatedTraceID())
	defer span.End()
	defer caller.End()

	tid := oteltrace.SpanContextFromContext(sctx).TraceID()
	if tid == callerTID {
		t.Fatal("related split reused the caller's trace id; want a distinct trace")
	}
	if got, want := tid[8:16], callerTID[8:16]; [8]byte(got) != [8]byte(want) {
		t.Errorf("sampling bytes: got = %x, want = %x", got, want)
	}
}

func TestSplit_NoCallerSpan(t *testing.T) {
	exporter := newTestProvider(t)

	_, span := Split(t.Context(), "apko-build")
	span.End()

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("exported spans: got = %d, want = 1 (ordinary start, no handoff)", len(spans))
	}
	if len(spans[0].Links) != 0 {
		t.Errorf("links without a caller: got = %+v, want none", spans[0].Links)
	}
}
