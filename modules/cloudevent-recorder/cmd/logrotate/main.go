/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/rotate"
	"github.com/sethvargo/go-envconfig"

	"syscall"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	Bucket        string        `env:"BUCKET"`
	FlushInterval time.Duration `env:"FLUSH_INTERVAL, default=3m"`
	LogPath       string        `env:"LOG_PATH"`
}{})

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if env.Bucket == "" {
		clog.FatalContextf(ctx, "BUCKET is required")
	}
	if env.LogPath == "" {
		clog.FatalContextf(ctx, "LOG_PATH is required")
	}

	uploader := rotate.NewUploader(env.LogPath, env.Bucket, env.FlushInterval)

	if err := uploader.Run(ctx); err != nil {
		clog.FatalContextf(ctx, "Failed to run the uploader: %v", err)
	}
}
