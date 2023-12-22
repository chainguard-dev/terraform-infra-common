/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	configPath := os.Getenv("KO_DATA_PATH") + "/config.yaml"

	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("error checking config.yaml: %v", err)
	}

	// run /otel-collector binary with --config config.yaml
	cmd := exec.Command("otelcol-contrib", "--config", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
