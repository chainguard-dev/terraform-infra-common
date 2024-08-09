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
	Bucket        string        `envconfig:"BUCKET" required:"true"`
	FlushInterval time.Duration `envconfig:"FLUSH_INTERVAL" default:"3m"`
	LogPath       string        `envconfig:"LOG_PATH" required:"true"`
}{})

func main() {
	uploader := rotate.NewUploader(env.LogPath, env.Bucket, env.FlushInterval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := uploader.Run(ctx); err != nil {
		clog.Fatalf("Failed to run the uploader: %v", err)
	}
}
