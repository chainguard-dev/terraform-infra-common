/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
)

func TestProcessPeakRSS(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		want   *uint64
	}{
		{name: "converts kernel KiB to bytes", status: "Name:\texport-wolfi\nVmRSS:\t20 kB\nVmHWM:\t12345 kB\n", want: new(uint64(12641280))},
		{name: "known zero", status: "VmHWM: 0 kB\n", want: new(uint64(0))},
		{name: "missing peak", status: "VmRSS: 42 kB\n"},
		{name: "missing unit", status: "VmHWM: 42\n"},
		{name: "unexpected unit", status: "VmHWM: 42 MB\n"},
		{name: "negative", status: "VmHWM: -42 kB\n"},
		{name: "overflow", status: "VmHWM: 18014398509481984 kB\n"},
		{name: "bounded read", status: "VmHWM: 42 kB\n" + strings.Repeat("x", 16*1024)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := processPeakRSS(fstest.MapFS{"proc/self/status": {Data: []byte(tc.status)}})
			if (err != nil) != (tc.want == nil) {
				t.Fatalf("error: got = %v, want error = %v", err, tc.want == nil)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("peak RSS (-want, +got): %s", diff)
			}
		})
	}
	if got, err := processPeakRSS(fstest.MapFS{}); got != nil || err == nil {
		t.Errorf("missing procfs: got = (%v, %v), want unavailable", got, err)
	}
}

func TestContainerPeakMemory(t *testing.T) {
	const v2 = "sys/fs/cgroup/memory.peak"
	const v1 = "sys/fs/cgroup/memory/memory.max_usage_in_bytes"
	const unified = "0::/\n"
	const unifiedMount = "30 20 0:28 / /sys/fs/cgroup ro - cgroup2 cgroup rw\n"
	const legacy = "5:memory:/\n"
	const legacyMount = "30 20 0:28 / /sys/fs/cgroup/memory ro - cgroup cgroup rw,memory\n"
	for _, tc := range []struct {
		name       string
		files      fstest.MapFS
		membership string
		mounts     string
		want       *uint64
		source     string
	}{
		{name: "v2", membership: unified, mounts: unifiedMount, files: fstest.MapFS{v2: {Data: []byte("635719680\n")}}, want: new(uint64(635719680)), source: v2},
		{name: "v1", membership: legacy, mounts: legacyMount, files: fstest.MapFS{v1: {Data: []byte("606076928\n")}}, want: new(uint64(606076928)), source: v1},
		{name: "hybrid v1 fallback", membership: unified + legacy, mounts: unifiedMount + legacyMount, files: fstest.MapFS{v1: {Data: []byte("34\n")}}, want: new(uint64(34)), source: v1},
		{name: "prefers v2", membership: unified + legacy, mounts: unifiedMount + legacyMount, files: fstest.MapFS{v2: {Data: []byte("12\n")}, v1: {Data: []byte("34\n")}}, want: new(uint64(12)), source: v2},
		{name: "missing accounting", membership: unified, mounts: unifiedMount, files: fstest.MapFS{}},
		{name: "invalid v2 does not hide behind v1", membership: unified + legacy, mounts: unifiedMount + legacyMount, files: fstest.MapFS{v2: {Data: []byte("max\n")}, v1: {Data: []byte("34\n")}}, source: v2},
		{name: "negative", membership: unified, mounts: unifiedMount, files: fstest.MapFS{v2: {Data: []byte("-1\n")}}, source: v2},
		{name: "overflow", membership: unified, mounts: unifiedMount, files: fstest.MapFS{v2: {Data: []byte("18446744073709551616\n")}}, source: v2},
		{name: "parent peak rejected", membership: "0::/child\n", mounts: unifiedMount, files: fstest.MapFS{v2: {Data: []byte("999999999\n")}}},
		{name: "unverified mount rejected", membership: unified, mounts: "30 20 0:28 /other /sys/fs/cgroup ro - cgroup2 cgroup rw\n", files: fstest.MapFS{v2: {Data: []byte("999999999\n")}}},
		{name: "wrong v1 controller rejected", membership: legacy, mounts: "30 20 0:28 / /sys/fs/cgroup/memory ro - cgroup cgroup rw,cpu\n", files: fstest.MapFS{v1: {Data: []byte("999999999\n")}}},
		{name: "unverified membership rejected", files: fstest.MapFS{v2: {Data: []byte("999999999\n")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.files["proc/self/cgroup"] = &fstest.MapFile{Data: []byte(tc.membership)}
			tc.files["proc/self/mountinfo"] = &fstest.MapFile{Data: []byte(tc.mounts)}
			got, source, err := containerPeakMemory(tc.files)
			if (err != nil) != (tc.want == nil) {
				t.Fatalf("error: got = %v, want error = %v", err, tc.want == nil)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("container peak (-want, +got): %s", diff)
			}
			if source != tc.source {
				t.Errorf("source: got = %q, want = %q", source, tc.source)
			}
		})
	}
}
