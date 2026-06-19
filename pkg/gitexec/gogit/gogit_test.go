/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gogit

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainguard-dev/clog"
	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs returns a context whose clog writes JSON to buf so tests can
// assert on the structured fields gitexec emits.
func captureLogs(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return clog.WithLogger(t.Context(), clog.New(h)), &buf
}

// opLine returns the last log line carrying the given op, failing the test if
// none is present. Operations append to the same buffer, so a per-op line lets
// a test assert the outcome of one operation without matching another's.
func opLine(t *testing.T, buf *bytes.Buffer, op string) string {
	t.Helper()
	var found string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if strings.Contains(line, `"op":"`+op+`"`) {
			found = line
		}
	}
	require.NotEmptyf(t, found, "no %q observation in logs:\n%s", op, buf.String())
	return found
}

func testSignature() *object.Signature {
	return &object.Signature{Name: "Test", Email: "test@example.com", When: time.Unix(1700000000, 0).UTC()}
}

func commitFile(t *testing.T, repo *git.Repository, name, contents string) {
	t.Helper()
	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(wt.Filesystem.Root(), name), []byte(contents), 0o600))
	_, err = wt.Add(name)
	require.NoError(t, err)
	_, err = wt.Commit("commit "+name, &git.CommitOptions{Author: testSignature(), Committer: testSignature()})
	require.NoError(t, err)
}

// seedRemote creates a bare repository pre-populated with one commit on its
// default branch, suitable as the origin for clone/fetch/push tests. Returns
// the bare repo path.
func seedRemote(t *testing.T) string {
	t.Helper()
	bareDir := t.TempDir()
	_, err := git.PlainInit(bareDir, true)
	require.NoError(t, err)

	workDir := t.TempDir()
	work, err := git.PlainInit(workDir, false)
	require.NoError(t, err)
	commitFile(t, work, "README.md", "seed\n")

	_, err = work.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{bareDir}})
	require.NoError(t, err)
	head, err := work.Head()
	require.NoError(t, err)
	spec := config.RefSpec(head.Name().String() + ":" + head.Name().String())
	require.NoError(t, work.Push(&git.PushOptions{RemoteName: "origin", RefSpecs: []config.RefSpec{spec}}))
	return bareDir
}

// A clone must record one "clone" observation and hand back a *Repository whose
// embedded local methods (here Head) work unchanged — the property that lets
// callers swap only the constructor.
func TestPlainCloneContext_RecordsCloneAndPromotesLocalMethods(t *testing.T) {
	ctx, buf := captureLogs(t)
	dst := filepath.Join(t.TempDir(), "clone")

	repo, err := PlainCloneContext(ctx, dst, false, &git.CloneOptions{URL: seedRemote(t)})
	require.NoError(t, err)
	require.NotNil(t, repo)

	// Promoted from the embedded *git.Repository, no override needed.
	_, err = repo.Head()
	require.NoError(t, err)

	line := opLine(t, buf, "clone")
	assert.Contains(t, line, `"outcome":"success"`)
}

// CloneContext is the constructor for cloning into a caller-provided storer —
// the in-memory case PlainCloneContext cannot serve. It must record one "clone"
// observation and return a *Repository whose embedded local methods work, so a
// custom-storer caller swaps only the constructor.
func TestCloneContext_RecordsCloneIntoCustomStorer(t *testing.T) {
	ctx, buf := captureLogs(t)

	repo, err := CloneContext(ctx, memory.NewStorage(), memfs.New(), &git.CloneOptions{URL: seedRemote(t)})
	require.NoError(t, err)
	require.NotNil(t, repo)

	// Promoted from the embedded *git.Repository, no override needed.
	_, err = repo.Head()
	require.NoError(t, err)

	line := opLine(t, buf, "clone")
	assert.Contains(t, line, `"outcome":"success"`)
}

// An up-to-date fetch must return NoErrAlreadyUpToDate to the caller (so their
// control flow is unchanged from raw go-git) yet record outcome=success: the
// round-trip happened and is real traffic, not a failure.
func TestRepositoryFetchContext_UpToDateReturnsSentinelButRecordsSuccess(t *testing.T) {
	ctx, buf := captureLogs(t)
	dst := filepath.Join(t.TempDir(), "clone")
	repo, err := PlainCloneContext(ctx, dst, false, &git.CloneOptions{URL: seedRemote(t)})
	require.NoError(t, err)

	err = repo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin"})
	require.ErrorIs(t, err, git.NoErrAlreadyUpToDate)

	line := opLine(t, buf, "fetch")
	assert.Contains(t, line, `"outcome":"success"`)
}

// A push of a new commit must record one "push" observation with success.
func TestRepositoryPushContext_RecordsPush(t *testing.T) {
	ctx, buf := captureLogs(t)
	dst := filepath.Join(t.TempDir(), "clone")
	repo, err := PlainCloneContext(ctx, dst, false, &git.CloneOptions{URL: seedRemote(t)})
	require.NoError(t, err)

	commitFile(t, repo.Repository, "added.txt", "more\n")
	head, err := repo.Head()
	require.NoError(t, err)
	spec := config.RefSpec(head.Name().String() + ":" + head.Name().String())

	err = repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin", RefSpecs: []config.RefSpec{spec}})
	require.NoError(t, err)

	line := opLine(t, buf, "push")
	assert.Contains(t, line, `"outcome":"success"`)
}

// A remote built with NewRemote must observe its fetch and derive repo_host and
// repo_path from the remote's configured URL, with no WithRepoURL at the call
// site. example.invalid fails fast (NXDOMAIN), exercising the failure path.
func TestRemoteFetchContext_DerivesRepoFromConfig(t *testing.T) {
	ctx, buf := captureLogs(t)
	dst := filepath.Join(t.TempDir(), "clone")
	repo, err := PlainCloneContext(ctx, dst, false, &git.CloneOptions{URL: seedRemote(t)})
	require.NoError(t, err)

	rem := NewRemote(repo.Storer, &config.RemoteConfig{
		Name: "observed",
		URLs: []string{"https://example.invalid/chainguard-dev/mono.git"},
	})

	fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = rem.FetchContext(fetchCtx, &git.FetchOptions{RemoteName: "observed"})
	require.Error(t, err)

	line := opLine(t, buf, "fetch")
	assert.Contains(t, line, `"outcome":"failure"`)
	assert.Contains(t, line, `"repo_host":"example.invalid"`)
	assert.Contains(t, line, `"repo_path":"chainguard-dev/mono"`)
}

func TestWrap_NilReturnsNil(t *testing.T) {
	assert.Nil(t, Wrap(nil))
}

func TestFirstURL(t *testing.T) {
	assert.Empty(t, firstURL(nil))
	assert.Empty(t, firstURL(&config.RemoteConfig{}))
	assert.Equal(t, "https://example.com/o/r.git",
		firstURL(&config.RemoteConfig{URLs: []string{"https://example.com/o/r.git", "second"}}))
}
