/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics_test

import (
	"context"
	"net/http"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
)

func ExampleSetupMetrics() {
	ctx := context.Background()
	cleanup := httpmetrics.SetupMetrics(ctx)
	defer cleanup()
}

func ExampleSetupTracer() {
	ctx := context.Background()
	cleanup := httpmetrics.SetupTracer(ctx)
	defer cleanup()
}

func ExampleHandler() {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := httpmetrics.Handler("my-handler", inner)
	_ = h
}

func ExampleHandlerFunc() {
	h := httpmetrics.HandlerFunc("my-handler", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	_ = h
}

func ExampleSetBuckets() {
	httpmetrics.SetBuckets(map[string]string{
		"api.example.com": "example-api",
	})
}

func ExampleSetBucketSuffixes() {
	httpmetrics.SetBucketSuffixes(map[string]string{
		"example.com": "example",
	})
}

func ExampleWrapTransport() {
	t := httpmetrics.WrapTransport(http.DefaultTransport)
	_ = t
}

func ExampleWrapTransport_skipBucketize() {
	t := httpmetrics.WrapTransport(http.DefaultTransport, httpmetrics.WithSkipBucketize(true))
	_ = t
}

func ExampleExtractInnerTransport() {
	wrapped := httpmetrics.WrapTransport(http.DefaultTransport)
	inner := httpmetrics.ExtractInnerTransport(wrapped)
	_ = inner
}
