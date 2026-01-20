/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"chainguard.dev/go-grpc-kit/pkg/duplex"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/hyperqueue"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type envConfig struct {
	Port      int      `env:"PORT, required"`
	ShardURLs []string `env:"SHARD_URLS, required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var env envConfig
	envconfig.MustProcess(ctx, &env)

	go httpmetrics.ServeMetrics()

	backends := make([]workqueue.WorkqueueServiceClient, len(env.ShardURLs))
	for i, url := range env.ShardURLs {
		client, err := workqueue.NewWorkqueueClient(ctx, url)
		if err != nil {
			log.Panicf("Failed to create client for shard %d (%s): %v", i, url, err)
		}
		defer client.Close()
		backends[i] = client
	}

	srv, err := hyperqueue.New(backends)
	if err != nil {
		log.Panicf("Failed to create hyperqueue server: %v", err)
	}

	d := duplex.New(
		env.Port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	workqueue.RegisterWorkqueueServiceServer(d.Server, srv)
	if err := d.ListenAndServe(ctx); err != nil {
		log.Panicf("ListenAndServe() = %v", err)
	}
}
