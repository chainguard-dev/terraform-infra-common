/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/storage"
	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/gcs"
	"github.com/sethvargo/go-envconfig"
)

type envConfig struct {
	Mode        string `env:"WORKQUEUE_MODE, required"`
	Bucket      string `env:"WORKQUEUE_BUCKET, required"`
	Concurrency int    `env:"WORKQUEUE_CONCURRENCY, default=100"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var env envConfig
	envconfig.MustProcess(ctx, &env)

	go httpmetrics.ServeMetrics()

	if env.Mode != "gcs" {
		clog.FatalContextf(ctx, "Unsupported mode: %q, only 'gcs' is supported", env.Mode)
	}

	// Create GCS client and workqueue
	cl, err := storage.NewClient(ctx)
	if err != nil {
		clog.FatalContextf(ctx, "Failed to create storage client: %v", err)
	}
	defer cl.Close()

	wq := gcs.NewWorkQueue(cl.Bucket(env.Bucket), env.Concurrency)

	// Enumerate to get dead-lettered keys
	_, _, deadLettered, err := wq.Enumerate(ctx)
	if err != nil {
		clog.FatalContextf(ctx, "Failed to enumerate workqueue: %v", err)
	}

	if len(deadLettered) == 0 {
		clog.InfoContext(ctx, "No dead-lettered keys found")
		return
	}

	clog.InfoContextf(ctx, "Found %d dead-lettered keys to reenqueue", len(deadLettered))

	var reenqueued, failed int
	for _, dlk := range deadLettered {
		key := dlk.Name()

		clog.InfoContextf(ctx, "Reenqueuing key: %s (failed at: %v, attempts: %d)",
			key, dlk.GetFailedTime(), dlk.GetAttempts())

		// Queue the key directly with the original priority
		if err := wq.Queue(ctx, key, workqueue.Options{Priority: dlk.Priority()}); err != nil {
			clog.ErrorContextf(ctx, "Failed to reenqueue key %s: %v", key, err)
			failed++
			continue
		}

		reenqueued++
		clog.InfoContextf(ctx, "Successfully reenqueued key: %s", key)
	}

	clog.InfoContextf(ctx, "Reenqueue complete: %d reenqueued, %d failed", reenqueued, failed)

	if failed > 0 {
		clog.FatalContextf(ctx, "Some keys failed to reenqueue")
	}
}
