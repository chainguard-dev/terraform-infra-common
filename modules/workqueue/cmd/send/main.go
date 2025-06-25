// Copyright 2024 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/rand"
	"flag"
	"log"
	"log/slog"
	"math/big"
	"net/url"
	"os"
	"os/signal"
	"runtime"

	delegate "chainguard.dev/go-grpc-kit/pkg/options"
	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx = clog.WithLogger(ctx, clog.New(slog.Default().Handler()))

	httpTarget := flag.String("target", "", "The target to send work to.")
	requests := flag.Int("requests", 100000, "The number of requests to send.")
	rng := flag.Int64("range", 1000, "The range of keys to send.")
	flag.Parse()

	uri, err := url.Parse(*httpTarget)
	if err != nil {
		log.Panicf("failed to parse URI: %v", err)
	}
	target, opts := delegate.GRPCOptions(*uri)

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		log.Panicf("failed to connect to the server: %v", err)
	}
	defer conn.Close()
	client := workqueue.NewWorkqueueServiceClient(conn)

	eg := errgroup.Group{}
	eg.SetLimit(5 * runtime.GOMAXPROCS(0))
	for i := 0; i < *requests; i++ {
		i := i
		eg.Go(func() error {
			bi, err := rand.Int(rand.Reader, big.NewInt(*rng))
			if err != nil {
				return err
			}
			clog.InfoContextf(ctx, "Sending request %d: %s", i, bi.String())
			_, err = client.Process(ctx, &workqueue.ProcessRequest{Key: bi.String()})
			return err
		})
	}
	if err := eg.Wait(); err != nil {
		log.Panicf("failed to send all requests: %v", err)
	}
}
