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

type Response struct {
	Message     string    `json:"message"`
	Region      string    `json:"region"`
	Environment string    `json:"environment"`
	Hostname    string    `json:"hostname"`
	Timestamp   time.Time `json:"timestamp"`
	Version     string    `json:"version"`
}

type HealthResponse struct {
	Status string    `json:"status"`
	Region string    `json:"region"`
	Uptime string    `json:"uptime"`
	Time   time.Time `json:"time"`
}

var startTime = time.Now()

func main() {
	// Get configuration from environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = "unknown"
	}

	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	version := os.Getenv("VERSION")
	if version == "" {
		version = "1.0.0"
	}

	log.Printf("Starting server on port %s", port)
	log.Printf("Region: %s", region)
	log.Printf("Environment: %s", environment)
	log.Printf("Log Level: %s", logLevel)
	log.Printf("Version: %s", version)

	// Root handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		response := Response{
			Message:     "Hello from AWS App Runner! 🚀",
			Region:      region,
			Environment: environment,
			Hostname:    hostname,
			Timestamp:   time.Now(),
			Version:     version,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		uptime := time.Since(startTime).Round(time.Second)

		response := HealthResponse{
			Status: "healthy",
			Region: region,
			Uptime: uptime.String(),
			Time:   time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})

	// Readiness check endpoint
	http.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		// Add any readiness checks here (database, external services, etc.)
		// For now, we're always ready
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
			"region": region,
		})
	})

	// Info endpoint with all environment details
	http.HandleFunc("/info", func(w http.ResponseWriter, _ *http.Request) {
		info := map[string]interface{}{
			"version":     version,
			"region":      region,
			"environment": environment,
			"hostname":    hostname,
			"uptime":      time.Since(startTime).Round(time.Second).String(),
			"go_version":  os.Getenv("GO_VERSION"),
			"started_at":  startTime,
		}

		// Only include secrets info (not values!) if they're set
		if os.Getenv("DATABASE_URL") != "" {
			info["has_database_url"] = true
		}
		if os.Getenv("API_KEY") != "" {
			info["has_api_key"] = true
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(info)
	})

	// Simple HTML page for browser testing
	http.HandleFunc("/ui", func(w http.ResponseWriter, _ *http.Request) {
		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>AWS App Runner Demo</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
        }
        .container {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
        }
        h1 { margin: 0 0 10px 0; font-size: 3em; }
        .subtitle { font-size: 1.2em; opacity: 0.9; margin-bottom: 30px; }
        .info { background: rgba(255, 255, 255, 0.1); padding: 20px; border-radius: 10px; margin: 20px 0; }
        .info-row { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid rgba(255, 255, 255, 0.1); }
        .info-row:last-child { border-bottom: none; }
        .label { font-weight: bold; opacity: 0.8; }
        .value { font-family: monospace; }
        .emoji { font-size: 1.5em; margin-right: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Sample AWS App Runner</h1>
        <div class="subtitle">Go service deployment</div>

        <div class="info">
            <div class="info-row">
                <span class="label">Region:</span>
                <span class="value">%s</span>
            </div>
            <div class="info-row">
                <span class="label">Environment:</span>
                <span class="value">%s</span>
            </div>
            <div class="info-row">
                <span class="label">Hostname:</span>
                <span class="value">%s</span>
            </div>
            <div class="info-row">
                <span class="label">Version:</span>
                <span class="value">%s</span>
            </div>
            <div class="info-row">
                <span class="label">Uptime:</span>
                <span class="value">%s</span>
            </div>
            <div class="info-row">
                <span class="label">Time:</span>
                <span class="value">%s</span>
            </div>
        </div>

        <div style="margin-top: 30px; opacity: 0.7; font-size: 0.9em;">
            <p><strong>API Endpoints:</strong></p>
            <ul>
                <li><code>/</code> - JSON response</li>
                <li><code>/health</code> - Health check</li>
                <li><code>/ready</code> - Readiness check</li>
                <li><code>/info</code> - Detailed info</li>
                <li><code>/ui</code> - This page</li>
            </ul>
        </div>
    </div>
</body>
</html>
`, region, environment, hostname, version, time.Since(startTime).Round(time.Second), time.Now().Format(time.RFC3339))

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, html)
	})

	// Start the server with proper timeouts
	addr := ":" + port
	log.Printf("Server listening on %s", addr)
	log.Printf("Endpoints: / /health /ready /info /ui")

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
