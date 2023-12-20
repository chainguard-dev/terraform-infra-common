/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"log"
	"time"

	"github.com/chainguard-dev/terraform-cloudrun-glue/pkg/rotate"
	"github.com/kelseyhightower/envconfig"
	"knative.dev/pkg/signals"
)

type rotateConfig struct {
	Bucket        string        `envconfig:"BUCKET" required:"true"`
	FlushInterval time.Duration `envconfig:"FLUSH_INTERVAL" default:"3m"`
	LogPath       string        `envconfig:"LOG_PATH" required:"true"`
}

func main() {
	var rc rotateConfig
	if err := envconfig.Process("", &rc); err != nil {
		log.Fatalf("Error processing environment: %v", err)
	}

	uploader := rotate.NewUploader(rc.LogPath, rc.Bucket, rc.FlushInterval)

	if err := uploader.Run(signals.NewContext()); err != nil {
		log.Fatalf("Failed to run the uploader: %v", err)
	}
}
