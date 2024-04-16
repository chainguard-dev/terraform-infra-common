package httpmetrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

// ServeMetrics serves the metrics endpoint if the METRICS_PORT env var is set.
func ServeMetrics() {
	// Start the metrics server on the metrics port, if defined.
	var env struct {
		MetricsPort int  `envconfig:"METRICS_PORT" default:"2112" required:"true"`
		EnablePprof bool `envconfig:"ENABLE_PPROF" default:"false" required:"true"`
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
		[]string{"handler", "service_name", "revision_name"},
	)
	duration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "A histogram of latencies for requests.",
			// TODO: tweak bucket values based on real usage.
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60},
		},
		[]string{"handler", "method", "service_name", "revision_name"},
	)
	responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_size_bytes",
			Help: "A histogram of response sizes for requests.",
			// TODO: tweak bucket values based on real usage.
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{"handler", "method", "service_name", "revision_name"},
	)
	counter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_status",
			Help: "The number of processed events by response code",
		},
		[]string{"handler", "method", "code", "service_name", "revision_name", "ce_type"},
	)
)

// https://cloud.google.com/run/docs/container-contract#services-env-vars
var env struct {
	KnativeServiceName  string `envconfig:"K_SERVICE" default:"unknown"`
	KnativeRevisionName string `envconfig:"K_REVISION" default:"unknown"`
}

func init() {
	if err := envconfig.Process("", &env); err != nil {
		slog.Warn("Failed to process environment variables", "error", err)
	}
}

// Handler wraps a given http handler in standard metrics handlers.
func Handler(name string, handler http.Handler) http.Handler {
	labels := prometheus.Labels{
		"handler":       name,
		"service_name":  env.KnativeServiceName,
		"revision_name": env.KnativeRevisionName,
	}
	return promhttp.InstrumentHandlerInFlight(
		inFlightGauge.With(labels),
		promhttp.InstrumentHandlerDuration(
			duration.MustCurryWith(labels),
			instrumentHandlerCounter(
				counter.MustCurryWith(labels),
				promhttp.InstrumentHandlerResponseSize(
					responseSize.MustCurryWith(labels),
					otelhttp.NewHandler(handler, ""),
				),
			),
		),
	)
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
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		clog.FromContext(ctx).Fatalf("SetupTracer() = %v", err)
	}
	bsp := trace.NewBatchSpanProcessor(traceExporter)
	res := resource.Default()

	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)
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
