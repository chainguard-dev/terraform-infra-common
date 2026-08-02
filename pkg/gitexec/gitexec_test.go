/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"strings"
	"testing"

	"github.com/chainguard-dev/clog"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs returns a context whose clog writes JSON to buf, so tests can
// assert on the structured fields we emit.
func captureLogs(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := clog.New(h)
	return clog.WithLogger(t.Context(), logger), &buf
}

// A successful exec should land exactly one INFO log carrying op, outcome,
// and a positive duration. Without those fields we cannot answer the user's
// question ("how often do we push, pull, clone, fetch?").
func TestRun_Success(t *testing.T) {
	ctx, buf := captureLogs(t)
	before := testutil.ToFloat64(operationsTotal.WithLabelValues("--version", outcomeSuccess))

	cmd := CommandContext(ctx, "--version")
	err := Run(ctx, "--version", cmd)
	require.NoError(t, err)

	after := testutil.ToFloat64(operationsTotal.WithLabelValues("--version", outcomeSuccess))
	assert.Equal(t, before+1, after, "success counter must advance by one")

	out := buf.String()
	assert.Contains(t, out, `"msg":"git_operation"`)
	assert.Contains(t, out, `"op":"--version"`)
	assert.Contains(t, out, `"outcome":"success"`)
	assert.Contains(t, out, `"exit_code":0`)
	// duration_ms is integer JSON; assert the key is present.
	assert.Contains(t, out, `"duration_ms":`)
}

// A failing exec must surface as an ERROR log with outcome=failure and a
// non-zero exit code. The metric must increment on the failure label so we
// can graph error rates separately from volume.
func TestRun_Failure(t *testing.T) {
	ctx, buf := captureLogs(t)
	before := testutil.ToFloat64(operationsTotal.WithLabelValues("clone", outcomeFailure))

	cmd := CommandContext(ctx, "clone", "/definitely/not/a/repo", "/tmp/gitexec-test-should-fail")
	err := Run(ctx, "clone", cmd)
	require.Error(t, err)

	after := testutil.ToFloat64(operationsTotal.WithLabelValues("clone", outcomeFailure))
	assert.Equal(t, before+1, after)

	out := buf.String()
	assert.Contains(t, out, `"outcome":"failure"`)
	assert.Contains(t, out, `"op":"clone"`)
	assert.Contains(t, out, `"err":`)
}

// Tokens embedded in URLs must not survive into the args field of the log.
// This is the safety property the sanitizer exists for; the end-to-end test
// guards it against future refactors that might bypass sanitizeArgs.
func TestRun_RedactsTokenInArgs(t *testing.T) {
	ctx, buf := captureLogs(t)

	// We don't actually need this to succeed; failure path also logs args.
	cmd := CommandContext(ctx, "ls-remote", "https://x-access-token:SUPERSECRET@example.invalid/o/r.git")
	_ = Run(ctx, "ls-remote", cmd)

	out := buf.String()
	assert.NotContains(t, out, "SUPERSECRET", "credential leaked into log output")
	assert.Contains(t, out, `"repo_host":"example.invalid"`)
}

// Observe is the go-git path. It must record the same observation shape as
// Run so callers can mix exec and go-git in the same metric.
func TestObserve_PathParity(t *testing.T) {
	ctx, buf := captureLogs(t)
	before := testutil.ToFloat64(operationsTotal.WithLabelValues("fetch", outcomeFailure))

	err := Observe(ctx, "fetch", func() error { return errors.New("boom") })
	require.Error(t, err)

	after := testutil.ToFloat64(operationsTotal.WithLabelValues("fetch", outcomeFailure))
	assert.Equal(t, before+1, after)
	assert.Contains(t, buf.String(), `"op":"fetch"`)
	assert.Contains(t, buf.String(), `"outcome":"failure"`)
}

