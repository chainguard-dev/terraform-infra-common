package httpmetrics

import (
	"strings"
	"testing"
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
