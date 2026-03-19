/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package profiler

import (
	"cmp"
	"os"

	"cloud.google.com/go/profiler"

	"github.com/chainguard-dev/clog"
)

func SetupProfiler() {
	enableProfiler := os.Getenv("ENABLE_PROFILER") == "true"
	clog.Debugf("Current Google Cloud Profiler setting (ENABLE_PROFILER = %t)", enableProfiler)
	if enableProfiler {
		// https://docs.cloud.google.com/run/docs/container-contract#env-vars
		service := cmp.Or(os.Getenv("K_SERVICE"), os.Getenv("CLOUD_RUN_JOB"))

		clog.Debugf("Enabling Google Cloud Profiler (ENABLE_PROFILER = %t, Service = %q)", enableProfiler, service)
		if err := profiler.Start(profiler.Config{Service: service}); err != nil {
			clog.Fatalf("failed to start profiler: %v", err)
		}
	}
}
