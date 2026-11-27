//go:build unix

/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// stubGitScript stands in for git. It installs its SIGTERM trap, which
// leaves a git.term marker that a SIGKILL would not, then starts a helper
// that inherits stderr, as git-remote-https does, and waits. The helper runs
// $HELPER, which must install its own signal disposition before recording
// its pid and signalling readiness with markReady, so a test that cancels
// after readiness knows both traps are in place.
const stubGitScript = `#!/bin/sh
trap ': > "$STUB_DIR/git.term"; exit 143' TERM
sh -c "$HELPER" &
wait
`

// markReady is the shell a helper runs once its signal disposition is in
// place.
const markReady = `echo $$ > "$STUB_DIR/helper.pid"; touch "$STUB_DIR/ready"`

// Tests that must wait out WaitDelay, because something holds stderr past
// SIGTERM, shorten it to testGracePeriod. Tests where SIGTERM ends everything
// keep a grace period of hangGuard, so a slow runner cannot let the final
// SIGKILL beat a SIGTERM trap. The traps write their markers with a shell
// builtin, so they do not fork. The tests assert what was signalled and what
// is left running, never how long anything took. hangGuard only turns a
// regression that would block forever into a failure with a message.
const (
	testGracePeriod = 200 * time.Millisecond
	hangGuard       = 30 * time.Second
)

// requireProc skips the test when /proc is missing: alive cannot tell a
// zombie from a running process without it, and would report every process
// as exited.
func requireProc(t *testing.T) {
	t.Helper()
	if _, err := os.Stat("/proc/self/stat"); err != nil {
		t.Skip("needs /proc to tell a zombie from a running process")
	}
}

// alive reports whether pid is a running process. A killed helper is
// reparented when the stub exits and may stay a zombie until reaped, so a
// zombie counts as exited.
func alive(pid int) bool {
	if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
		return false
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return false
	}
	// The state is the first field after the parenthesized command name.
	_, rest, _ := strings.Cut(string(stat), ") ")
	return !strings.HasPrefix(rest, "Z")
}

// waitFor polls cond until it holds or timeout passes.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

// exists reports whether path exists.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// startStub puts the stub git on PATH, calls run in the background, waits for the stub's helper to start and returns the helper's
// pid and the channel Run's error arrives on. The helper is killed at
// cleanup in case the code under test left it running.
func startStub(t *testing.T, stubDir, helper string, run func() error) (int, <-chan error) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(stubDir, "git"), []byte(stubGitScript), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("STUB_DIR", stubDir)
	t.Setenv("HELPER", helper)

	// run builds its command after PATH is set, because exec resolves "git"
	// when the command is built.
	done := make(chan error, 1)
	go func() { done <- run() }()

	if !waitFor(hangGuard, func() bool { return exists(filepath.Join(stubDir, "ready")) }) {
		t.Fatal("stub git never signalled readiness")
	}
	raw, err := os.ReadFile(filepath.Join(stubDir, "helper.pid"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(pid, syscall.SIGKILL)
		// A helper may start its own child, which would otherwise outlive a
		// failing test and hold go test's output open.
		if raw, err := os.ReadFile(filepath.Join(stubDir, "sleep.pid")); err == nil {
			if child, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				_ = syscall.Kill(child, syscall.SIGKILL)
			}
		}
	})
	return pid, done
}

// awaitRun waits for Run's error and wants a non-nil one, since the command
// was cancelled.
func awaitRun(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err == nil {
			t.Error("Run after cancel: got = nil error, want an error")
		}
	case <-time.After(hangGuard):
		t.Fatal("Run did not return after cancel")
	}
}

// setGracePeriod sets stopGracePeriod to d for the test.
func setGracePeriod(t *testing.T, d time.Duration) {
	t.Helper()
	prev := stopGracePeriod
	stopGracePeriod = d
	t.Cleanup(func() { stopGracePeriod = prev })
}

// TestRunStopsProcessGroupOnCancel cancels a command whose helper holds
// stderr open and checks that Run returns and the helper is gone. Not
// parallel: it replaces PATH and stopGracePeriod.
func TestRunStopsProcessGroupOnCancel(t *testing.T) {
	requireProc(t)

	tests := []struct {
		name   string
		helper string
		// grace is the WaitDelay for the row: long where SIGTERM ends the
		// helper, short where the helper must be waited out.
		grace time.Duration
		// wantHelperTerm is set when the helper exits on SIGTERM, so it must
		// have received the group's SIGTERM rather than the final SIGKILL.
		wantHelperTerm bool
	}{{
		name:           "helper exits on SIGTERM",
		helper:         `trap ': > "$STUB_DIR/helper.term"; exit 0' TERM; sleep 300 & echo $! > "$STUB_DIR/sleep.pid"; ` + markReady + `; wait`,
		grace:          hangGuard,
		wantHelperTerm: true,
	}, {
		name:           "helper ignores SIGTERM",
		helper:         "trap '' TERM; " + markReady + "; exec sleep 300",
		grace:          testGracePeriod,
		wantHelperTerm: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setGracePeriod(t, tt.grace)
			stubDir := t.TempDir()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			pid, done := startStub(t, stubDir, tt.helper, func() error {
				return Run(ctx, "fetch", CommandContext(ctx, "fetch"))
			})

			cancel()
			awaitRun(t, done)
			if got := exists(filepath.Join(stubDir, "helper.term")); got != tt.wantHelperTerm {
				t.Errorf("helper received SIGTERM: got = %t, want = %t", got, tt.wantHelperTerm)
			}
			if !waitFor(hangGuard, func() bool { return !alive(pid) }) {
				t.Errorf("helper %d still running after Run returned", pid)
			}
		})
	}
}

