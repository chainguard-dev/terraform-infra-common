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

// A "-c key=value" config value can carry a credential (an authenticated clone
// passes the GitHub token as http.<url>.extraHeader). The value must never
// reach the logs, but the key is safe and useful for debugging, so it stays.
func TestSanitizeArgs_redactsConfigValue(t *testing.T) {
	in := []string{
		"-c", "http.https://github.com/.extraHeader=Authorization: Basic eC1hY2Nlc3MtdG9rZW46U0VDUkVU",
		"clone", "--bare", "https://github.com/o/r.git", "/tmp/dest",
	}
	got := sanitizeArgs(in)
	for _, a := range got {
		assert.NotContains(t, a, "eC1hY2Nlc3MtdG9rZW46U0VDUkVU", "config value leaked through sanitizeArgs")
		assert.NotContains(t, a, "Authorization", "config value leaked through sanitizeArgs")
	}
	assert.Equal(t, "http.https://github.com/.extraHeader=<redacted>", got[1])
	assert.Equal(t, "clone", got[2])
	assert.Equal(t, "https://github.com/o/r.git", got[4])
}

// A bare "-c key" with no value is git's shorthand for "key=true"; there is
// nothing secret to mask, so the arg passes through untouched.
func TestSanitizeArgs_bareConfigKeyUnchanged(t *testing.T) {
	in := []string{"-c", "protocol.version", "fetch"}
	got := sanitizeArgs(in)
	assert.Equal(t, []string{"-c", "protocol.version", "fetch"}, got)
}

// Only the arg following "-c" is treated as a config pair. The uppercase
// "-C <dir>" flag (a working directory) must keep its value intact so logs
// still show where the command ran.
func TestSanitizeArgs_uppercaseCNotTreatedAsConfig(t *testing.T) {
	in := []string{"-C", "/work/repo=clone", "status"}
	got := sanitizeArgs(in)
	assert.Equal(t, "/work/repo=clone", got[1])
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
