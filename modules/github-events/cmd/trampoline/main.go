/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/secrets"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/internal/trampoline"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	"github.com/sethvargo/go-envconfig"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	Port       int    `env:"PORT, default=8080"`
	IngressURI string `env:"EVENT_INGRESS_URI, required"`
	// Note: any environment variable starting with "WEBHOOK_SECRET" will be loaded as as a webhook secret to be checked.
	WebhookSecret string `env:"WEBHOOK_SECRET"`
	// If set, any matching webhook IDs will only pass through the event if the event is a requested event:
	// - check_run.requested_action
	// - check_run.rerequested
	// - check_suite.requested
	// - check_suite.rerequested
	// This is intended to be a temporary measure to allow multiple webhooks to be sent sent to the same endpoint'
	// while avoiding duplicate events.
	RequestedOnly []string `env:"REQUESTED_ONLY_WEBHOOK_ID"`
	// If set, only events from the specified webhook IDs will be processed.
	WebhookID []string `env:"WEBHOOK_ID"`
}{})

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Get all secrets from the environment.
	secrets := secrets.LoadFromEnv(ctx)

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	ceclient, err := mce.NewClientHTTP("trampoline", mce.WithTarget(ctx, env.IngressURI)...)
	if err != nil {
		clog.FatalContextf(ctx, "failed to create cloudevents client: %v", err)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", env.Port),
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           httpmetrics.Handler("trampoline", trampoline.NewServer(ceclient, secrets, env.WebhookID, env.RequestedOnly)),
	}
	clog.FatalContextf(ctx, "ListenAndServe: %v", srv.ListenAndServe())
}
