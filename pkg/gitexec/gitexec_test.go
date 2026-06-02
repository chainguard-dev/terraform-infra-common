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
