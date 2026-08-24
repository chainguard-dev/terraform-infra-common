/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package tracesplit

import (
	"context"
	"crypto/rand"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"go.opentelemetry.io/otel"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type config struct {
	related bool
}

// Option configures a Split.
type Option func(*config)

// WithRelatedTraceID ties the new trace's export decision to the caller's, so
// links between them never dangle under sampling.
func WithRelatedTraceID() Option {
	return func(c *config) { c.related = true }
}

// Split starts a span in its own trace, leaving a handoff span of the same
// name in the caller's trace. The two link both ways, and ending the returned
// span ends both. Without a span in ctx it is an ordinary span start.
func Split(ctx context.Context, name string, opts ...Option) (context.Context, oteltrace.Span) {
	var c config
	for _, opt := range opts {
		opt(&c)
	}
	tracer := otel.Tracer("tracesplit")
	caller := oteltrace.SpanContextFromContext(ctx)
	if !caller.IsValid() {
		return tracer.Start(ctx, name)
	}

	_, handoff := tracer.Start(ctx, name)

	tid := randomTraceID()
	if c.related {
		tid = httpmetrics.GenerateRelatedTraceID(ctx)
	}
	// A span context with a trace id but no span id: the SDK adopts the id
	// and still records a true root.
	ctx = oteltrace.ContextWithSpanContext(ctx, oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: tid,
	}))
	newCtx, root := tracer.Start(ctx, name,
		oteltrace.WithLinks(oteltrace.Link{SpanContext: handoff.SpanContext()}))
	handoff.AddLink(oteltrace.Link{SpanContext: root.SpanContext()})

	return newCtx, pair{Span: root, handoff: handoff}
}

type pair struct {
	oteltrace.Span // the new trace's root
	handoff        oteltrace.Span
}

func (p pair) End(opts ...oteltrace.SpanEndOption) {
	p.Span.End(opts...)
	p.handoff.End(opts...)
}

// randomTraceID panics only if crypto/rand fails, which it does not on
// supported platforms. An all-zero (invalid) result is possible in
// principle; the SDK generates a fresh root id when an invalid id is
// planted, so no retry here.
func randomTraceID() oteltrace.TraceID {
	var tid oteltrace.TraceID
	if _, err := rand.Read(tid[:]); err != nil {
		panic(err)
	}
	return tid
}
