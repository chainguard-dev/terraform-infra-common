/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics

import (
	"strings"
	"testing"
	"time"
)

func TestScrapeDiskUsage(t *testing.T) {
	t.Parallel()

	usage := scrapeDiskUsage()
	if len(usage) == 0 {
		t.Error("expected disk usage, got none")
	}

	for k := range usage {
		if strings.HasPrefix(k, "/dev") {
			t.Errorf("expected non-dev mount, got %q", k)
		}
	}
}

func TestScrapeInterval(t *testing.T) {
	// Do not use t.Parallel() — t.Setenv modifies process-global state.

	if got, want := scrapeInterval(), DiskUsageScrapeInterval; got != want {
		t.Errorf("expected positive scrape interval, want %v got %v", want, got)
	}

	t.Setenv(DiskUsageScrapeIntervalEnv, "1m30s")
	if got, want := scrapeInterval(), 90*time.Second; got != want {
		t.Errorf("expected positive scrape interval, want %v got %v", want, got)
	}
}
