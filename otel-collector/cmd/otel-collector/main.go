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
)

func main() {
	configPath := os.Getenv("KO_DATA_PATH") + "/config.yaml"

	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("error checking config.yaml: %v", err)
	}

	log.Printf("config.yaml found at %s", configPath)

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
	srv := &http.Server{Addr: ":31415", Handler: mux}
	mux.HandleFunc("/quitquitquit", func(w http.ResponseWriter, r *http.Request) {
		go srv.Shutdown(ctx)
		cancel()
	})
	go srv.ListenAndServe()

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
