/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package profiler

import (
	"context"
	"os"

	"cloud.google.com/go/profiler"
	"github.com/sethvargo/go-envconfig"

	"github.com/chainguard-dev/clog"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	EnableProfiler bool `envconfig:"ENABLE_PROFILER" default:"false" required:"false"`
}{})

func SetupProfiler() {
	if env.EnableProfiler {
		if err := profiler.Start(profiler.Config{Service: os.Getenv("K_SERVICE")}); err != nil {
			clog.Fatalf("failed to start profiler: %v", err)
		}
	}
}
