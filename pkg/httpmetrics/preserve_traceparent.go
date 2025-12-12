/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"net/http"
)

func newPreserveTraceparentTransport(rt http.RoundTripper) http.RoundTripper {
	return &preserveTraceparentTransport{rt}
}

type preserveTraceparentTransport struct {
	http.RoundTripper
}

func preserveTraceparentHeader(r *http.Request) {
	if v := r.Header.Get("traceparent"); v != "" {
		r.Header.Set(OriginalTraceHeader, v)
	}
}

func preserveTraceparentHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		preserveTraceparentHeader(r)
		next.ServeHTTP(w, r)
	})
}

func (pt *preserveTraceparentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	preserveTraceparentHeader(r)
	return pt.RoundTripper.RoundTrip(r)
}
