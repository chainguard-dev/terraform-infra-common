/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func main() {
	configPath := os.Getenv("KO_DATA_PATH") + "/config.yaml"

	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("error checking config.yaml: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// run /otel-collector binary with --config config.yaml
	cmd := exec.CommandContext(ctx, "otelcol-contrib", "--config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command but don't wait.
	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start otel-collector: %v", err)
	}

	// Start a server that serves /quitquitquit on port 31415
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:              ":31415",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Minute,
	}
	mux.HandleFunc("/quitquitquit", func(w http.ResponseWriter, r *http.Request) {
		cancel()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Fatalf("failed to shutdown: %v", err)
		}
	})
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
