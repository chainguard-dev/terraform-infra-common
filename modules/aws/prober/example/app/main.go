/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type ProbeResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Message   string            `json:"message"`
}

func main() {
	// Get the shared authorization secret from environment
	expectedAuth := os.Getenv("AUTHORIZATION")
	if expectedAuth == "" {
		log.Fatal("AUTHORIZATION environment variable not set")
	}

	// Get configuration from environment
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "https://httpbin.org/status/200" // Default to a test endpoint
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting prober on port %s", port)
	log.Printf("Target URL: %s", targetURL)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != expectedAuth {
			log.Printf("Unauthorized request from %s", r.RemoteAddr)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Perform health checks
		checks := make(map[string]string)
		allHealthy := true

		// Check 1: Target URL is reachable
		if err := checkHTTP(targetURL); err != nil {
			checks["target_url"] = fmt.Sprintf("FAILED: %v", err)
			allHealthy = false
			log.Printf("Target URL check failed: %v", err)
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
				log.Printf("Failed to encode response: %v", err)
			}
			log.Println("Health check PASSED")
		} else {
			response.Status = "unhealthy"
			response.Message = "One or more checks failed"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Failed to encode response: %v", err)
			}
			log.Println("Health check FAILED")
		}
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		// Basic health endpoint (no auth required)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Create server with timeouts for security
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

// checkHTTP performs an HTTP GET request to verify the target is reachable
func checkHTTP(url string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