// Pre-existing cmd.Stderr must still receive output (we only tee through our
// tail-capture buffer). This protects callers that already pipe stderr.
func TestRun_PreservesCallerStderr(t *testing.T) {
	ctx, _ := captureLogs(t)
	var callerStderr bytes.Buffer
	cmd := CommandContext(ctx, "clone", "/definitely/not/a/repo", "/tmp/gitexec-test-also-fail")
	cmd.Stderr = &callerStderr
	_ = Run(ctx, "clone", cmd)
	assert.True(t, strings.Contains(callerStderr.String(), "fatal") || callerStderr.Len() > 0,
		"caller's stderr writer must still receive output, got %q", callerStderr.String())
}

// Callers that point Stdout and Stderr at one writer (the combined-output
// idiom) must not provoke a data race. os/exec shares a single copier
// goroutine only while those two fields stay the same writer, so the
// tail-capture wrapping must keep them identical; if it splits them, exec runs
// two goroutines that race writing the shared buffer. Run under -race this
// fails on a regression, and the combined output must still reach the caller.
func TestRun_CombinedOutputSharedWriterNoRace(t *testing.T) {
	ctx, buf := captureLogs(t)

	// Write to both streams so the race detector sees concurrent writes if the
	// dedup were broken; a single stream would not exercise both goroutines.
	cmd := exec.CommandContext(ctx, "sh", "-c", "echo to-stdout; echo to-stderr 1>&2") //nolint:gosec // fixed test args
	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	require.NoError(t, Run(ctx, "status", cmd))

	got := combined.String()
	assert.Contains(t, got, "to-stdout", "stdout must reach the shared writer")
	assert.Contains(t, got, "to-stderr", "stderr must reach the shared writer")
	assert.Contains(t, buf.String(), `"outcome":"success"`)
}

// WithLoggedArgs must substitute the recorded argv while leaving execution,
// metrics, and repo-coordinate derivation on the real argv — the option
// exists so a caller can redact sensitive argv elements (search patterns,
// confidential file paths) from the log line without changing what runs.
func TestOutput_WithLoggedArgs(t *testing.T) {
	ctx, buf := captureLogs(t)

	cmd := CommandContext(ctx, "--version")
	out, err := Output(ctx, "--version", cmd, WithLoggedArgs([]string{"--version", "[redacted]"}))
	require.NoError(t, err)
	assert.NotEmpty(t, out, "stdout must still be captured from the real command")

	logged := buf.String()
	assert.Contains(t, logged, `"[redacted]"`, "the substituted argv must be what lands in the log")
	assert.Contains(t, logged, `"op":"--version"`)
	assert.Contains(t, logged, `"outcome":"success"`)
}

// The substituted argv still passes through credential sanitization — a
// caller redacting one element must not accidentally re-expose a credential
// embedded in another.
func TestRun_WithLoggedArgsStillSanitized(t *testing.T) {
	ctx, buf := captureLogs(t)

	cmd := CommandContext(ctx, "--version")
	err := Run(ctx, "--version", cmd, WithLoggedArgs([]string{
		"fetch", "https://user:hunter2@github.com/org/repo.git", "[redacted]",
	}))
	require.NoError(t, err)

	logged := buf.String()
	assert.NotContains(t, logged, "hunter2", "credentials in substituted args must still be stripped")
	assert.Contains(t, logged, `"[redacted]"`)
}

// WithoutStderrTail must drop stderr_tail from the failure log while keeping
// the error, exit code, and outcome — the failure stays diagnosable without
// echoing stderr content the caller knows to be sensitive.
func TestRun_WithoutStderrTail(t *testing.T) {
	ctx, buf := captureLogs(t)

	// A guaranteed failure whose stderr quotes its argument back.
	cmd := CommandContext(ctx, "rev-parse", "--verify", "sensitive-value")
	cmd.Dir = t.TempDir() // not a repo: fails, stderr mentions the setup
	err := Run(ctx, "rev-parse", cmd, WithoutStderrTail())
	require.Error(t, err)

	logged := buf.String()
	assert.Contains(t, logged, `"outcome":"failure"`)
	assert.Contains(t, logged, `"err":`)
	assert.NotContains(t, logged, "stderr_tail", "stderr_tail must be suppressed")
}
