/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/chainguard-dev/clog"
)

// Log records Linux kernel memory high-water marks with the context logger.
// Missing or unsupported accounting is logged as null with an error, not zero.
// The process RSS excludes children and unmapped tmpfs; the cgroup peak includes
// all memory charged to the container, including child processes and scratch files.
// These are lifetime peaks, not per-call measurements. Log is safe for concurrent use.
func Log(ctx context.Context) {
	root := os.DirFS("/")
	rss, rssErr := processPeakRSS(root)
	peak, source, peakErr := containerPeakMemory(root)
	clog.InfoContext(ctx, "memory usage",
		"process_peak_rss_bytes", rss, "process_peak_rss_error", rssErr,
		"container_peak_memory_bytes", peak, "container_peak_memory_source", source,
		"container_peak_memory_error", peakErr)
}

func processPeakRSS(root fs.FS) (*uint64, error) {
	status, err := readMemoryFile(root, "proc/self/status")
	if err != nil {
		return nil, err
	}
	for line := range strings.SplitSeq(status, "\n") {
		if value, ok := strings.CutPrefix(line, "VmHWM:"); ok {
			fields := strings.Fields(value)
			if len(fields) != 2 || fields[1] != "kB" {
				return nil, fmt.Errorf("unexpected VmHWM format")
			}
			kb, err := strconv.ParseUint(fields[0], 10, 64)
			if err != nil || kb > math.MaxUint64/1024 {
				return nil, fmt.Errorf("invalid VmHWM value")
			}
			return new(kb * 1024), nil
		}
	}
	return nil, fmt.Errorf("VmHWM unavailable")
}

func containerPeakMemory(root fs.FS) (*uint64, string, error) {
	return containerMemory(root, "memory.peak", "memory.max_usage_in_bytes")
}

// containerMemory reads one memory accounting file from the cgroup controller
// mounted at the current process's root: v2File under cgroup v2, else v1File
// under cgroup v1. A v2 value of "max" means no limit and is unknown, not zero.
func containerMemory(root fs.FS, v2File, v1File string) (*uint64, string, error) {
	membership, err := readMemoryFile(root, "proc/self/cgroup")
	if err != nil {
		return nil, "", err
	}
	mounts, err := readMemoryFile(root, "proc/self/mountinfo")
	if err != nil {
		return nil, "", err
	}
	// Only use a controller mounted at the current process's namespace root.
	// A parent cgroup's hierarchical peak would include unrelated processes.
	// Non-root or unusual layouts remain unknown rather than guessing a path.
	for _, candidate := range []struct {
		mount, kind, file string
	}{
		{mount: "/sys/fs/cgroup", kind: "cgroup2", file: v2File},
		{mount: "/sys/fs/cgroup/memory", kind: "cgroup", file: v1File},
	} {
		if !rootMemoryController(membership, mounts, candidate.kind, candidate.mount) {
			continue
		}
		name := strings.TrimPrefix(candidate.mount, "/") + "/" + candidate.file
		value, err := readMemoryFile(root, name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, name, err
		}
		if strings.TrimSpace(value) == "max" {
			return nil, name, fmt.Errorf("%s has no limit", name)
		}
		bytes, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return nil, name, fmt.Errorf("invalid cgroup memory value in %s", name)
		}
		return new(bytes), name, nil
	}
	return nil, "", fmt.Errorf("cgroup memory accounting %s unavailable", v2File)
}

func rootMemoryController(membership, mounts, kind, mount string) bool {
	member := false
	for line := range strings.SplitSeq(membership, "\n") {
		id, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		controllers, path, ok := strings.Cut(rest, ":")
		if !ok || path != "/" {
			continue
		}
		if kind == "cgroup2" && id == "0" && controllers == "" {
			member = true
		}
		if kind == "cgroup" {
			for controller := range strings.SplitSeq(controllers, ",") {
				if controller == "memory" {
					member = true
				}
			}
		}
	}
	if !member {
		return false
	}
	for line := range strings.SplitSeq(mounts, "\n") {
		before, after, ok := strings.Cut(line, " - ")
		if !ok {
			continue
		}
		fields, options := strings.Fields(before), strings.Fields(after)
		if len(fields) < 6 || fields[3] != "/" || fields[4] != mount || len(options) < 3 || options[0] != kind {
			continue
		}
		if kind == "cgroup2" {
			return true
		}
		for option := range strings.SplitSeq(options[2], ",") {
			if option == "memory" {
				return true
			}
		}
	}
	return false
}

func readMemoryFile(root fs.FS, name string) (string, error) {
	f, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	const limit = 16 * 1024
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return "", err
	}
	if len(b) > limit {
		return "", fmt.Errorf("memory accounting file exceeds %d bytes", limit)
	}
	return string(b), nil
}
