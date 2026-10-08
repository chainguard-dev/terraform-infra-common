/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"context"
	"io/fs"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/chainguard-dev/clog"
)

const (
	// heartbeatTick is how often usage is sampled. A sample reads a few small
	// cgroup and proc files, which takes microseconds, so sampling every
	// second is cheap; only the lines are rationed.
	heartbeatTick = time.Second
	// heartbeatEvery is how often a line is logged while usage stays at or
	// over half the limit, counting the working set. Below half, nothing is
	// logged.
	heartbeatEvery = time.Minute
	// heartbeatGrowthDivisor sets the growth, as a fraction of the limit, that
	// logs a line at once, so a fast climb toward the limit is logged about
	// every second rather than every heartbeatEvery.
	heartbeatGrowthDivisor = 32
	// heartbeatGrowthLines caps growth lines per heartbeatEvery. A climb from
	// half the limit to the limit takes at most this many steps, so a climb is
	// logged in full unless churn earlier in the same window spent the cap,
	// while usage that churns by a step, such as a scratch file written and
	// deleted per request, logs at most this many lines plus one a minute.
	heartbeatGrowthLines = heartbeatGrowthDivisor / 2
)

var heartbeatStarted atomic.Bool

// Heartbeat logs the container's memory use while it is at or over half its
// limit, until ctx is done. Cloud Run SIGKILLs an instance that reaches its
// limit, and when the memory is tmpfs files or child processes it logs no
// out-of-memory event, so these lines are the record of what filled memory.
// Each line splits usage into anonymous memory (Go heap and child processes),
// tmpfs files, and page cache. Only the first call in a process samples;
// later calls return at once, so every entry point can start it.
func Heartbeat(ctx context.Context) {
	if !heartbeatStarted.CompareAndSwap(false, true) {
		return
	}
	ticker := time.NewTicker(heartbeatTick)
	defer ticker.Stop()
	heartbeat(ctx, os.DirFS("/"), ticker.C)
}

func heartbeat(ctx context.Context, root fs.FS, tick <-chan time.Time) {
	var b beat
	reported := false
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick:
			c := readContainer(root)
			if !reported && (c.Current == nil || c.Limit == nil) {
				// Say once why there will be no lines, so their absence is not
				// read as memory staying low.
				reported = true
				clog.InfoContext(ctx, "memory heartbeat unavailable",
					"cgroup_current_error", c.CurrentErr, "cgroup_limit_error", c.LimitErr)
			}
			if b.sample(now, workingSet(c), c.Limit) {
				logHeartbeat(ctx, c)
			}
		}
	}
}

// workingSet is current usage minus the inactive page cache, which the kernel
// reclaims first; the kubelet counts the same figure toward its limit. tmpfs
// pages sit on the anonymous LRU, never in the inactive file cache, so tmpfs
// files stay in. When the inactive cache is unknown, it is current usage.
func workingSet(c Container) *uint64 {
	if c.Current == nil || c.InactiveFile == nil || *c.InactiveFile > *c.Current {
		return c.Current
	}
	return new(*c.Current - *c.InactiveFile)
}

// beat rations lines. window is when the last once-a-minute line was logged,
// growthLines counts the growth lines logged since then, and lowUsed is the
// lowest usage sampled since the last line of either kind.
type beat struct {
	window      time.Time
	growthLines int
	lowUsed     uint64
}

// sample reports whether a sample of used memory taken at now is logged.
// Nothing is logged below half the limit or when either value is unknown. At
// or over half, a line is logged once every heartbeatEvery, and at once
// whenever used has grown by limit/heartbeatGrowthDivisor over the lowest
// sample since the last line, up to heartbeatGrowthLines such lines per
// heartbeatEvery. Measuring growth from the lowest sample means a climb that
// follows a drop, such as scratch files deleted and then written again, logs
// as it climbs.
func (b *beat) sample(now time.Time, used, limit *uint64) bool {
	if used != nil {
		b.lowUsed = min(b.lowUsed, *used)
	}
	if used == nil || limit == nil || *limit == 0 || *used < *limit/2 {
		return false
	}
	if b.window.IsZero() || now.Sub(b.window) >= heartbeatEvery {
		b.window, b.growthLines, b.lowUsed = now, 0, *used
		return true
	}
	if *used >= b.lowUsed+*limit/heartbeatGrowthDivisor && b.growthLines < heartbeatGrowthLines {
		b.growthLines++
		b.lowUsed = *used
		return true
	}
	return false
}

func logHeartbeat(ctx context.Context, c Container) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	clog.InfoContext(ctx, "memory heartbeat",
		"cgroup_current_bytes", c.Current, "cgroup_current_error", c.CurrentErr,
		"cgroup_limit_bytes", c.Limit, "cgroup_limit_error", c.LimitErr,
		"cgroup_peak_bytes", c.Peak, "cgroup_peak_error", c.PeakErr,
		"cgroup_working_set_bytes", workingSet(c),
		"cgroup_anon_bytes", c.Anon, "cgroup_anon_error", c.AnonErr,
		"cgroup_shmem_bytes", c.Shmem, "cgroup_shmem_error", c.ShmemErr,
		"cgroup_file_bytes", c.File, "cgroup_file_error", c.FileErr,
		"cgroup_inactive_file_bytes", c.InactiveFile, "cgroup_inactive_file_error", c.InactiveFileErr,
		"go_heap_alloc_bytes", m.HeapAlloc,
		"go_sys_bytes", m.Sys,
		"goroutines", runtime.NumGoroutine())
}
