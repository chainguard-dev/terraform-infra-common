/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestSelectExporters(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		onGCP bool
		want  []string
	}{
		{"unset on GCP defaults to gcp", "", true, []string{"gcp"}},
		{"unset off GCP defaults to nothing", "", false, nil},
		{"explicit none", "none", false, []string{"none"}},
		{"gcp only", "gcp", false, []string{"gcp"}},
		{"otlp only", "otlp", false, []string{"otlp"}},
		{"fan-out", "gcp,otlp", true, []string{"gcp", "otlp"}},
		{"trims whitespace", " gcp , otlp ", true, []string{"gcp", "otlp"}},
		{"drops empty entries", "gcp,,otlp", true, []string{"gcp", "otlp"}},
		{"unknown names passed through for caller to warn on", "gcp,foo", true, []string{"gcp", "foo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_EXPORTER", tt.env)
			if diff := cmp.Diff(tt.want, selectExporters(tt.onGCP)); diff != "" {
				t.Errorf("selectExporters(onGCP=%v) mismatch (-want, +got):\n%s", tt.onGCP, diff)
			}
		})
	}
}

func TestGCPSamplingRate(t *testing.T) {
	tests := []struct {
		env  string
		want float64
	}{
		{"", 0.1},
		{"0.5", 0.5},
		{"1", 1.0},
		{"0", 0.0},
		{"garbage", 0.1},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("OTEL_TRACE_SAMPLING_RATE", tt.env)
			if got := gcpSamplingRate(); got != tt.want {
				t.Errorf("gcpSamplingRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// countingProcessor counts OnEnd invocations. Safe for concurrent use.
type countingProcessor struct{ count atomic.Int64 }

func (c *countingProcessor) OnStart(context.Context, trace.ReadWriteSpan) {}
func (c *countingProcessor) OnEnd(trace.ReadOnlySpan)                     { c.count.Add(1) }
func (c *countingProcessor) Shutdown(context.Context) error               { return nil }
func (c *countingProcessor) ForceFlush(context.Context) error             { return nil }

func TestNewSamplingProcessor_RateFullUnwraps(t *testing.T) {
	inner := &countingProcessor{}
	if sp := newSamplingProcessor(inner, 1.0); sp != trace.SpanProcessor(inner) {
		t.Fatalf("rate >= 1.0 should return inner processor unwrapped; got %T", sp)
	}
}

func TestSamplingSpanProcessor_ZeroRateDropsAll(t *testing.T) {
	inner := &countingProcessor{}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(newSamplingProcessor(inner, 0.0)),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tr := tp.Tracer("test")
	for range 100 {
		_, span := tr.Start(t.Context(), "op")
		span.End()
	}
	if got := inner.count.Load(); got != 0 {
		t.Errorf("rate=0 should drop every span; got count=%d", got)
	}
}

func TestSamplingSpanProcessor_RateMatchesRoughly(t *testing.T) {
	const (
		n    = 20000
		rate = 0.25
		tol  = 0.02 // trace-ID hash is uniform; ±2% over 20k is plenty.
	)
	inner := &countingProcessor{}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(newSamplingProcessor(inner, rate)),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tr := tp.Tracer("test")
	for range n {
		_, span := tr.Start(t.Context(), "op")
		span.End()
	}
	got := float64(inner.count.Load()) / float64(n)
	if got < rate-tol || got > rate+tol {
		t.Errorf("sampled fraction = %.3f, want %.3f ± %.3f", got, rate, tol)
	}
}

func TestSamplingSpanProcessor_TraceIsAllOrNothing(t *testing.T) {
	// Every span in a trace shares one TraceID, so the ratio check is
	// deterministic across the trace: either every span is forwarded or none
	// are. A bypass that only fires on the first local span would keep the
	// root while dropping its local descendants, producing orphan roots in
	// the backend — the exact outcome this processor must not produce.
	const (
		n    = 5000
		rate = 0.25
	)
	inner := &countingProcessor{}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
		trace.WithSpanProcessor(newSamplingProcessor(inner, rate)),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tr := tp.Tracer("test")
	kept, dropped := 0, 0
	for range n {
		before := inner.count.Load()
		ctx, root := tr.Start(t.Context(), "root")
		_, child := tr.Start(ctx, "child")
		_, grandchild := tr.Start(ctx, "grandchild")
		grandchild.End()
		child.End()
		root.End()
		switch delta := inner.count.Load() - before; delta {
		case 0:
			dropped++
		case 3:
			kept++
		default:
			t.Fatalf("trace produced %d forwarded spans; want 0 (all dropped) or 3 (all kept)", delta)
		}
	}
	if kept+dropped != n {
		t.Fatalf("kept=%d + dropped=%d != n=%d", kept, dropped, n)
	}
	if kept == 0 || dropped == 0 {
		t.Fatalf("degenerate outcome kept=%d dropped=%d — rate=%.2f should mix both over %d traces", kept, dropped, rate, n)
	}
}

func TestLLMSpanFilter_ForwardsOnlyGenAISpans(t *testing.T) {
	// The filter must drop infra spans (no gen_ai.* attributes) and forward
	// LLM spans (any gen_ai.* attribute) so evaluation backends like
	// Braintrust don't fill up with noise.
	inner := &countingProcessor{}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(&llmSpanFilterProcessor{inner: inner}),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tr := tp.Tracer("test")

	// Span 1: no attributes at all — drop.
	_, s1 := tr.Start(t.Context(), "no-attrs")
	s1.End()

	// Span 2: infra attribute (http.method) — drop.
	_, s2 := tr.Start(t.Context(), "http-span")
	s2.SetAttributes(attribute.String("http.method", "GET"))
	s2.End()

	// Span 3: gen_ai.request.model — keep.
	_, s3 := tr.Start(t.Context(), "llm-request")
	s3.SetAttributes(attribute.String("gen_ai.request.model", "gemini-2.5-flash"))
	s3.End()

	// Span 4: gen_ai.usage.input_tokens — keep.
	_, s4 := tr.Start(t.Context(), "llm-usage")
	s4.SetAttributes(attribute.Int("gen_ai.usage.input_tokens", 123))
	s4.End()

	// Span 5: mixed attrs where at least one is gen_ai.* — keep.
	_, s5 := tr.Start(t.Context(), "llm-mixed")
	s5.SetAttributes(
		attribute.String("http.method", "POST"),
		attribute.String("gen_ai.response.model", "gemini-2.5-flash"),
	)
	s5.End()

	if got, want := inner.count.Load(), int64(3); got != want {
		t.Errorf("inner processor received %d spans; want %d (only gen_ai.* spans should forward)", got, want)
	}
}

func TestTracerOptions_Matrix(t *testing.T) {
	// Exercises the off-GCP path (no metadata server) to verify the exporter
	// list drives the number of span processors without actually reaching
	// Cloud Trace.
	tests := []struct {
		name           string
		env            map[string]string
		wantProcessors int
	}{
		{
			name:           "none drops all exporters",
			env:            map[string]string{"OTEL_TRACES_EXPORTER": "none"},
			wantProcessors: 0,
		},
		{
			name: "otlp only (bare)",
			env: map[string]string{
				"OTEL_TRACES_EXPORTER":               "otlp",
				"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://localhost:4318",
			},
			wantProcessors: 1,
		},
		{
			name: "otlp named single",
			env: map[string]string{
				"OTEL_TRACES_EXPORTER":                          "otlp/braintrust",
				"OTEL_EXPORTER_OTLP_TRACES_BRAINTRUST_ENDPOINT": "https://bt.example/v1/traces",
			},
			wantProcessors: 1,
		},
		{
			name: "otlp named fan-out to two backends",
			env: map[string]string{
				"OTEL_TRACES_EXPORTER":                          "otlp/braintrust,otlp/langfuse",
				"OTEL_EXPORTER_OTLP_TRACES_BRAINTRUST_ENDPOINT": "https://bt.example/v1/traces",
				"OTEL_EXPORTER_OTLP_TRACES_LANGFUSE_ENDPOINT":   "https://lf.example/v1/traces",
			},
			wantProcessors: 2,
		},
		{
			name: "otlp named missing endpoint is skipped, not fatal",
			env: map[string]string{
				"OTEL_TRACES_EXPORTER":                          "otlp/braintrust,otlp/langfuse",
				"OTEL_EXPORTER_OTLP_TRACES_BRAINTRUST_ENDPOINT": "https://bt.example/v1/traces",
				// LANGFUSE endpoint deliberately unset.
			},
			wantProcessors: 1,
		},
		{
			name:           "invalid name in entry logged and skipped",
			env:            map[string]string{"OTEL_TRACES_EXPORTER": "otlp/../../etc/passwd"},
			wantProcessors: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			opts := tracerOptions(t.Context())
			// Two non-processor opts: WithResource + WithSampler.
			if got := len(opts) - 2; got != tt.wantProcessors {
				t.Errorf("tracerOptions produced %d span processors, want %d (total opts=%d)", got, tt.wantProcessors, len(opts))
			}
		})
	}
}

func TestParseExporterEntry(t *testing.T) {
	tests := []struct {
		entry    string
		wantKind string
		wantName string
	}{
		{"gcp", "gcp", ""},
		{"otlp", "otlp", ""},
		{"none", "none", ""},
		{"otlp/braintrust", "otlp", "braintrust"},
		{"otlp/lang-fuse_1", "otlp", "lang-fuse_1"},
		{"otlp/", "", ""},           // empty name → invalid
		{"otlp/../etc", "", ""},     // path traversal → invalid
		{"otlp/with space", "", ""}, // spaces → invalid
	}
	for _, tt := range tests {
		t.Run(tt.entry, func(t *testing.T) {
			k, n := parseExporterEntry(tt.entry)
			if k != tt.wantKind || n != tt.wantName {
				t.Errorf("parseExporterEntry(%q) = (%q, %q), want (%q, %q)", tt.entry, k, n, tt.wantKind, tt.wantName)
			}
		})
	}
}

func TestParseOTLPHeaders(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    map[string]string
		wantErr bool
	}{
		{"empty", "", map[string]string{}, false},
		{"single", "Authorization=Bearer%20abc", map[string]string{"Authorization": "Bearer abc"}, false},
		{"multi with whitespace", " x-api-key=secret , tenant=acme ",
			map[string]string{"x-api-key": "secret", "tenant": "acme"}, false},
		{"trailing comma ignored", "a=b,", map[string]string{"a": "b"}, false},
		{"missing equals", "no-value", nil, true},
		{"empty key", "=value", nil, true},
		{"bad percent encoding", "a=%ZZ", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOTLPHeaders(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("parseOTLPHeaders(%q) mismatch (-want, +got):\n%s", tt.in, diff)
			}
		})
	}
}
