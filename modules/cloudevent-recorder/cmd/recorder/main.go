/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/sethvargo/go-envconfig"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	"github.com/chainguard-dev/terraform-infra-common/pkg/profiler"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	Port    int    `env:"PORT, default=8080"`
	LogPath string `env:"LOG_PATH, required"`
}{})

func main() {
	profiler.SetupProfiler()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	c, err := mce.NewClientHTTP("ce-recorder", cloudevents.WithPort(env.Port))
	if err != nil {
		clog.Fatalf("failed to create event client, %v", err)
	}
	if err := c.StartReceiver(ctx, func(_ context.Context, event cloudevents.Event) error {
		dir := filepath.Join(env.LogPath, event.Type(), event.Time().Format("2006-01-02"))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		filename := filepath.Join(dir, event.ID())
		if err := os.WriteFile(filename, event.Data(), 0600); err != nil {
			clog.Warnf("failed to write file %s; %v", filename, err)
			if err := os.RemoveAll(filename); err != nil {
				clog.Warnf("failed to remove failed write file: %s; %v", filename, err)
			}
			return err
		}
		return nil
	}); err != nil {
		clog.Fatalf("failed to start event receiver, %v", err)
	}
}
