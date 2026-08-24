/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"context"
	"crypto/rand"

	oteltrace "go.opentelemetry.io/otel/trace"
)

// samplingBytesStart and samplingBytesEnd delimit the trace-id bytes the GCP
// export filter reads and GenerateRelatedTraceID copies; the two must agree.
const (
	samplingBytesStart = 8
	samplingBytesEnd   = 16
)

// GenerateRelatedTraceID returns a new trace id carrying the same GCP export
// decision as the trace in ctx, for re-rooting work without dangling links
// under sampling. Requires both services to run the same
// OTEL_TRACE_SAMPLING_RATE. Without a span in ctx the id is fully random.
// An all-zero (invalid) result is possible in principle; the SDK generates a
// fresh root id when an invalid id is planted, so no retry here.
func GenerateRelatedTraceID(ctx context.Context) oteltrace.TraceID {
	caller := oteltrace.SpanContextFromContext(ctx).TraceID()
	var tid oteltrace.TraceID
	mustRandom(tid[:])
	if caller.IsValid() {
		copy(tid[samplingBytesStart:samplingBytesEnd], caller[samplingBytesStart:samplingBytesEnd])
	}
	return tid
}

// mustRandom panics only if crypto/rand fails, which it does not on
// supported platforms.
func mustRandom(b []byte) {
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
}
