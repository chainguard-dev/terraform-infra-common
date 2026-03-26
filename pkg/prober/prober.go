/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package prober

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/chainguard-dev/clog"
)

// Interface is implemented by Chainguard probers to encapsulate their
// probing logic
type Interface interface {
	// Probe performs a single probe and is passed the HTTP request context.
	Probe(context.Context) error
}

// Func is a convenience wrapper for turning a function into an Interface.
type Func func(context.Context) error

// Probe implements Interface
func (pf Func) Probe(ctx context.Context) error {
	return pf(ctx)
}

// Go launches the prober process, and does not return.
// On errors it terminates the process.
func Go(_ context.Context, i Interface) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	authz := os.Getenv("AUTHORIZATION")
	if authz == "" {
		clog.Fatalf("Expected AUTHORIZATION environment variable to be configured.")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); subtle.ConstantTimeCompare([]byte(auth), []byte(authz)) != 1 {
			clog.ErrorContext(r.Context(), "request was not authorized")
			http.Error(w, "not authorized", http.StatusUnauthorized)
			return
		}
		if err := i.Probe(r.Context()); err != nil {
			clog.ErrorContextf(r.Context(), "probe error: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	clog.Fatalf("listen and serve: %v", srv.ListenAndServe())
}