// TestRunWithoutProcessGroupReturns checks that a caller that turns the
// process group off still gets a Run that returns after cancel: Cancel falls
// back to sending git SIGTERM, and WaitDelay bounds the wait on its helper.
func TestRunWithoutProcessGroupReturns(t *testing.T) {
	requireProc(t)
	setGracePeriod(t, testGracePeriod)

	stubDir := t.TempDir()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, done := startStub(t, stubDir, markReady+"; exec sleep 300", func() error {
		cmd := CommandContext(ctx, "fetch")
		cmd.SysProcAttr = nil
		return Run(ctx, "fetch", cmd)
	})

	cancel()
	awaitRun(t, done)
	// Without the fallback, git is only killed by WaitDelay's SIGKILL, which
	// leaves no marker.
	if !exists(filepath.Join(stubDir, "git.term")) {
		t.Error("git received SIGTERM: got = false, want = true")
	}
}

// pgidStubScript prints the stub's pid and its process group id, read from
// /proc since ps may be missing.
const pgidStubScript = `#!/bin/sh
echo $$ $(cut -d' ' -f5 /proc/$$/stat)
`

// TestCommandContextProcessGroup checks which process group git runs in: its
// own by default, and the caller's once the caller clears SysProcAttr, the
// opt-out the CommandContext doc gives callers that need to prompt.
func TestCommandContextProcessGroup(t *testing.T) {
	requireProc(t)
	stubDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(stubDir, "git"), []byte(pgidStubScript), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tests := []struct {
		name         string
		clearAttr    bool
		wantOwnGroup bool
	}{{
		name:         "default runs git in its own group",
		clearAttr:    false,
		wantOwnGroup: true,
	}, {
		name:         "cleared SysProcAttr keeps git in the caller's group",
		clearAttr:    true,
		wantOwnGroup: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := CommandContext(t.Context(), "version")
			if tt.clearAttr {
				cmd.SysProcAttr = nil
			}
			out, err := Output(t.Context(), "version", cmd)
			if err != nil {
				t.Fatal(err)
			}
			fields := strings.Fields(string(out))
			if len(fields) != 2 {
				t.Fatalf("stub output: got = %q, want \"<pid> <pgid>\"", out)
			}
			pid, pgid := fields[0], fields[1]
			want := strconv.Itoa(syscall.Getpgrp())
			if tt.wantOwnGroup {
				want = pid
			}
			if pgid != want {
				t.Errorf("git's process group: got = %s, want = %s", pgid, want)
			}
		})
	}
}

// TestRunBoundsWaitAfterCleanExit pins the documented clean-exit behavior of
// the default WaitDelay: when git exits 0 but a process it started keeps
// stderr open, Run returns exec.ErrWaitDelay instead of waiting as long as
// that process lives.
func TestRunBoundsWaitAfterCleanExit(t *testing.T) {
	setGracePeriod(t, testGracePeriod)
	stubDir := t.TempDir()
	// The stub exits 0 at once, leaving its helper holding stderr.
	script := "#!/bin/sh\nsh -c \"$HELPER\" &\nwhile [ ! -e \"$STUB_DIR/ready\" ]; do sleep 0.01; done\nexit 0\n"
	if err := os.WriteFile(filepath.Join(stubDir, "git"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("STUB_DIR", stubDir)
	t.Setenv("HELPER", markReady+"; exec sleep 300")
	t.Cleanup(func() {
		if raw, err := os.ReadFile(filepath.Join(stubDir, "helper.pid")); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
	})

	done := make(chan error, 1)
	go func() { done <- Run(t.Context(), "fetch", CommandContext(t.Context(), "fetch")) }()
	select {
	case err := <-done:
		if !errors.Is(err, exec.ErrWaitDelay) {
			t.Errorf("Run after a clean exit with stderr held open: got = %v, want = %v", err, exec.ErrWaitDelay)
		}
	case <-time.After(hangGuard):
		t.Fatal("Run did not return after git exited")
	}
}
