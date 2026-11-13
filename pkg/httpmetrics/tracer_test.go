/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// toKVStrings converts []attribute.KeyValue to []string ("key=value") for
// cmp.Diff comparisons without needing to reach into unexported Value fields.
func toKVStrings(attrs []attribute.KeyValue) []string {
	if attrs == nil {
		return nil
	}
	out := make([]string, 0, len(attrs))
	for _, a := range attrs {
		out = append(out, string(a.Key)+"="+a.Value.AsString())
	}
	return out
}

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
	// LLM spans (any gen_ai.* attribute) so eval-grade backends don't fill
	// up with noise.
	inner := &countingProcessor{}
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(&spanFilterProcessor{inner: inner, keep: hasGenAIAttribute}),
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

// TestOTLPSpanFilter exports through tracerOptions to an OTLP server and
// checks which of three spans arrive.
func TestOTLPSpanFilter(t *testing.T) {
	tests := []struct {
		name  string
		entry string
		opts  []TracerOption
		want  []string
	}{{
		name:  "default keeps only gen_ai spans",
		entry: "otlp",
		want:  []string{"llm-span"},
	}, {
		name:  "nil filter keeps the default",
		entry: "otlp",
		opts:  []TracerOption{WithOTLPSpanFilter(nil)},
		want:  []string{"llm-span"},
	}, {
		name:  "export all keeps every span",
		entry: "otlp",
		opts:  []TracerOption{WithOTLPExportAllSpans()},
		want:  []string{"kept-span", "plain-span", "llm-span"},
	}, {
		name:  "custom predicate is honoured",
		entry: "otlp",
		opts:  []TracerOption{WithOTLPSpanFilter(func(s trace.ReadOnlySpan) bool { return s.Name() == "kept-span" })},
		want:  []string{"kept-span"},
	}, {
		name:  "named entry uses the configured filter",
		entry: "otlp/primary",
		opts:  []TracerOption{WithOTLPSpanFilter(func(s trace.ReadOnlySpan) bool { return s.Name() == "kept-span" })},
		want:  []string{"kept-span"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				mu     sync.Mutex
				bodies bytes.Buffer
			)
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				mu.Lock()
				bodies.Write(b)
				mu.Unlock()
			}))
			t.Cleanup(srv.Close)
			t.Setenv("OTEL_TRACES_EXPORTER", tt.entry)
			t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", srv.URL+"/v1/traces")
			t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PRIMARY_ENDPOINT", srv.URL+"/v1/traces")

			tp := trace.NewTracerProvider(tracerOptions(t.Context(), newTracerConfig(append(tt.opts, NoGCPDetection())...))...)
			tr := tp.Tracer("test")
			_, s1 := tr.Start(t.Context(), "kept-span")
			s1.End()
			_, s2 := tr.Start(t.Context(), "plain-span")
			s2.End()
			_, s3 := tr.Start(t.Context(), "llm-span")
			s3.SetAttributes(attribute.String("gen_ai.request.model", "m"))
			s3.End()
			if err := tp.Shutdown(t.Context()); err != nil {
				t.Fatal(err)
			}

			mu.Lock()
			defer mu.Unlock()
			var got []string
			for _, name := range []string{"kept-span", "plain-span", "llm-span"} {
				if bytes.Contains(bodies.Bytes(), []byte(name)) {
					got = append(got, name)
				}
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("spans exported (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestTracerOptions_Matrix(t *testing.T) {
	// The stubbed metadata probe reports that the process is off GCP. Each case
	// checks how many span processors its exporter list installs, and no case
	// reaches Cloud Trace.
	stubProbeGCP(t)
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
				"OTEL_TRACES_EXPORTER":                       "otlp/primary",
				"OTEL_EXPORTER_OTLP_TRACES_PRIMARY_ENDPOINT": "https://primary.example/v1/traces",
			},
			wantProcessors: 1,
		},
		{
			name: "otlp named fan-out to two backends",
			env: map[string]string{
				"OTEL_TRACES_EXPORTER":                         "otlp/primary,otlp/secondary",
				"OTEL_EXPORTER_OTLP_TRACES_PRIMARY_ENDPOINT":   "https://primary.example/v1/traces",
				"OTEL_EXPORTER_OTLP_TRACES_SECONDARY_ENDPOINT": "https://secondary.example/v1/traces",
			},
			wantProcessors: 2,
		},
		{
			name: "otlp named missing endpoint is skipped, not fatal",
			env: map[string]string{
				"OTEL_TRACES_EXPORTER":                         "otlp/primary,otlp/secondary",
				"OTEL_EXPORTER_OTLP_TRACES_PRIMARY_ENDPOINT":   "https://primary.example/v1/traces",
				"OTEL_EXPORTER_OTLP_TRACES_SECONDARY_ENDPOINT": "",
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
			opts := tracerOptions(t.Context(), tracerConfig{})
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
		{"otlp/primary", "otlp", "primary"},
		{"otlp/snake-case_1", "otlp", "snake-case_1"},
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

func TestParseZoneAttributes(t *testing.T) {
	tests := []struct {
		name string
		zone string
		want []string // "key=value" pairs, nil means nil return
	}{
		{
			name: "standard us zone",
			zone: "us-central1-a",
			want: []string{"cloud.availability_zone=us-central1-a", "cloud.region=us-central1"},
		},
		{
			name: "european zone",
			zone: "europe-west1-b",
			want: []string{"cloud.availability_zone=europe-west1-b", "cloud.region=europe-west1"},
		},
		{
			name: "us-east zone",
			zone: "us-east1-d",
			want: []string{"cloud.availability_zone=us-east1-d", "cloud.region=us-east1"},
		},
		{
			name: "empty zone returns nil",
			zone: "",
			want: nil,
		},
		{
			name: "one-segment zone returns nil",
			zone: "invalid",
			want: nil,
		},
		{
			name: "two-segment zone returns nil",
			zone: "us-central1",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseZoneAttributes(tt.zone)
			if diff := cmp.Diff(tt.want, toKVStrings(got)); diff != "" {
				t.Errorf("parseZoneAttributes(%q) mismatch (-want, +got):\n%s", tt.zone, diff)
			}
		})
	}
}

// TestBuildResource_PartialErrorIsNotFatal checks that when the GCP detector
// returns ErrPartialResource, buildResource keeps the attributes that the
// detector found and does not exit.
//
// The detector returns a partial resource on Cloud Batch. Cloud Batch runs jobs
// on GCE with CLOUD_RUN_JOB set, so the detector reads Cloud Run Job metadata,
// and the GCE metadata server returns 404 for instance/region. The test serves
// the same responses from a local metadata server, and points the detector at
// it through GCE_METADATA_HOST.
func TestBuildResource_PartialErrorIsNotFatal(t *testing.T) {
	project := fmt.Sprintf("project-%x", rand.Uint64())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/computeMetadata/v1/project/project-id":
			fmt.Fprint(w, project)
		case "/computeMetadata/v1/instance/zone":
			fmt.Fprint(w, "projects/123/zones/us-central1-a")
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GCE_METADATA_HOST", srv.Listener.Addr().String())
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
	// Each of these variables changes what the detector finds. The detector
	// checks for bare metal, GKE, Cloud Functions and Cloud Run before Cloud Run
	// Job, prefers K_SERVICE to CLOUD_RUN_JOB for a job's name, and reads a
	// job's execution and task index from CLOUD_RUN_EXECUTION and
	// CLOUD_RUN_TASK_INDEX.
	for _, key := range []string{
		"BMS_PROJECT_ID", "BMS_REGION", "BMS_INSTANCE_ID", "KUBERNETES_SERVICE_HOST",
		"FUNCTION_TARGET", "K_CONFIGURATION", "K_SERVICE", "CLOUD_RUN_EXECUTION",
		"CLOUD_RUN_TASK_INDEX",
	} {
		unsetenv(t, key)
	}
	job := fmt.Sprintf("job-%x", rand.Uint64())
	t.Setenv("CLOUD_RUN_JOB", job)
	svc := stubServiceName(t)

	if _, err := gcp.NewDetector().Detect(t.Context()); !errors.Is(err, resource.ErrPartialResource) {
		t.Fatalf("GCP detector: got error %v, want ErrPartialResource", err)
	}

	got := buildResource(t.Context(), true)
	want, err := resource.New(t.Context(), resource.WithTelemetrySDK(), resource.WithAttributes(
		attribute.String("cloud.provider", "gcp"),
		attribute.String("cloud.platform", "gcp_cloud_run"),
		attribute.String("cloud.account.id", project),
		attribute.String("faas.name", job),
		attribute.String("cloud.availability_zone", "us-central1-a"),
		attribute.String("cloud.region", "us-central1"),
		attribute.String("service.name", svc),
	))
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(toKVStrings(want.Attributes()), toKVStrings(got.Attributes())); diff != "" {
		t.Errorf("buildResource(ctx, onGCP=true) attributes (-want, +got):\n%s", diff)
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

// TestTracerOptions_NoGCPDetection pins that NoGCPDetection skips the
// metadata probe, while exporters named in OTEL_TRACES_EXPORTER still
// install, and that without it the probe still runs.
func TestTracerOptions_NoGCPDetection(t *testing.T) {
	tests := []struct {
		name           string
		opts           []TracerOption
		wantProbes     int64
		wantProcessors int
	}{{
		name:           "default probes the metadata server",
		wantProbes:     1,
		wantProcessors: 1,
	}, {
		name:           "NoGCPDetection does not",
		opts:           []TracerOption{NoGCPDetection()},
		wantProcessors: 1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probes := stubProbeGCP(t)
			t.Setenv("OTEL_TRACES_EXPORTER", "otlp")
			t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://localhost:4318")

			opts := tracerOptions(t.Context(), newTracerConfig(tt.opts...))
			if got := probes.Load(); got != tt.wantProbes {
				t.Errorf("metadata probes: got = %d, want = %d", got, tt.wantProbes)
			}
			// Two non-processor opts: WithResource + WithSampler.
			if got := len(opts) - 2; got != tt.wantProcessors {
				t.Errorf("span processors: got = %d, want = %d", got, tt.wantProcessors)
			}
		})
	}
}

// TestSetupTracerWith_NoGCPDetection runs the configuration a process that
// cannot reach the metadata server uses: no exporter named, and the public
// option. It pins that nothing probes the metadata server and that the
// installed propagator still carries trace context and baggage.
func TestSetupTracerWith_NoGCPDetection(t *testing.T) {
	probes := stubProbeGCP(t)
	origTP, origProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	t.Cleanup(func() {
		otel.SetTracerProvider(origTP)
		otel.SetTextMapPropagator(origProp)
	})
	t.Setenv("OTEL_TRACES_EXPORTER", "")

	shutdown := SetupTracerWith(t.Context(), NoGCPDetection())
	defer shutdown()
	if got := probes.Load(); got != 0 {
		t.Errorf("metadata probes: got = %d, want = 0", got)
	}

	traceID, err := oteltrace.TraceIDFromHex(fmt.Sprintf("%032x", rand.Uint64()|1))
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := oteltrace.SpanIDFromHex(fmt.Sprintf("%016x", rand.Uint64()|1))
	if err != nil {
		t.Fatal(err)
	}
	member, err := baggage.NewMember("k", fmt.Sprintf("v%d", rand.Int64()))
	if err != nil {
		t.Fatal(err)
	}
	bag, err := baggage.New(member)
	if err != nil {
		t.Fatal(err)
	}
	ctx := oteltrace.ContextWithSpanContext(t.Context(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	}))
	ctx = baggage.ContextWithBaggage(ctx, bag)

	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	out := otel.GetTextMapPropagator().Extract(t.Context(), carrier)
	if got := oteltrace.SpanContextFromContext(out).TraceID(); got != traceID {
		t.Errorf("trace ID: got = %s, want = %s", got, traceID)
	}
	if got, want := baggage.FromContext(out).Member("k").Value(), member.Value(); got != want {
		t.Errorf("baggage: got = %q, want = %q", got, want)
	}
}

// stubProbeGCP replaces the metadata probe for the rest of the test. The stub
// reports that the process is not on GCP and counts its calls, and
// stubProbeGCP returns the counter.
func stubProbeGCP(t *testing.T) *atomic.Int64 {
	t.Helper()
	var probes atomic.Int64
	orig := probeGCP
	probeGCP = func(context.Context) bool {
		probes.Add(1)
		return false
	}
	t.Cleanup(func() { probeGCP = orig })
	return &probes
}

// stubServiceName sets the service name to a random value for the rest of the
// test, and returns it. The package reads K_SERVICE and CLOUD_RUN_JOB once,
// when it is initialized, so t.Setenv cannot change the name.
func stubServiceName(t *testing.T) string {
	t.Helper()
	orig := serviceName
	serviceName = fmt.Sprintf("svc-%x", rand.Uint64())
	t.Cleanup(func() { serviceName = orig })
	return serviceName
}

// stubRevisionName sets K_REVISION to a random value for the rest of the test,
// and returns it.
func stubRevisionName(t *testing.T) string {
	t.Helper()
	rev := fmt.Sprintf("rev-%x", rand.Uint64())
	t.Setenv("K_REVISION", rev)
	return rev
}

// unsetenv unsets key for the rest of the test. The GCP detector treats some
// variables as present even when they are empty, so setting them to "" is not
// enough.
func unsetenv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}

