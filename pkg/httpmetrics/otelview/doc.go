/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package otelview provides OpenTelemetry metric views that keep
// OpenTelemetry instruments from colliding with the Prometheus metrics that
// httpmetrics registers.
//
// It is separate from httpmetrics so that a meter provider can use its views
// without running httpmetrics' package initialization, which replaces the
// global meter provider and registers collectors in the default Prometheus
// registry.
package otelview
