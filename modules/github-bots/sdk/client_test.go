/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// roundTripFunc is a tiny http.RoundTripper that records whether it ran.
type roundTripFunc struct {
	called bool
	resp   *http.Response
}

func (rt *roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.called = true
	if rt.resp != nil {
		rt.resp.Request = req
		return rt.resp, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: req}, nil
}

func TestNewClient_DispatchesThroughBaseTransport(t *testing.T) {
	// The httpmetrics wrapper must compose with the supplied base RoundTripper:
	// any request issued through the returned *github.Client should reach the
	// base transport.
	base := &roundTripFunc{}
	c := NewClient(base)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	resp, err := c.Client().Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()

	if !base.called {
		t.Error("expected base RoundTripper to be called via the httpmetrics wrapper")
	}
}

func TestNewClient_TransportWrapped(t *testing.T) {
	// The returned client must not expose the bare base transport: httpmetrics
	// wraps it, so the Transport on the inner *http.Client should be a
	// different value.
	base := &roundTripFunc{}
	c := NewClient(base)

	if got := c.Client().Transport; got == http.RoundTripper(base) {
		t.Error("NewClient returned the bare base transport; expected httpmetrics wrapping")
	}
}
