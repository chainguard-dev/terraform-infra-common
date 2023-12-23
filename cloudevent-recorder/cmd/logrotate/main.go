/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/chainguard-dev/terraform-cloudrun-glue/pkg/rotate"
	"github.com/kelseyhightower/envconfig"
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := uploader.Run(ctx); err != nil {
		log.Fatalf("Failed to run the uploader: %v", err)
	}
}
