/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/chainguard-dev/clog"
	gcpclog "github.com/chainguard-dev/clog/gcp"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/mileusna/useragent"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sethvargo/go-envconfig"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	prometheusexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/api/option"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	MetricsPort int `env:"METRICS_PORT, default=2112"`

	// https://cloud.google.com/run/docs/container-contract#services-env-vars
	KnativeServiceName  string `env:"K_SERVICE, default=unknown"`
	KnativeRevisionName string `env:"K_REVISION, default=unknown"`
}{})

// SetupMetrics setups a prometheus exporter for otel metrics
//
// Expected usage:
//
//	defer metrics.SetupMetrics(ctx)()
func SetupMetrics(ctx context.Context) func() {
	// OTel → Prometheus exporter (no GCP SDK needed)
	exporter, err := prometheusexporter.New()
	if err != nil {
		clog.FatalContextf(ctx, "prometheusexporter.New() = %v", err)
	}

	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	otel.SetMeterProvider(provider)

	return func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			clog.ErrorContext(ctx, "Error shutting down meter provider", "error", err)
		}
	}
}

// ServeMetrics serves the metrics endpoint if the METRICS_PORT env var is set.
func ServeMetrics() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			ErrorHandling: promhttp.ContinueOnError, // IMPORTANT: This returns partial metrics + logs error
		}))
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", env.MetricsPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go ScrapeDiskUsage(ctx)
	if err := srv.ListenAndServe(); err != nil {
		clog.ErrorContext(ctx, "listen and serve for http /metrics", "error", err)
	}
}

var (
	inFlightGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_inflight_requests",
			Help: "A gauge of requests currently being served by the wrapped handler.",
		},
		[]string{"handler", "service_name", "revision_name", "email"},
	)
	duration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "A histogram of latencies for requests.",
			// TODO: tweak bucket values based on real usage.
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60},
		},
		[]string{"handler", "method", "service_name", "revision_name", "email"},
	)
	responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_size_bytes",
			Help: "A histogram of response sizes for requests.",
			// TODO: tweak bucket values based on real usage.
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{"handler", "method", "service_name", "revision_name", "email"},
	)
	counter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_status",
			Help: "The number of processed events by response code",
		},
		[]string{"handler", "method", "code", "service_name", "revision_name", "ce_type", "email", "user_agent_browser_name", "user_agent_browser_version", "user_agent_os"},
	)
)

func init() {
	// Set the global metric provider to a no-op so that any metrics created from otelgrpc interceptors
	// are disabled to prevent memory leaks.
	// See https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4226
	otel.SetMeterProvider(noop.MeterProvider{})
}

// Handler wraps a given http handler in standard metrics handlers.
func Handler(name string, handler http.Handler) http.Handler {
	verify := extractCloudRunCaller()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		restoreTraceparentHeader(r)

		labels := prometheus.Labels{
			"handler":       name,
			"service_name":  env.KnativeServiceName,
			"revision_name": env.KnativeRevisionName,
			"email":         "unknown",
		}

		if token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "); token != "" {
			if email, ok := verify(r.Context(), token); ok {
				labels["email"] = email
			}
		}

		h := gcpclog.WithCloudTraceContext(promhttp.InstrumentHandlerInFlight(
			inFlightGauge.With(labels),
			promhttp.InstrumentHandlerDuration(
				duration.MustCurryWith(labels),
				instrumentHandlerCounter(
					counter.MustCurryWith(labels),
					promhttp.InstrumentHandlerResponseSize(
						responseSize.MustCurryWith(labels),
						otelhttp.NewHandler(preserveTraceparentHandler(handler), name),
					),
				),
			),
		))
		h.ServeHTTP(w, r)
	})
}

func restoreTraceparentHeader(r *http.Request) {
	for _, k := range []string{
		// If the incoming request has a googclient trace header, use it instead.
		// These are messages coming from pubsub, and the googclient trace header contains
		// the original outgoing span.
		GoogClientTraceHeader,
		// Else, if the incoming request has a original-traceparent header, use it
		// to avoid missing Cloud Run spans.
		OriginalTraceHeader,
	} {
		if v := r.Header.Get(k); v != "" {
			r.Header.Set("traceparent", v)
			return
		}
	}
}

var providerOnce = sync.OnceValues(func() (*oidc.Provider, error) {
	// I expect this takes milliseconds, so 5s is overkill but smaller than forever.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return oidc.NewProvider(ctx, "https://accounts.google.com")
})

