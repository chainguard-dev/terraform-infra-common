/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package profiler

import (
	"cmp"
	"context"
	"os"

	"cloud.google.com/go/profiler"
	"github.com/sethvargo/go-envconfig"

	"github.com/chainguard-dev/clog"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	EnableProfiler bool `env:"ENABLE_PROFILER, default=false"`
}{})

func SetupProfiler() {
	clog.Debugf("Current Google Cloud Profiler setting (ENABLE_PROFILER = %t)", env.EnableProfiler)
	if env.EnableProfiler {
		// https://docs.cloud.google.com/run/docs/container-contract#env-vars
		service := cmp.Or(os.Getenv("K_SERVICE"), os.Getenv("CLOUD_RUN_JOB"))

		clog.Debugf("Enabling Google Cloud Profiler (ENABLE_PROFILER = %t, Service = %q)", env.EnableProfiler, service)
		if err := profiler.Start(profiler.Config{Service: service}); err != nil {
			clog.Fatalf("failed to start profiler: %v", err)
		}
	}
}
