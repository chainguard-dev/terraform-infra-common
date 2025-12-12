/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Copyright 2024 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"chainguard.dev/go-grpc-kit/pkg/duplex"
	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/dispatcher"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/inmem"
)

type envConfig struct {
	Port        int    `env:"PORT, required"`
	Concurrency int    `env:"WORKQUEUE_CONCURRENCY, required"`
	BatchSize   int    `env:"WORKQUEUE_BATCH_SIZE, required"`
	Target      string `env:"WORKQUEUE_TARGET, required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var env envConfig
	envconfig.MustProcess(ctx, &env)

	go httpmetrics.ServeMetrics()

	d := duplex.New(
		env.Port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	client, err := workqueue.NewWorkqueueClient(ctx, env.Target)
	if err != nil {
		clog.FatalContextf(ctx, "failed to create client: %v", err)
	}
	defer client.Close()

	wq := inmem.NewWorkQueue(env.Concurrency)

	eg := errgroup.Group{}
	eg.Go(func() error {
		tick := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
				// Do this in a go routine, so it doesn't block the
				// dispatch loop.
				eg.Go(func() error {
					return dispatcher.Handle(context.WithoutCancel(ctx), wq,
						env.Concurrency, env.BatchSize, dispatcher.ServiceCallback(client))
				})
			}
		}
	})

	eg.Go(func() error {
		workqueue.RegisterWorkqueueServiceServer(d.Server, &enq{wq: wq})
		if err := d.ListenAndServe(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		clog.FromContext(ctx).Errorf("Error group failed: %v", err)
	}
}

type enq struct {
	workqueue.UnimplementedWorkqueueServiceServer

	wq workqueue.Interface
}

func (y *enq) Process(ctx context.Context, req *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
	if err := y.wq.Queue(ctx, req.Key, workqueue.Options{
		Priority: req.Priority,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "Queue() = %v", err)
	}
	return &workqueue.ProcessResponse{}, nil
}

func (y *enq) GetKeyState(ctx context.Context, req *workqueue.GetKeyStateRequest) (*workqueue.KeyState, error) {
	return y.wq.Get(ctx, req.Key)
}