// clearResourceEnv clears OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME,
// which the SDK merges into every resource, so that a test sees only the
// resource under test. It also sets OTEL_TRACES_EXPORTER to none, so the
// test's tracer providers install no exporters.
func clearResourceEnv(t *testing.T) {
	t.Helper()
	t.Setenv("OTEL_TRACES_EXPORTER", "none")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "")
	t.Setenv("OTEL_SERVICE_NAME", "")
}

// spanResource returns the resource of a span from a tracer provider built
// with opts.
func spanResource(t *testing.T, opts ...TracerOption) *resource.Resource {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(append(tracerOptions(t.Context(), newTracerConfig(opts...)), trace.WithSpanProcessor(sr))...)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	_, span := tp.Tracer("test").Start(t.Context(), "span")
	span.End()
	return sr.Ended()[0].Resource()
}

// TestResource checks that Resource returns the resource that a tracer
// provider built with the same options puts on its spans, and how many times
// Resource probes the metadata server.
func TestResource(t *testing.T) {
	passed := resource.NewSchemaless(attribute.String("service.name", fmt.Sprintf("passed-%x", rand.Uint64())))
	for _, tt := range []struct {
		name       string
		opts       []TracerOption
		wantProbes int64
	}{{
		name:       "default probes once",
		wantProbes: 1,
	}, {
		name: "NoGCPDetection does not probe",
		opts: []TracerOption{NoGCPDetection()},
	}, {
		name: "WithResource returns it without probing",
		opts: []TracerOption{WithResource(passed)},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			clearResourceEnv(t)
			probes := stubProbeGCP(t)
			stubServiceName(t)

			got := Resource(t.Context(), tt.opts...)
			if got := probes.Load(); got != tt.wantProbes {
				t.Errorf("metadata probes: got = %d, want = %d", got, tt.wantProbes)
			}
			if want := spanResource(t, tt.opts...); !got.Equal(want) {
				t.Errorf("Resource() = %v, want the spans' resource %v", got, want)
			}
		})
	}
}

