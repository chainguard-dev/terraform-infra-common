/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package profiler_test

import (
	"github.com/chainguard-dev/terraform-infra-common/pkg/profiler"
)

func ExampleSetupProfiler() {
	// SetupProfiler enables Google Cloud Profiler when ENABLE_PROFILER=true.
	profiler.SetupProfiler()
}
