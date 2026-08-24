/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"context"
	"encoding/binary"
	mrand "math/rand/v2"
	"testing"

	trace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func randomTestTraceID(t *testing.T) oteltrace.TraceID {
	t.Helper()
	var tid oteltrace.TraceID
	for !tid.IsValid() {
		binary.BigEndian.PutUint64(tid[0:8], mrand.Uint64())
		binary.BigEndian.PutUint64(tid[8:16], mrand.Uint64())
	}
	return tid
}

func TestGenerateRelatedTraceID_ExportMatchesCaller(t *testing.T) {
	// A root planted with a related trace id must be exported iff a span
	// carrying the caller's trace id would be.
	const (
		n    = 2000
		rate = 0.25
	)
	inner := &countingProcessor{}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
		trace.WithSpanProcessor(newSamplingProcessor(inner, rate)),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	tr := tp.Tracer("test")

	threshold := uint64(rate * (1 << 63))
	kept, dropped := 0, 0
	for range n {
		caller := randomTestTraceID(t)
		wantExported := binary.BigEndian.Uint64(caller[8:16])>>1 < threshold

		callerCtx := oteltrace.ContextWithSpanContext(t.Context(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID: caller,
			SpanID:  oteltrace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
			Remote:  true,
		}))
		ctx := oteltrace.ContextWithSpanContext(t.Context(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID: GenerateRelatedTraceID(callerCtx),
		}))
		before := inner.count.Load()
		_, root := tr.Start(ctx, "build")
		tid := root.SpanContext().TraceID()
		if got, want := tid[8:16], caller[8:16]; [8]byte(got) != [8]byte(want) {
			t.Fatalf("sampling bytes: got = %x, want = %x", got, want)
		}
		if tid == caller {
			t.Fatalf("derived trace id equals caller id %s; want a distinct trace", caller)
		}
		root.End()

		if gotExported := inner.count.Load() > before; gotExported != wantExported {
			t.Fatalf("export decision for caller %s: got = %t, want = %t", caller, gotExported, wantExported)
		}
		if wantExported {
			kept++
		} else {
			dropped++
		}
	}
	if kept == 0 || dropped == 0 {
		t.Fatalf("degenerate outcome kept=%d dropped=%d — rate=%.2f should mix both over %d traces", kept, dropped, rate, n)
	}
}

func TestGenerateRelatedTraceID_DistinctPerCall(t *testing.T) {
	// Every split needs its own trace, so ids must not repeat for one caller.
	caller := randomTestTraceID(t)
	ctx := oteltrace.ContextWithSpanContext(t.Context(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: caller,
		SpanID:  oteltrace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Remote:  true,
	}))
	if tid1, tid2 := GenerateRelatedTraceID(ctx), GenerateRelatedTraceID(ctx); tid1 == tid2 {
		t.Errorf("trace ids collide across calls: %s", tid1)
	}
}

func TestGenerateRelatedTraceID_NoCallerIsRandom(t *testing.T) {
	// Copying zero bytes would force export at every rate > 0.
	tid := GenerateRelatedTraceID(t.Context())
	if !tid.IsValid() {
		t.Fatalf("invalid trace id %s", tid)
	}
	var zero [8]byte
	if [8]byte(tid[samplingBytesStart:samplingBytesEnd]) == zero {
		t.Errorf("sampling bytes are zero: got = %s, want random fallback", tid)
	}
}
