/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package profiler

import (
	"os"

	"cloud.google.com/go/profiler"
	"github.com/kelseyhightower/envconfig"

	"github.com/chainguard-dev/clog"
)

type envConfig struct {
	EnableProfiler bool `envconfig:"ENABLE_PROFILER" default:"false" required:"false"`
}

func SetupProfiler() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		clog.Fatalf("failed to process env var: %v", err)
	}

	if env.EnableProfiler {
		if err := profiler.Start(profiler.Config{Service: os.Getenv("K_SERVICE")}); err != nil {
			clog.Fatalf("failed to start profiler: %v", err)
		}
	}
}
