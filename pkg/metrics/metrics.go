/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/compute/metadata"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/chainguard-dev/clog"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/detectors/gcp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/api/option"
)

var env struct {
	KnativeServiceName       string `envconfig:"K_SERVICE" default:"unknown"`
	KnativeConfigurationName string `envconfig:"K_CONFIGURATION" default:"unknown"`
	KnativeRevisionName      string `envconfig:"K_REVISION" default:"unknown"`
}

func init() {
	logger := clog.FromContext(context.Background())
	if err := envconfig.Process("", &env); err != nil {
		logger.Warn("Failed to process environment variables", "error", err)
	}
}

// ServeMetrics serves the metrics endpoint if the METRICS_PORT env var is set.
func ServeMetrics(ctx context.Context) {
	logger := clog.FromContext(ctx)

	// Start the metrics server on the metrics port, if defined.
	var env struct {
		MetricsPort int  `envconfig:"METRICS_PORT" default:"2112" required:"true"`
		EnablePprof bool `envconfig:"ENABLE_PPROF" default:"false" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		logger.Errorf("unable to process environment for METRICS_PORT, ENABLE_PPROF: %v", err)
	} else {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", env.MetricsPort),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}

		if env.EnablePprof {
			// pprof handles
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
			mux.Handle("/debug/pprof/block", pprof.Handler("block"))
			mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
			mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
			log.Println("registering handle for /debug/pprof")
		}

		if err := srv.ListenAndServe(); err != nil {
			logger.Errorf("listen and serve for http /metrics: %v", err)
		}
	}
}

var (
	inFlightGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_inflight_requests",
			Help: "A gauge of requests currently being served by the wrapped handler.",
		},
		[]string{"handler", "service_name", "configuration_name", "revision_name"},
	)
	duration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "A histogram of latencies for requests.",
			// TODO: tweak bucket values based on real usage.
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"handler", "method", "service_name", "configuration_name", "revision_name"},
	)
	responseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_size_bytes",
			Help: "A histogram of response sizes for requests.",
			// TODO: tweak bucket values based on real usage.
			Buckets: []float64{200, 500, 900, 1500},
		},
		[]string{"handler", "method", "service_name", "configuration_name", "revision_name"},
	)
	counter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_status",
			Help: "The number of processed events by response code",
		},
		[]string{"handler", "method", "code", "service_name", "configuration_name", "revision_name", "ce_type"},
	)
)

// Handler wraps a given http handler in standard metrics handlers.
func Handler(name string, handler http.Handler) http.Handler {
	labels := prometheus.Labels{
		"handler":            name,
		"service_name":       env.KnativeServiceName,
		"configuration_name": env.KnativeConfigurationName,
		"revision_name":      env.KnativeRevisionName,
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

func tracerOptionsGCP(ctx context.Context) []trace.TracerProviderOption {
	// Else, we upload directly to Cloud Trace.
	traceExporter, err := texporter.New(
		// Avoid infinite recursion in trace uploads
		//   https://github.com/open-telemetry/opentelemetry-go/issues/1928
		texporter.WithTraceClientOptions([]option.ClientOption{option.WithTelemetryDisabled()}),
	)
	if err != nil {
		log.Panicf("tracerOptionsGCP() = %v", err)
	}
	res, err := resource.New(ctx,
		// Use the GCP resource detector to detect information about the GCP platform
		resource.WithDetectors(gcp.NewDetector()),
		// Keep the default detectors
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		log.Panicf("tracerOptionsGCP() = %v", err)
	}
	bsp := trace.NewBatchSpanProcessor(traceExporter)
	return []trace.TracerProviderOption{
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
		// On Cloud Run, this gives fuller traces. We can tune this down
		// in the future if cost becomes an issue.
		trace.WithSampler(trace.AlwaysSample()),
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

// Expected usage:
//
//	defer metrics.SetupTracer(ctx)()
func SetupTracer(ctx context.Context) func() {
	logger := clog.FromContext(ctx)

	traceEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	projectID, _ := metadata.ProjectID()
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
			logger.Infof("Error shutting down tracer provider: %v", err)
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