func extractCloudRunCaller() func(context.Context, string) (string, bool) {
	provider, err := providerOnce()
	if err != nil {
		// If we are unable to build a provider for Google, then this is likely
		// being used somewhere other than Cloud Run, so fast-path to returning
		// false.
		return func(context.Context, string) (string, bool) {
			return "", false
		}
	}

	verifier := provider.Verifier(&oidc.Config{
		// When on Cloud Run, this is checked by the platform.
		SkipClientIDCheck: true,
	})
	return func(ctx context.Context, token string) (string, bool) {
		tok, err := verifier.Verify(ctx, token)
		if err != nil {
			// If the issuer isn't Google, then this may be a public service
			// with its own auth.
			return "", false
		}
		var claims struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
		}
		if err := tok.Claims(&claims); err == nil {
			return claims.Email, claims.EmailVerified
		}
		return "", false
	}
}

// HandlerFunc wraps a given http handler func in standard metrics handlers.
func HandlerFunc(name string, f func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return Handler(name, http.HandlerFunc(f)).ServeHTTP
}

// Fractions >= 1 will always sample. Fractions < 0 are treated as zero. To
// respect the parent trace's `SampledFlag`, the `TraceIDRatioBased` sampler
// should be used as a delegate of a `Parent` sampler.
//
// Expected usage:
//
//	defer metrics.SetupTracer(ctx)()
func SetupTracer(ctx context.Context) func() {
	tp := trace.NewTracerProvider(tracerOptions(ctx)...)
	otel.SetTracerProvider(tp)

	prp := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prp)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			clog.ErrorContext(ctx, "Error shutting down tracer provider", "error", err)
		}
	}
}

// tracerOptions builds TracerProvider options driven by OTEL_TRACES_EXPORTER,
// a comma-separated list per the OTel spec. Supported entries:
//
//   - "gcp"         — Google Cloud Trace (requires a reachable GCP metadata server).
//   - "otlp"        — OTLP HTTP, configured by the standard OTEL_EXPORTER_OTLP_*
//     variables (OTEL_EXPORTER_OTLP_TRACES_ENDPOINT, _HEADERS, …).
//   - "otlp/<name>" — named OTLP HTTP, configured by OTEL_EXPORTER_OTLP_TRACES_<UPPER>_ENDPOINT
//     and optional OTEL_EXPORTER_OTLP_TRACES_<UPPER>_HEADERS (W3C Baggage format:
//     "key1=value1,key2=value2"). Enables fan-out to multiple OTLP backends
//     (e.g. "otlp/primary,otlp/secondary") when one pair of standard
//     OTEL_EXPORTER_OTLP_TRACES_* vars isn't enough.
//   - "none"        — no exporter.
//
// When unset, defaults to "gcp" on GCP and "none" elsewhere. Listing several
// enables fan-out (e.g. "gcp,otlp/primary,otlp/secondary").
//
// Filtering is applied per exporter, not globally: OTEL_TRACE_SAMPLING_RATE
// (default 0.1) gates the GCP exporter via TraceID-deterministic ratio
// sampling; OTLP exporters receive only spans carrying at least one
// attribute under the gen_ai.* namespace, so evaluation backends see LLM
// traces without infra noise.
func tracerOptions(ctx context.Context) []trace.TracerProviderOption {
	// Bound the metadata probe so non-GCP startup isn't delayed by the
	// metadata client's default retry window (~15s).
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	projectID, _ := metadata.ProjectIDWithContext(probeCtx)
	onGCP := projectID != ""

	opts := []trace.TracerProviderOption{
		trace.WithResource(buildResource(ctx, onGCP)),
		// Record everything at the provider; per-exporter processors apply
		// their own sampling. ParentBased preserves upstream "not sampled"
		// decisions so we don't fabricate spans for traces the caller dropped.
		trace.WithSampler(trace.ParentBased(trace.AlwaysSample())),
	}
	for _, entry := range selectExporters(onGCP) {
		kind, name := parseExporterEntry(entry)
		switch kind {
		case "gcp":
			opts = append(opts, trace.WithSpanProcessor(newGCPProcessor(ctx)))
		case "otlp":
			sp, ok := newOTLPProcessor(ctx, name)
			if ok {
				opts = append(opts, trace.WithSpanProcessor(sp))
			}
		case "none":
		default:
			clog.WarnContextf(ctx, "tracerOptions(): unknown OTEL_TRACES_EXPORTER entry %q", entry)
		}
	}
	return opts
}

// parseExporterEntry splits "otlp/primary" into ("otlp", "primary") and
// "gcp" into ("gcp", ""). Names outside [A-Za-z0-9_-] are rejected by returning
// a kind of "" so the caller logs and skips.
func parseExporterEntry(entry string) (kind, name string) {
	if k, n, ok := strings.Cut(entry, "/"); ok {
		if !exporterNameRe.MatchString(n) {
			return "", ""
		}
		return k, n
	}
	return entry, ""
}

var exporterNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// selectExporters parses OTEL_TRACES_EXPORTER. When unset, defaults to "gcp"
// on GCP and an empty list otherwise.
func selectExporters(onGCP bool) []string {
	v := os.Getenv("OTEL_TRACES_EXPORTER")
	if v == "" {
		if onGCP {
			return []string{"gcp"}
		}
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, s := range parts {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// buildResource constructs the OTel Resource. The GCP detector is used only
// when on GCP; OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME are honoured
// everywhere via resource.WithFromEnv.
func buildResource(ctx context.Context, onGCP bool) *resource.Resource {
	resOpts := []resource.Option{
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(),
	}
	if onGCP {
		resOpts = append(resOpts, resource.WithDetectors(gcp.NewDetector()))
	}
	if env.KnativeServiceName != "unknown" {
		resOpts = append(resOpts, resource.WithAttributes(
			attribute.String("service.name", env.KnativeServiceName),
		))
	}
	res, err := resource.New(ctx, resOpts...)
	if err != nil {
		clog.FatalContextf(ctx, "tracerOptions(); resource.New() = %v", err)
	}
	return res
}

func newGCPProcessor(ctx context.Context) trace.SpanProcessor {
	exporter, err := texporter.New(
		// Avoid infinite recursion in trace uploads
		//   https://github.com/open-telemetry/opentelemetry-go/issues/1928
		texporter.WithTraceClientOptions([]option.ClientOption{option.WithTelemetryDisabled()}),
	)
	if err != nil {
		clog.FatalContextf(ctx, "tracerOptions(); texporter.New() = %v", err)
	}
	return newSamplingProcessor(trace.NewBatchSpanProcessor(exporter), gcpSamplingRate())
}

// newOTLPProcessor builds a BatchSpanProcessor around an OTLP/HTTP exporter,
// wrapped with llmSpanFilterProcessor so only spans carrying at least one
// gen_ai.* attribute (OTel semantic conventions for generative-AI) reach the
// backend. This keeps eval-grade backends focused on LLM traces instead of
// infra noise (OIDC, GitHub API, workqueue spans, …).
//
// When name == "", the exporter reads the standard OTEL_EXPORTER_OTLP_TRACES_*
// and OTEL_EXPORTER_OTLP_* env vars. When name is non-empty, endpoint and
// headers come from OTEL_EXPORTER_OTLP_TRACES_<UPPER(name)>_ENDPOINT and
// _HEADERS; a missing endpoint logs a warning and the exporter is skipped,
// since the bare SDK default (localhost:4318) would silently spam the loopback
// and mask a misconfiguration.
func newOTLPProcessor(ctx context.Context, name string) (trace.SpanProcessor, bool) {
	opts, ok := otlpOptionsForName(ctx, name)
	if !ok {
		return nil, false
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		clog.FatalContextf(ctx, "tracerOptions(); otlptracehttp.New(%q) = %v", name, err)
	}
	return &llmSpanFilterProcessor{inner: trace.NewBatchSpanProcessor(exporter)}, true
}

func otlpOptionsForName(ctx context.Context, name string) ([]otlptracehttp.Option, bool) {
	if name == "" {
		return nil, true
	}
	upper := strings.ToUpper(name)
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_" + upper + "_ENDPOINT")
	if endpoint == "" {
		clog.WarnContextf(ctx, "tracerOptions(): skipping otlp/%s — OTEL_EXPORTER_OTLP_TRACES_%s_ENDPOINT unset", name, upper)
		return nil, false
	}
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(endpoint)}
	if raw := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_" + upper + "_HEADERS"); raw != "" {
		headers, err := parseOTLPHeaders(raw)
		if err != nil {
			clog.WarnContextf(ctx, "tracerOptions(): ignoring OTEL_EXPORTER_OTLP_TRACES_%s_HEADERS: %v", upper, err)
		} else if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}
	}
	return opts, true
}

// parseOTLPHeaders parses the W3C Baggage-style header string used by
// OTEL_EXPORTER_OTLP_*_HEADERS: comma-separated key=value pairs, with values
// URL-encoded per RFC 3986. Matches the SDK's built-in parser for the standard
// env var so named entries behave the same way.
func parseOTLPHeaders(s string) (map[string]string, error) {
	parts := strings.Split(s, ",")
	out := make(map[string]string, len(parts))
	for _, pair := range parts {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("malformed header %q (expected key=value)", pair)
		}
		key := strings.TrimSpace(pair[:eq])
		val, err := url.QueryUnescape(strings.TrimSpace(pair[eq+1:]))
		if err != nil {
			return nil, fmt.Errorf("malformed value for %q: %w", key, err)
		}
		out[key] = val
	}
	return out, nil
}

