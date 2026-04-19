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
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/sethvargo/go-envconfig"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	ExampleEnv string `env:"EXAMPLE_ENV"`
}{})

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	clog.InfoContextf(ctx, "hello the time is now %v %s", time.Now(), env.ExampleEnv) //nolint:gosec // G706: example app logging operational data
}
