/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package otelview

import sdkmetric "go.opentelemetry.io/otel/sdk/metric"

var clientRequestDuration = sdkmetric.NewView(
	sdkmetric.Instrument{Name: "http.client.request.duration"},
	sdkmetric.Stream{Name: "otel.http.client.request.duration"},
)

// ClientRequestDuration is an [sdkmetric.View] that renames the
// http.client.request.duration histogram to otel.http.client.request.duration.
// It matches no other instrument.
//
// otelhttp records request durations in a histogram called
// http.client.request.duration. Other HTTP client instrumentation uses the same
// name, because it comes from the OpenTelemetry semantic conventions.
//
// When this histogram reaches Prometheus through the OpenTelemetry exporter,
// its name becomes http_client_request_duration_seconds. httpmetrics'
// transport registers its own histogram with that name. Two metric families
// with the same name and different help text make gathering from the default
// registry fail.
//
// The SDK exports a stream for every view that matches an instrument. A meter
// provider that also changes this histogram's stream, such as its aggregation,
// must change the stream that this view returns. A second view that matches
// the instrument exports it a second time.
func ClientRequestDuration(i sdkmetric.Instrument) (sdkmetric.Stream, bool) {
	return clientRequestDuration(i)
}
