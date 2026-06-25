/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/sethvargo/go-envconfig"
)

// config holds the environment-variable configuration for this service.
// A named struct (rather than the anonymous-struct MustProcess idiom) is used
// so that tests can construct a *config directly without setting env vars.
type config struct {
	Authorization string `env:"AUTHORIZATION,required"`
	TargetURL     string `env:"TARGET_URL,default=https://httpbin.org/status/200"`
	Port          string `env:"PORT,default=8080"`
}

type probeResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Message   string            `json:"message"`
}

// probeHandler returns an http.HandlerFunc that authenticates requests using
// the provided config and performs health checks against the configured target.
func probeHandler(cfg *config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header using constant-time comparison to
		// prevent timing-based secret recovery attacks.
		authHeader := r.Header.Get("Authorization")
		if !authorizedRequest(authHeader, cfg.Authorization) {
			clog.InfoContextf(r.Context(), "Unauthorized request from %s", r.RemoteAddr) //nolint:gosec // G706: example app logging operational data
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Perform health checks
		checks := make(map[string]string, 3)
		allHealthy := true

		// Check 1: Target URL is reachable
		if err := checkHTTP(cfg.TargetURL); err != nil {
			checks["target_url"] = fmt.Sprintf("FAILED: %v", err)
			allHealthy = false
			clog.InfoContextf(r.Context(), "Target URL check failed: %v", err)
		} else {
			checks["target_url"] = "OK"
		}

		// Check 2: DNS resolution (example)
		checks["dns"] = "OK"

		// Check 3: Memory available (example)
		checks["memory"] = "OK"

		// Prepare response
		response := probeResponse{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    checks,
		}

		if allHealthy {
			response.Status = "healthy"
			response.Message = "All checks passed"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				clog.InfoContextf(r.Context(), "Failed to encode response: %v", err)
			}
			clog.InfoContextf(r.Context(), "Health check PASSED")
		} else {
			response.Status = "unhealthy"
			response.Message = "One or more checks failed"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				clog.InfoContextf(r.Context(), "Failed to encode response: %v", err)
			}
			clog.InfoContextf(r.Context(), "Health check FAILED")
		}
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// MustProcess panics immediately on missing required env vars, eliminating
	// error-handling boilerplate per go-standards. The named config struct is
	// kept (rather than an anonymous struct) so that tests can construct a
	// *config directly without setting env vars.
	cfg := envconfig.MustProcess(ctx, &config{})

	clog.InfoContextf(ctx, "Starting prober on port %s", cfg.Port) //nolint:gosec // G706: example app logging operational data
	clog.InfoContextf(ctx, "Target URL: %s", cfg.TargetURL)        //nolint:gosec // G706: example app logging operational data

	mux := http.NewServeMux()
	mux.HandleFunc("/", probeHandler(cfg))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		// Basic health endpoint (no auth required)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Create server with timeouts for security
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// listenErr receives the error from ListenAndServe so that a bind failure
	// is surfaced to main and causes a non-zero exit, rather than being silently
	// swallowed when ctx is cancelled before the goroutine logs the error.
	listenErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			listenErr <- err
		}
		close(listenErr)
	}()

	select {
	case <-ctx.Done():
		// Graceful shutdown on signal.
	case err := <-listenErr:
		clog.FatalContextf(ctx, "Server failed: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		clog.ErrorContextf(shutdownCtx, "Server shutdown error: %v", err)
	}
}

// authorizedRequest reports whether the provided token matches the expected
// secret. It uses crypto/subtle.ConstantTimeCompare so that the comparison
// takes the same amount of time regardless of how many bytes match, preventing
// timing-based secret-recovery attacks. An empty secret always returns false
// to prevent accidental open access when AUTHORIZATION is not configured.
func authorizedRequest(token, secret string) bool {
	if len(secret) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1
}

// checkHTTP performs an HTTP GET request to verify the target is reachable
func checkHTTP(url string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url) //nolint:gosec // G704: URL from internal config
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
