/*
Copyright 2026 Chainguard, Inc.
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
	"github.com/chainguard-dev/terraform-infra-common/modules/gcp-incident-events/internal/trampoline"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	"github.com/sethvargo/go-envconfig"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	Port       int    `env:"PORT,default=8080"`
	IngressURI string `env:"EVENT_INGRESS_URI"`
}{})

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if env.IngressURI == "" {
		clog.FatalContextf(ctx, "EVENT_INGRESS_URI is required")
	}

	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()

	ceclient, err := mce.NewClientHTTP("gcp-incident-trampoline", mce.WithTarget(ctx, env.IngressURI)...)
	if err != nil {
		clog.FatalContextf(ctx, "failed to create cloudevents client: %v", err)
	}

	clog.FatalContextf(ctx, "ListenAndServe: %v", (&http.Server{
		Addr:              fmt.Sprintf(":%d", env.Port),
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           httpmetrics.Handler("gcp-incident-trampoline", trampoline.NewServer(ceclient)),
	}).ListenAndServe())
}
