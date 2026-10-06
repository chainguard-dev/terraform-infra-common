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