// TestResourceServiceName checks that Resource takes service.name from
// K_SERVICE or CLOUD_RUN_JOB, as the spans' resource does.
func TestResourceServiceName(t *testing.T) {
	clearResourceEnv(t)
	stubProbeGCP(t)
	want := stubServiceName(t)

	if v, _ := Resource(t.Context()).Set().Value("service.name"); v.AsString() != want {
		t.Errorf("Resource() service.name = %q, want %q", v.AsString(), want)
	}
}

// TestTracerConfigOnGCP checks that the metadata probe decides whether the
// process runs on GCP, even when WithResource passes a resource that says it
// does, and that NoGCPDetection skips the probe. The stubbed probe reports
// that the process is off GCP.
func TestTracerConfigOnGCP(t *testing.T) {
	for _, tt := range []struct {
		name       string
		opts       []TracerOption
		wantProbes int64
	}{{
		name:       "default probes",
		wantProbes: 1,
	}, {
		name: "NoGCPDetection does not probe",
		opts: []TracerOption{NoGCPDetection()},
	}, {
		name:       "WithResource still probes",
		opts:       []TracerOption{WithResource(resource.NewSchemaless(attribute.String("cloud.provider", "gcp")))},
		wantProbes: 1,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			probes := stubProbeGCP(t)
			if newTracerConfig(tt.opts...).onGCP(t.Context()) {
				t.Error("onGCP: got = true, want = false")
			}
			if got := probes.Load(); got != tt.wantProbes {
				t.Errorf("metadata probes: got = %d, want = %d", got, tt.wantProbes)
			}
		})
	}
}

// TestWithResource checks that the spans' resource is the resource passed to
// WithResource.
func TestWithResource(t *testing.T) {
	clearResourceEnv(t)
	stubProbeGCP(t)
	want := resource.NewSchemaless(attribute.String("service.name", fmt.Sprintf("passed-%x", rand.Uint64())))
	if got := spanResource(t, WithResource(want)); !got.Equal(want) {
		t.Errorf("span resource = %v, want %v", got, want)
	}
}
