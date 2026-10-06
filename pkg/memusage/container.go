/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"io/fs"
	"os"
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
}

// ReadContainer reads the container cgroup's current usage, limit, and
// lifetime peak. Each read is bounded and verified as Log's peak read is.
func ReadContainer() Container {
	return readContainer(os.DirFS("/"))
}

func readContainer(root fs.FS) Container {
	var c Container
	c.Current, _, c.CurrentErr = containerMemory(root, "memory.current", "memory.usage_in_bytes")
	c.Limit, _, c.LimitErr = containerMemory(root, "memory.max", "memory.limit_in_bytes")
	c.Peak, _, c.PeakErr = containerMemory(root, "memory.peak", "memory.max_usage_in_bytes")
	return c
}
