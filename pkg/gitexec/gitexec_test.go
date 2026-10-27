/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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

	// The remote is a loopback server that answers 404: the operation fails,
	// and the failure path logs args just as the success path would.
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	host := srv.Listener.Addr().String()

	cmd := CommandContext(ctx, "ls-remote", "http://x-access-token:SUPERSECRET@"+host+"/o/r.git")
	_ = Run(ctx, "ls-remote", cmd)

	out := buf.String()
	assert.NotContains(t, out, "SUPERSECRET", "credential leaked into log output")
	assert.Contains(t, out, `"repo_host":"`+host+`"`)
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

// WithFailure lets a caller that reads stderr itself say what a failure's
// line carries: its own text in place of the raw tail, a classification,
// and a level. A caller that strips progress from stderr and recognises a
// missing branch would otherwise log a tail of progress, at ERROR, for a
// failure it handles as an outcome. The metric counts the failure however
// it is logged.
func TestRun_WithFailure(t *testing.T) {
	ctx, buf := captureLogs(t)
	before := testutil.ToFloat64(operationsTotal.WithLabelValues("rev-parse", outcomeFailure))

	cmd := CommandContext(ctx, "rev-parse", "--verify", "HEAD")
	cmd.Dir = t.TempDir() // not a repository: fails
	var seen error
	err := Run(ctx, "rev-parse", cmd, WithFailure(func(err error) Failure {
		seen = err
		return Failure{Stderr: "the caller's reading of stderr", Class: "expected_miss", Level: slog.LevelInfo}
	}))
	require.Error(t, err)
	assert.Equal(t, err, seen, "fn must be given the error Run returns")

	after := testutil.ToFloat64(operationsTotal.WithLabelValues("rev-parse", outcomeFailure))
	assert.Equal(t, before+1, after)

	logged := buf.String()
	assert.Contains(t, logged, `"level":"INFO"`)
	assert.Contains(t, logged, `"outcome":"failure"`)
	assert.Contains(t, logged, `"exit_code":128`)
	assert.Contains(t, logged, `"classification":"expected_miss"`)
	assert.Contains(t, logged, `"stderr_tail":"the caller's reading of stderr"`)
	assert.NotContains(t, logged, "fatal", "the raw stderr must give way to the caller's text")
}

// A failure the caller does not recognise keeps ERROR and carries no
// classification field, so its line reads as it would without the option.
func TestRun_WithFailureUnrecognised(t *testing.T) {
	ctx, buf := captureLogs(t)

	cmd := CommandContext(ctx, "rev-parse", "--verify", "HEAD")
	cmd.Dir = t.TempDir()
	err := Run(ctx, "rev-parse", cmd, WithFailure(func(error) Failure {
		return Failure{Stderr: "unrecognised", Level: slog.LevelError}
	}))
	require.Error(t, err)

	logged := buf.String()
	assert.Contains(t, logged, `"level":"ERROR"`)
	assert.Contains(t, logged, `"stderr_tail":"unrecognised"`)
	assert.NotContains(t, logged, "classification")
}

// fn is not consulted for a command that succeeded; there is no failure to
// describe, and a caller's fn may assume one.
func TestRun_WithFailureNotCalledOnSuccess(t *testing.T) {
	ctx, buf := captureLogs(t)

	cmd := CommandContext(ctx, "--version")
	err := Run(ctx, "--version", cmd, WithFailure(func(err error) Failure {
		t.Errorf("fn called for a successful command with %v", err)
		return Failure{}
	}))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"outcome":"success"`)
}

// The caller's text is bounded as the raw tail is, and from the same end:
// a failure's line stays small whatever the caller kept of stderr, and
// git's fatal line comes last.
func TestRun_WithFailureBoundsStderr(t *testing.T) {
	ctx, buf := captureLogs(t)

	cmd := CommandContext(ctx, "rev-parse", "--verify", "HEAD")
	cmd.Dir = t.TempDir()
	long := strings.Repeat("x", 2*stderrTailBytes) + "END"
	err := Run(ctx, "rev-parse", cmd, WithFailure(func(error) Failure {
		return Failure{Stderr: long, Level: slog.LevelError}
	}))
	require.Error(t, err)

	var line struct {
		StderrTail string `json:"stderr_tail"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &line))
	assert.Len(t, line.StderrTail, stderrTailBytes)
	assert.True(t, strings.HasSuffix(line.StderrTail, "END"), "the end of the caller's text must be what is kept, got %q", line.StderrTail)
}

