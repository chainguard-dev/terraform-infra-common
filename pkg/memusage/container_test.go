/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package memusage

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
)

func TestReadContainer(t *testing.T) {
	const unified = "0::/\n"
	const unifiedMount = "30 20 0:28 / /sys/fs/cgroup ro - cgroup2 cgroup rw\n"
	files := func(extra fstest.MapFS) fstest.MapFS {
		extra["proc/self/cgroup"] = &fstest.MapFile{Data: []byte(unified)}
		extra["proc/self/mountinfo"] = &fstest.MapFile{Data: []byte(unifiedMount)}
		return extra
	}
	for _, tc := range []struct {
		name                             string
		root                             fstest.MapFS
		wantCurrent, wantLimit, wantPeak *uint64
	}{{
		name: "all three",
		root: files(fstest.MapFS{
			"sys/fs/cgroup/memory.current": {Data: []byte("1024\n")},
			"sys/fs/cgroup/memory.max":     {Data: []byte("17179869184\n")},
			"sys/fs/cgroup/memory.peak":    {Data: []byte("4096\n")},
		}),
		wantCurrent: new(uint64(1024)),
		wantLimit:   new(uint64(17179869184)),
		wantPeak:    new(uint64(4096)),
	}, {
		name: "no limit is unknown, not zero",
		root: files(fstest.MapFS{
			"sys/fs/cgroup/memory.current": {Data: []byte("1024\n")},
			"sys/fs/cgroup/memory.max":     {Data: []byte("max\n")},
		}),
		wantCurrent: new(uint64(1024)),
	}, {
		name: "unverified cgroup",
		root: fstest.MapFS{"sys/fs/cgroup/memory.current": {Data: []byte("1024\n")}},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got := readContainer(tc.root)
			for _, v := range []struct {
				field string
				want  *uint64
				got   *uint64
				err   error
			}{
				{"current", tc.wantCurrent, got.Current, got.CurrentErr},
				{"limit", tc.wantLimit, got.Limit, got.LimitErr},
				{"peak", tc.wantPeak, got.Peak, got.PeakErr},
			} {
				if diff := cmp.Diff(v.want, v.got); diff != "" {
					t.Errorf("%s (-want, +got): %s", v.field, diff)
				}
				if (v.got == nil) != (v.err != nil) {
					t.Errorf("%s: got = (%v, %v), want an error exactly when the value is unknown", v.field, v.got, v.err)
				}
			}
		})
	}
}

func TestReadContainerStat(t *testing.T) {
	const unified = "0::/\n"
	const unifiedMount = "30 20 0:28 / /sys/fs/cgroup ro - cgroup2 cgroup rw\n"
	const legacy = "5:memory:/\n"
	const legacyMount = "30 20 0:28 / /sys/fs/cgroup/memory ro - cgroup cgroup rw,memory\n"
	for _, tc := range []struct {
		name                                            string
		membership, mounts                              string
		files                                           fstest.MapFS
		wantAnon, wantShmem, wantFile, wantInactiveFile *uint64
	}{{
		name:       "cgroup v2",
		membership: unified, mounts: unifiedMount,
		files:    fstest.MapFS{"sys/fs/cgroup/memory.stat": {Data: []byte("anon 800\nfile 1400\ninactive_file 90\nkernel 7\nshmem 1300\n")}},
		wantAnon: new(uint64(800)), wantShmem: new(uint64(1300)), wantFile: new(uint64(1400)), wantInactiveFile: new(uint64(90)),
	}, {
		name:       "cgroup v1 reads the total_ keys, which include child cgroups",
		membership: legacy, mounts: legacyMount,
		files:    fstest.MapFS{"sys/fs/cgroup/memory/memory.stat": {Data: []byte("cache 7\ninactive_file 7\nrss 7\nshmem 7\ntotal_cache 1400\ntotal_inactive_file 90\ntotal_rss 800\ntotal_shmem 1300\n")}},
		wantAnon: new(uint64(800)), wantShmem: new(uint64(1300)), wantFile: new(uint64(1400)), wantInactiveFile: new(uint64(90)),
	}, {
		name:       "a missing key is unknown, not zero",
		membership: unified, mounts: unifiedMount,
		files:    fstest.MapFS{"sys/fs/cgroup/memory.stat": {Data: []byte("anon 800\nfile 1400\n")}},
		wantAnon: new(uint64(800)), wantFile: new(uint64(1400)),
	}, {
		name:       "a malformed value is unknown",
		membership: unified, mounts: unifiedMount,
		files:     fstest.MapFS{"sys/fs/cgroup/memory.stat": {Data: []byte("anon lots\nfile 1400\nshmem 1300\n")}},
		wantShmem: new(uint64(1300)), wantFile: new(uint64(1400)),
	}, {
		name:       "no memory.stat",
		membership: unified, mounts: unifiedMount,
		files: fstest.MapFS{},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			tc.files["proc/self/cgroup"] = &fstest.MapFile{Data: []byte(tc.membership)}
			tc.files["proc/self/mountinfo"] = &fstest.MapFile{Data: []byte(tc.mounts)}
			got := readContainer(tc.files)
			for _, v := range []struct {
				field string
				want  *uint64
				got   *uint64
				err   error
			}{
				{"anon", tc.wantAnon, got.Anon, got.AnonErr},
				{"shmem", tc.wantShmem, got.Shmem, got.ShmemErr},
				{"file", tc.wantFile, got.File, got.FileErr},
				{"inactive_file", tc.wantInactiveFile, got.InactiveFile, got.InactiveFileErr},
			} {
				if diff := cmp.Diff(v.want, v.got); diff != "" {
					t.Errorf("%s (-want, +got): %s", v.field, diff)
				}
				if (v.got == nil) != (v.err != nil) {
					t.Errorf("%s: got = (%v, %v), want an error exactly when the value is unknown", v.field, v.got, v.err)
				}
			}
		})
	}
}
