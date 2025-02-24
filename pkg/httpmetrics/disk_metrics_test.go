package httpmetrics

import (
	"os"
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
	t.Parallel()

	if got, want := scrapeInterval(), DiskUsageScrapeInterval; got != want {
		t.Errorf("expected positive scrape interval, want %v got %v", want, got)
	}

	// set the env
	os.Setenv(DiskUsageScrapeIntervalEnv, "1m30s")
	defer os.Unsetenv(DiskUsageScrapeIntervalEnv)
	if got, want := scrapeInterval(), 90*time.Second; got != want {
		t.Errorf("expected positive scrape interval, want %v got %v", want, got)
	}
}
