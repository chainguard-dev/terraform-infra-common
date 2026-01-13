/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"cloud.google.com/go/storage"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/dispatcher"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/gcs"
)

type envConfig struct {
	Port        int    `env:"PORT, required"`
	Concurrency int    `env:"WORKQUEUE_CONCURRENCY, required"`
	BatchSize   int    `env:"WORKQUEUE_BATCH_SIZE, required"`
	Mode        string `env:"WORKQUEUE_MODE, required"`
	Bucket      string `env:"WORKQUEUE_BUCKET"`
	Target      string `env:"WORKQUEUE_TARGET, required"`
	MaxRetry    int    `env:"WORKQUEUE_MAX_RETRY, default=0"` // 0 means unlimited retries
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var env envConfig
	envconfig.MustProcess(ctx, &env)

	go httpmetrics.ServeMetrics()

	var wq workqueue.Interface
	switch env.Mode {
	case "gcs":
		cl, err := storage.NewClient(ctx)
		if err != nil {
			log.Panicf("Failed to create client: %v", err)
		}
		wq = gcs.NewWorkQueue(cl.Bucket(env.Bucket), env.Concurrency)

		// Launch a go routine in the background to periodically call Enumerate
		// to ensure that each replica surfaces the latest and greatest metrics
		// even if the worker isn't being invoked for fresh work.
		go func() {
			tick := time.NewTicker(30 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					_, _, _, err := wq.Enumerate(ctx)
					if err != nil {
						log.Printf("Failed to enumerate: %v", err)
					}
				}
			}
		}()

	default:
		log.Panicf("Unsupported mode: %q", env.Mode)
	}

	client, err := workqueue.NewWorkqueueClient(ctx, env.Target)
	if err != nil {
		log.Panicf("failed to create client: %v", err)
	}
	defer client.Close()

	h := dispatcher.Handler(wq, env.Concurrency, env.BatchSize, dispatcher.ServiceCallback(client), env.MaxRetry)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", env.Port),
		Handler:           h2c.NewHandler(h, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Panicf("failed to start server: %v", err)
	}
}
