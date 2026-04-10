/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
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

type ProbeResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Message   string            `json:"message"`
}

var env = envconfig.MustProcess(context.Background(), &struct {
	Authorization string `env:"AUTHORIZATION,required"`
	TargetURL     string `env:"TARGET_URL,default=https://httpbin.org/status/200"`
	Port          string `env:"PORT,default=8080"`
}{})

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	clog.InfoContextf(ctx, "Starting prober on port %s", env.Port) //nolint:gosec // G706: example app logging operational data
	clog.InfoContextf(ctx, "Target URL: %s", env.TargetURL)        //nolint:gosec // G706: example app logging operational data

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != env.Authorization {
			clog.InfoContextf(r.Context(), "Unauthorized request from %s", r.RemoteAddr) //nolint:gosec // G706: example app logging operational data
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Perform health checks
		checks := make(map[string]string)
		allHealthy := true

		// Check 1: Target URL is reachable
		if err := checkHTTP(env.TargetURL); err != nil {
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
		response := ProbeResponse{
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
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		// Basic health endpoint (no auth required)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Create server with timeouts for security
	server := &http.Server{
		Addr:         ":" + env.Port,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		clog.FatalContextf(ctx, "Server failed: %v", err)
	}
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
