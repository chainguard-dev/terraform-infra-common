/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"testing/fstest"
	"time"

	"github.com/chainguard-dev/clog"
)

func TestBeatSample(t *testing.T) {
	start := time.Date(2026, 10, 7, 23, 45, 0, 0, time.UTC)
	limit := uint64(16 << 30)
	growth := limit / heartbeatGrowthDivisor
	half := limit / 2
	var b beat
	for _, step := range []struct {
		name  string
		after time.Duration
		used  *uint64
		limit *uint64
		want  bool
	}{{
		name: "below half is quiet", after: 0, used: new(limit / 4), limit: &limit,
	}, {
		name: "crossing half logs at once", after: time.Second, used: new(half), limit: &limit, want: true,
	}, {
		name: "growth under the step is quiet", after: 2 * time.Second, used: new(half + growth - 1), limit: &limit,
	}, {
		name: "growth of the step logs at once", after: 3 * time.Second, used: new(half + growth), limit: &limit, want: true,
	}, {
		name: "flat usage inside heartbeatEvery is quiet", after: heartbeatEvery, used: new(half + growth), limit: &limit,
	}, {
		name: "flat usage logs every heartbeatEvery", after: time.Second + heartbeatEvery, used: new(half + growth), limit: &limit, want: true,
	}, {
		name: "dropping below half is quiet", after: 4*time.Second + heartbeatEvery, used: new(limit / 4), limit: &limit,
	}, {
		name: "a climb after a drop logs at once", after: 5*time.Second + heartbeatEvery, used: new(half), limit: &limit, want: true,
	}, {
		name: "a dip under the step is quiet", after: 6*time.Second + heartbeatEvery, used: new(half - growth/2), limit: &limit,
	}, {
		name: "recovering from the dip is quiet", after: 7*time.Second + heartbeatEvery, used: new(half), limit: &limit,
	}, {
		name: "growth of the step over the dip logs at once", after: 8*time.Second + heartbeatEvery, used: new(half + growth/2), limit: &limit, want: true,
	}, {
		name: "unknown limit is quiet", after: time.Hour, used: new(half + growth),
	}, {
		name: "unknown usage is quiet", after: time.Hour, limit: &limit,
	}, {
		name: "zero limit is quiet", after: time.Hour, used: new(half), limit: new(uint64(0)),
	}} {
		if got := b.sample(start.Add(step.after), step.used, step.limit); got != step.want {
			t.Errorf("%s: got = %t, want = %t", step.name, got, step.want)
		}
	}
}

func TestBeatSampleBounds(t *testing.T) {
	start := time.Date(2026, 10, 7, 23, 45, 0, 0, time.UTC)
	limit := uint64(16 << 30)
	growth := limit / heartbeatGrowthDivisor
	half := limit / 2
	for _, tc := range []struct {
		name string
		used func(second int) uint64
		want int
	}{{
		// A scratch file written and deleted every other second.
		name: "churn of one step logs the minute line and the growth cap",
		used: func(second int) uint64 { return half + uint64(second%2)*growth },
		want: 1 + heartbeatGrowthLines,
	}, {
		// The fastest climb a second-by-second sampler sees: one step a second
		// from half to the limit.
		name: "a climb from half to the limit logs every step",
		used: func(second int) uint64 { return min(half+uint64(second)*growth, limit) },
		want: 1 + heartbeatGrowthLines,
	}, {
		name: "flat usage logs only the minute line",
		used: func(int) uint64 { return half },
		want: 1,
	}} {
		t.Run(tc.name, func(t *testing.T) {
			var b beat
			got := 0
			for second := range int(heartbeatEvery / time.Second) {
				if b.sample(start.Add(time.Duration(second)*time.Second), new(tc.used(second)), &limit) {
					got++
				}
			}
			if got != tc.want {
				t.Errorf("lines in one heartbeatEvery: got = %d, want = %d", got, tc.want)
			}
		})
	}
}

func TestBeatSampleCapResetsEachWindow(t *testing.T) {
	start := time.Date(2026, 10, 7, 23, 45, 0, 0, time.UTC)
	limit := uint64(16 << 30)
	growth := limit / heartbeatGrowthDivisor
	half := limit / 2
	window := int(heartbeatEvery / time.Second)
	var b beat
	// Churn by one step for a whole window, which spends the growth cap.
	for second := range window {
		b.sample(start.Add(time.Duration(second)*time.Second), new(half+uint64(second%2)*growth), &limit)
	}
	// In the next window, a climb of one step a second must log every step.
	for i := range heartbeatGrowthLines + 1 {
		now := start.Add(time.Duration(window+i) * time.Second)
		if !b.sample(now, new(half+uint64(i)*growth), &limit) {
			t.Errorf("climb step %d in the second window: got = quiet, want = logged", i)
		}
	}
}