func TestCommandRunnerProcess(_ *testing.T) {
	if os.Getenv("GITEXEC_RUNNER_FIXTURE") != "1" {
		return
	}
	_, _ = os.Stdout.WriteString("out\n")
	_, _ = os.Stderr.WriteString("err\n")
	switch os.Getenv("GITEXEC_RUNNER_OUTCOME") {
	case "exit-one":
		os.Exit(1)
	case "signal":
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			os.Exit(99)
		}
		_ = p.Kill()
		os.Exit(99)
	}
	os.Exit(0)
}

func runnerFixture(ctx context.Context, outcome string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCommandRunnerProcess$")
	cmd.Env = append(os.Environ(), "GITEXEC_RUNNER_FIXTURE=1", "GITEXEC_RUNNER_OUTCOME="+outcome)
	return cmd
}

func TestWithCommandRunner(t *testing.T) {
	for _, mode := range []struct {
		name    string
		capture bool
	}{{name: "run"}, {name: "output", capture: true}} {
		for _, tc := range []struct {
			name    string
			outcome string
			code    int
		}{
			{name: "success", outcome: "success", code: 0},
			{name: "exit-one", outcome: "exit-one", code: 1},
			{name: "signal", outcome: "signal", code: -1},
		} {
			t.Run(tc.name+"/"+mode.name, func(t *testing.T) {
				ctx, logs := captureLogs(t)
				cmd := runnerFixture(ctx, tc.outcome)
				var combined bytes.Buffer
				cmd.Stdout, cmd.Stderr = &combined, &combined
				calls := 0
				opts := []Option{WithCommandRunner(func(c *exec.Cmd) error {
					calls++
					if c != cmd || c.Stdout != c.Stderr {
						t.Error("runner lost the original command or shared-writer identity")
					}
					return c.Run()
				})}
				var err error
				if mode.capture {
					var out []byte
					out, err = Output(ctx, "runner-fixture", cmd, opts...)
					assert.Equal(t, "out\nerr\n", string(out))
				} else {
					err = Run(ctx, "runner-fixture", cmd, opts...)
				}
				assert.Equal(t, tc.code, exitCodeFor(err))
				assert.Equal(t, 1, calls)
				assert.Equal(t, "out\nerr\n", combined.String())
				assert.Equal(t, 1, strings.Count(logs.String(), `"msg":"git_operation"`))
				var observation struct {
					Code int `json:"exit_code"`
				}
				require.NoError(t, json.Unmarshal(logs.Bytes(), &observation))
				assert.Equal(t, tc.code, observation.Code)
			})
		}
	}
}

type runnerStatusError struct{ code int }

func (e runnerStatusError) Error() string { return "validated target outcome" }
func (e runnerStatusError) ExitCode() int { return e.code }

func TestCommandRunnerFailureAttribution(t *testing.T) {
	for _, tc := range []struct {
		name string
		run  func(*exec.Cmd) error
		code int
	}{
		{name: "typed target", run: func(*exec.Cmd) error { return runnerStatusError{code: 7} }, code: 7},
		{name: "target signal", run: func(*exec.Cmd) error { return runnerStatusError{code: -1} }, code: -1},
		{name: "invalid status", run: func(*exec.Cmd) error { return runnerStatusError{code: 300} }, code: -1},
		{name: "containment failure", run: func(*exec.Cmd) error { return errors.New("missing completion proof") }, code: -1},
		{name: "start failure", run: func(cmd *exec.Cmd) error {
			cmd.Path = "/does-not-exist/command"
			return cmd.Run()
		}, code: -1},
		{name: "explicit nil", code: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logs := captureLogs(t)
			cmd := runnerFixture(ctx, "success")
			err := Run(ctx, "runner-failure", cmd, WithCommandRunner(tc.run))
			require.Error(t, err)
			assert.Nil(t, cmd.Process, "a selected runner must not fall back to native execution")
			assert.Equal(t, tc.code, exitCodeFor(err))
			var observation struct {
				Code int `json:"exit_code"`
			}
			require.NoError(t, json.Unmarshal(logs.Bytes(), &observation))
			assert.Equal(t, tc.code, observation.Code)
		})
	}
}
