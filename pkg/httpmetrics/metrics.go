package httpmetrics

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	gcpclog "github.com/chainguard-dev/clog/gcp"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/api/option"
)

// ServeMetrics serves the metrics endpoint if the METRICS_PORT env var is set.
func ServeMetrics() {
	// Start the metrics server on the metrics port, if defined.
	var env struct {
		MetricsPort int `envconfig:"METRICS_PORT" default:"2112" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		slog.Error("Failed to process environment variables", "error", err)
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", env.MetricsPort),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("listen and serve for http /metrics", "error", err)
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
		[]string{"handler", "method", "code", "service_name", "revision_name", "ce_type", "email"},
	)
)

// https://cloud.google.com/run/docs/container-contract#services-env-vars
var env struct {
	KnativeServiceName  string `envconfig:"K_SERVICE" default:"unknown"`
	KnativeRevisionName string `envconfig:"K_REVISION" default:"unknown"`
}

func init() {
	// Set the global metric provider to a no-op so that any metrics created from otelgrpc interceptors
	// are disabled to prevent memory leaks.
	// See https://github.com/open-telemetry/opentelemetry-go-contrib/issues/4226
	otel.SetMeterProvider(noop.MeterProvider{})

	if err := envconfig.Process("", &env); err != nil {
		slog.Warn("Failed to process environment variables", "error", err)
	}
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

func extractCloudRunCaller() func(context.Context, string) (string, bool) {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
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

// Handler wraps a given http handler func in standard metrics handlers.
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
	traceEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	projectID, _ := metadata.ProjectIDWithContext(ctx)
	var options []trace.TracerProviderOption
	if traceEndpoint == "" && projectID != "" {
		// No trace endpoint provided and we are on GCP.
		options = tracerOptionsGCP(ctx)
	} else {
		// We are either on KinD or GKE.
		options = tracerOptions(ctx)
	}
	tp := trace.NewTracerProvider(options...)
	otel.SetTracerProvider(tp)

	prp := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prp)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("Error shutting down tracer provider", "error", err)
		}
	}
}

func tracerOptionsGCP(ctx context.Context) []trace.TracerProviderOption {
	// Else, we upload directly to Cloud Trace.
	traceExporter, err := texporter.New(
		// Avoid infinite recursion in trace uploads
		//   https://github.com/open-telemetry/opentelemetry-go/issues/1928
		texporter.WithTraceClientOptions([]option.ClientOption{option.WithTelemetryDisabled()}),
	)
	if err != nil {
		log.Panicf("tracerOptionsGCP(); texporter.New() = %v", err)
	}
	res, err := resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		// Keep the default detectors
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		log.Panicf("tracerOptionsGCP(); resource.New() = %v", err)
	}
	bsp := trace.NewBatchSpanProcessor(traceExporter)
	// Default to 10%
	samplingRate := 0.1
	// If OTEL_TRACE_SAMPLING_RATE is set, use that value.
	if v := os.Getenv("OTEL_TRACE_SAMPLING_RATE"); v != "" {
		if rate, err := strconv.ParseFloat(v, 64); err == nil {
			samplingRate = rate
		}
	}
	return []trace.TracerProviderOption{
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(samplingRate))),
	}
}

func tracerOptions(ctx context.Context) []trace.TracerProviderOption {
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		log.Panicf("traceOptions() = %v", err)
	}
	bsp := trace.NewBatchSpanProcessor(traceExporter)
	res := resource.Default()

	return []trace.TracerProviderOption{
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	}
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
			Status:         200,
		}

		next.ServeHTTP(d, r)
		counter.With(prometheus.Labels{
			"method":  r.Method,
			"code":    strconv.Itoa(d.Status),
			"ce_type": r.Header.Get(CeTypeHeader),
		}).Inc()
	}
}
