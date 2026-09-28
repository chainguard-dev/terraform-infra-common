/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"os"
	"testing"
)

// Check the parser against the real kernel as well as the malformed/unsupported
// fixture cases. Container accounting may be unavailable on a non-container host.
func TestKernelMemoryAccounting(t *testing.T) {
	root := os.DirFS("/")
	rss, err := processPeakRSS(root)
	if err != nil || rss == nil || *rss == 0 {
		t.Fatalf("kernel peak RSS: got = (%v, %v), want a positive byte count", rss, err)
	}
	t.Logf("process_peak_rss_bytes=%d", *rss)
	peak, source, err := containerPeakMemory(root)
	if err != nil {
		if peak != nil {
			t.Errorf("unavailable container peak: got = %d, want nil", *peak)
		}
		t.Logf("container peak unavailable: %v", err)
		return
	}
	if peak == nil || *peak == 0 {
		t.Fatalf("kernel container peak: got = %v, want a positive byte count", peak)
	}
	t.Logf("container_peak_memory_bytes=%d source=%s", *peak, source)
}
