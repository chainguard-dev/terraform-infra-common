/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// stripUserinfo must remove embedded credentials so they never reach our logs
// or downstream sinks. This is the primary safety property of the sanitizer.
func TestStripUserinfo(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"token in https", "https://x-access-token:ghs_secret@github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"basic auth", "https://user:pass@gitlab.com/g/p.git", "https://gitlab.com/g/p.git"},
		{"no userinfo", "https://github.com/org/repo.git", "https://github.com/org/repo.git"},
		{"ssh url with user", "ssh://git@github.com/org/repo.git", "ssh://github.com/org/repo.git"},
		{"not a url", "--depth=1", "--depth=1"},
		{"scp form is left alone", "git@github.com:org/repo.git", "git@github.com:org/repo.git"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, stripUserinfo(tc.in))
		})
	}
}

// sanitizeArgs is what gets logged as the args field. The contract: no token
// ever appears in the output, but argv shape (flags, paths) is preserved.
func TestSanitizeArgs(t *testing.T) {
	in := []string{"clone", "--filter=tree:0", "https://x-access-token:SECRET@github.com/o/r.git", "/tmp/dest"}
	got := sanitizeArgs(in)
	for _, a := range got {
		assert.NotContains(t, a, "SECRET", "token leaked through sanitizeArgs")
	}
	assert.Equal(t, "clone", got[0])
	assert.Equal(t, "/tmp/dest", got[3])
}

// repoFromArgs feeds the repo_host and repo_path log fields. We want clean
// values that are easy to group by in log queries.
func TestRepoFromArgs(t *testing.T) {
	host, path := repoFromArgs([]string{"clone", "--depth=1", "https://github.com/chainguard-dev/mono.git", "dest"})
	assert.Equal(t, "github.com", host)
	assert.Equal(t, "chainguard-dev/mono", path)

	host, path = repoFromArgs([]string{"for-each-ref", "refs/tags"})
	assert.Equal(t, "", host)
	assert.Equal(t, "", path)
}
