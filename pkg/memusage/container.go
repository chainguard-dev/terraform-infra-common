/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
)

// Container is a snapshot of the container cgroup's memory accounting, read
// from the controller mounted at the current process's root. A nil value is
// unknown, and its error says why; it is never reported as zero.
type Container struct {
	// Current is the memory charged to the container now: cgroup v2
	// memory.current or v1 memory.usage_in_bytes.
	Current    *uint64
	CurrentErr error

	// Limit is the memory limit: cgroup v2 memory.max or v1
	// memory.limit_in_bytes. A v2 limit of "max" is unknown.
	Limit    *uint64
	LimitErr error

	// Peak is the lifetime high-water mark: cgroup v2 memory.peak or v1
	// memory.max_usage_in_bytes. It only rises, so it shows how close the
	// container came to its limit even between samples.
	Peak    *uint64
	PeakErr error

	// Anon, Shmem, and File split Current by kind, from memory.stat.
	// Anon is anonymous memory of every process in the container, the Go heap
	// and child processes alike (v2 anon, v1 total_rss). Shmem is files in
	// tmpfs mounts such as Cloud Run's in-memory /tmp (v2 shmem, v1
	// total_shmem). File is the page cache, and it includes Shmem (v2 file, v1
	// total_cache). On v1, the total_ keys include child cgroups, as
	// memory.usage_in_bytes does.
	Anon     *uint64
	AnonErr  error
	Shmem    *uint64
	ShmemErr error
	File     *uint64
	FileErr  error

	// InactiveFile is the page cache the kernel reclaims first under
	// pressure (v2 inactive_file, v1 total_inactive_file).
	InactiveFile    *uint64
	InactiveFileErr error
}

// ReadContainer reads the container cgroup's current usage, limit, lifetime
// peak, and the memory.stat breakdown of current usage. Each read is bounded
// and verified as Log's peak read is.
func ReadContainer() Container {
	return readContainer(os.DirFS("/"))
}

func readContainer(root fs.FS) Container {
	var c Container
	c.Current, _, c.CurrentErr = containerMemory(root, "memory.current", "memory.usage_in_bytes")
	c.Limit, _, c.LimitErr = containerMemory(root, "memory.max", "memory.limit_in_bytes")
	c.Peak, _, c.PeakErr = containerMemory(root, "memory.peak", "memory.max_usage_in_bytes")
	stat, name, v2, err := containerFile(root, "memory.stat", "memory.stat")
	if err != nil {
		c.AnonErr, c.ShmemErr, c.FileErr, c.InactiveFileErr = err, err, err, err
		return c
	}
	// v1 keys without the total_ prefix cover only the container's own cgroup,
	// while memory.usage_in_bytes includes its children.
	anonKey, shmemKey, fileKey, inactiveKey := "total_rss", "total_shmem", "total_cache", "total_inactive_file"
	if v2 {
		anonKey, shmemKey, fileKey, inactiveKey = "anon", "shmem", "file", "inactive_file"
	}
	c.Anon, c.AnonErr = statValue(stat, name, anonKey)
	c.Shmem, c.ShmemErr = statValue(stat, name, shmemKey)
	c.File, c.FileErr = statValue(stat, name, fileKey)
	c.InactiveFile, c.InactiveFileErr = statValue(stat, name, inactiveKey)
	return c
}

// statValue returns the value of key in the contents of a memory.stat file,
// which has one "key value" pair per line.
func statValue(stat, name, key string) (*uint64, error) {
	for line := range strings.SplitSeq(stat, "\n") {
		k, v, ok := strings.Cut(line, " ")
		if !ok || k != key {
			continue
		}
		bytes, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value in %s", key, name)
		}
		return new(bytes), nil
	}
	return nil, fmt.Errorf("%s has no %s", name, key)
}