func gcpSamplingRate() float64 {
	if v := os.Getenv("OTEL_TRACE_SAMPLING_RATE"); v != "" {
		if rate, err := strconv.ParseFloat(v, 64); err == nil {
			return rate
		}
	}
	return 0.1
}

// samplingSpanProcessor applies a per-exporter trace-ID ratio filter before
// handing spans to the wrapped processor. The decision is TraceID-deterministic,
// so every span in a trace is kept or dropped together — no orphan roots.
//
// Remote-sampled parents are not special-cased: a bypass that only fires on the
// first local span would keep that root while the ratio check later dropped its
// local descendants, producing exactly the orphan-subtree outcome this comment
// used to claim to prevent.
type samplingSpanProcessor struct {
	inner     trace.SpanProcessor
	threshold uint64
}

var _ trace.SpanProcessor = (*samplingSpanProcessor)(nil)

func newSamplingProcessor(inner trace.SpanProcessor, rate float64) trace.SpanProcessor {
	if rate >= 1.0 {
		return inner
	}
	if rate <= 0 {
		return &samplingSpanProcessor{inner: inner, threshold: 0}
	}
	return &samplingSpanProcessor{
		inner:     inner,
		threshold: uint64(rate * (1 << 63)),
	}
}

func (s *samplingSpanProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	s.inner.OnStart(ctx, span)
}

func (s *samplingSpanProcessor) OnEnd(span trace.ReadOnlySpan) {
	if s.threshold == 0 {
		return
	}
	tid := span.SpanContext().TraceID()
	x := binary.BigEndian.Uint64(tid[8:16]) >> 1
	if x < s.threshold {
		s.inner.OnEnd(span)
	}
}

func (s *samplingSpanProcessor) Shutdown(ctx context.Context) error {
	return s.inner.Shutdown(ctx)
}

func (s *samplingSpanProcessor) ForceFlush(ctx context.Context) error {
	return s.inner.ForceFlush(ctx)
}

// llmSpanFilterProcessor forwards only spans carrying at least one attribute
// under the gen_ai.* namespace (OTel semantic conventions for generative-AI)
// to the wrapped processor. It is used on the OTLP path so eval-grade
// backends see LLM traces without infra noise (OIDC token exchanges, GitHub
// API calls, workqueue dispatches, …).
//
// The predicate is attribute-prefix-based rather than span-name- or
// tracer-name-based because gen_ai.* is the OTel standard namespace for LLM
// telemetry: any compliant instrumentation emits these attributes, while span
// and tracer names are free to change without breaking spec compliance.
//
// OnStart is forwarded unconditionally so the inner processor (typically a
// BatchSpanProcessor) observes the full span lifecycle. Shutdown and
// ForceFlush are likewise forwarded unconditionally.
type llmSpanFilterProcessor struct {
	inner trace.SpanProcessor
}

var _ trace.SpanProcessor = (*llmSpanFilterProcessor)(nil)

func (f *llmSpanFilterProcessor) OnStart(ctx context.Context, span trace.ReadWriteSpan) {
	f.inner.OnStart(ctx, span)
}

func (f *llmSpanFilterProcessor) OnEnd(span trace.ReadOnlySpan) {
	for _, attr := range span.Attributes() {
		if strings.HasPrefix(string(attr.Key), "gen_ai.") {
			f.inner.OnEnd(span)
			return
		}
	}
}

func (f *llmSpanFilterProcessor) Shutdown(ctx context.Context) error {
	return f.inner.Shutdown(ctx)
}

func (f *llmSpanFilterProcessor) ForceFlush(ctx context.Context) error {
	return f.inner.ForceFlush(ctx)
}

type delegator struct {
	http.ResponseWriter
	Status int
}

func (d *delegator) WriteHeader(status int) {
	d.Status = status
	d.ResponseWriter.WriteHeader(status)
}

func instrumentHandlerCounter(counter *prometheus.CounterVec, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d := &delegator{
			ResponseWriter: w,
			Status:         http.StatusOK,
		}

		next.ServeHTTP(d, r)
		ua := useragent.Parse(r.UserAgent())
		counter.With(prometheus.Labels{
			"method":                     r.Method,
			"code":                       strconv.Itoa(d.Status),
			"ce_type":                    r.Header.Get(CeTypeHeader),
			"user_agent_browser_name":    ua.Name,
			"user_agent_browser_version": ua.VersionNoShort(),
			"user_agent_os":              ua.OSVersionNoShort(),
		}).Inc()
	}
}
