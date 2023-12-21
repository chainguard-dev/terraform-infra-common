// package otel provides an OpenTelemetry Collector with components
// suitable to be used on Cloud Run.
//
// This is based on the sidecar example at
// https://cloud.google.com/run/docs/tutorials/custom-metrics-sidecar,
// but with an in-process collector instead of a sidecar.
//
// The main function to use is StartCollectorAsync, which starts the
// collector asynchronously,
//
//	    shutdown, runErr, err := otel.StartCollectorAsync(ctx)
//	    if err != nil {
//		      // handle errors
//	    }
//		   defer shutdown()
//	    // ... optionally wait for runErr and process it.
//
// The prometheus endpoint is expected to be serving at localhost:9090/metrics.
package otel