func TestWorkingSet(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    Container
		want *uint64
	}{{
		name: "subtracts the inactive page cache",
		c:    Container{Current: new(uint64(1000)), InactiveFile: new(uint64(300))},
		want: new(uint64(700)),
	}, {
		name: "unknown inactive cache leaves current usage",
		c:    Container{Current: new(uint64(1000))},
		want: new(uint64(1000)),
	}, {
		name: "inactive cache over current usage leaves current usage",
		c:    Container{Current: new(uint64(1000)), InactiveFile: new(uint64(2000))},
		want: new(uint64(1000)),
	}, {
		name: "unknown current usage is unknown",
		c:    Container{InactiveFile: new(uint64(300))},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := workingSet(tc.c)
			if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
				t.Errorf("workingSet: got = %v, want = %v", got, tc.want)
			}
		})
	}
}

// runHeartbeat runs heartbeat over root for the given number of ticks and
// returns the lines it logged.
func runHeartbeat(t *testing.T, root fstest.MapFS, ticks int) []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(clog.WithLogger(t.Context(), clog.New(slog.NewJSONHandler(&buf, nil))))
	tick := make(chan time.Time)
	done := make(chan struct{})
	go func() {
		defer close(done)
		heartbeat(ctx, root, tick)
	}()
	now := time.Now()
	for i := range ticks {
		// Each send returns once the loop has the tick, and the loop finishes
		// that sample before it receives again or checks ctx.
		tick <- now.Add(time.Duration(i) * heartbeatEvery)
	}
	cancel()
	<-done

	var lines []map[string]any
	for s := bufio.NewScanner(&buf); s.Scan(); {
		var line map[string]any
		if err := json.Unmarshal(s.Bytes(), &line); err != nil {
			t.Fatalf("log line %q: %v", s.Text(), err)
		}
		lines = append(lines, line)
	}
	return lines
}

func cgroupV2(stat string) fstest.MapFS {
	return fstest.MapFS{
		"proc/self/cgroup":             {Data: []byte("0::/\n")},
		"proc/self/mountinfo":          {Data: []byte("30 20 0:28 / /sys/fs/cgroup ro - cgroup2 cgroup rw\n")},
		"sys/fs/cgroup/memory.current": {Data: []byte("15000000000\n")},
		"sys/fs/cgroup/memory.max":     {Data: []byte("17179869184\n")},
		"sys/fs/cgroup/memory.peak":    {Data: []byte("15500000000\n")},
		"sys/fs/cgroup/memory.stat":    {Data: []byte(stat)},
	}
}

func TestHeartbeatLogsBreakdown(t *testing.T) {
	lines := runHeartbeat(t, cgroupV2("anon 800000000\nfile 14100000000\ninactive_file 100000000\nshmem 14000000000\n"), 1)
	if len(lines) != 1 {
		t.Fatalf("lines: got = %d, want = 1", len(lines))
	}
	for key, want := range map[string]any{
		"msg":                        "memory heartbeat",
		"cgroup_current_bytes":       float64(15000000000),
		"cgroup_limit_bytes":         float64(17179869184),
		"cgroup_peak_bytes":          float64(15500000000),
		"cgroup_working_set_bytes":   float64(14900000000),
		"cgroup_anon_bytes":          float64(800000000),
		"cgroup_shmem_bytes":         float64(14000000000),
		"cgroup_file_bytes":          float64(14100000000),
		"cgroup_inactive_file_bytes": float64(100000000),
	} {
		if got := lines[0][key]; got != want {
			t.Errorf("%s: got = %v, want = %v", key, got, want)
		}
	}
}

func TestHeartbeatQuietWhenCacheFillsMemory(t *testing.T) {
	// 15 GB charged, but 10 GB of it is inactive page cache, so the working
	// set is under half the 16 GiB limit.
	if lines := runHeartbeat(t, cgroupV2("anon 5000000000\nfile 10000000000\ninactive_file 10000000000\nshmem 0\n"), 2); len(lines) != 0 {
		t.Errorf("lines: got = %v, want = none", lines)
	}
}

func TestHeartbeatSaysOnceWhenUnavailable(t *testing.T) {
	lines := runHeartbeat(t, fstest.MapFS{}, 3)
	if len(lines) != 1 {
		t.Fatalf("lines: got = %v, want = one", lines)
	}
	if got, want := lines[0]["msg"], "memory heartbeat unavailable"; got != want {
		t.Errorf("msg: got = %v, want = %v", got, want)
	}
}

func TestHeartbeatSecondCallReturns(t *testing.T) {
	heartbeatStarted.Store(true)
	t.Cleanup(func() { heartbeatStarted.Store(false) })
	done := make(chan struct{})
	go func() {
		defer close(done)
		Heartbeat(t.Context())
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Heartbeat did not return when a sampler was already running")
	}
}
