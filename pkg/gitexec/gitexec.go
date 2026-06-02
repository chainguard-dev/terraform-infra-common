/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"time"

	"github.com/chainguard-dev/clog"
)

// stderrTailBytes bounds how much trailing stderr we attach to a failure log.
// Keep it small: this lands in log storage on every failure.
const stderrTailBytes = 512

// Option customizes how a single observation is recorded.
type Option func(*options)

type options struct {
	repoHost string
	repoPath string
	repoURL  string
}

// WithRepoURL sets the remote URL associated with the operation. It is parsed
// to derive repo_host and repo_path log fields and is the only reliable source
// of those fields for operations whose argv contains a local path rather than
// a URL (e.g. push from a working tree).
func WithRepoURL(rawURL string) Option {
	return func(o *options) { o.repoURL = rawURL }
}

// CommandContext returns an *exec.Cmd configured to invoke "git" with the
// given arguments. The first argument SHOULD be the git subcommand
// (e.g. "clone", "fetch", "push"). Callers may further configure the returned
// Cmd (Dir, Env, Stdout, Stderr) before passing it to Run or Output.
func CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "git", args...)
}

// Run executes cmd, records one observation (log + metric), and returns
// cmd.Run's error. op is the git subcommand used for the metric label and
// the log "op" field.
func Run(ctx context.Context, op string, cmd *exec.Cmd, opts ...Option) error {
	_, err := run(ctx, op, cmd, false, opts...)
	return err
}

// Output runs cmd, returns its captured stdout, and records one observation.
// Use this instead of Run when the caller would otherwise call cmd.Output().
func Output(ctx context.Context, op string, cmd *exec.Cmd, opts ...Option) ([]byte, error) {
	return run(ctx, op, cmd, true, opts...)
}

// Observe wraps a non-exec git operation (e.g. a go-git call) and records the
// same observation shape as Run. Use this to keep go-git callers in the same
// metric/log namespace as exec callers.
func Observe(ctx context.Context, op string, fn func() error, opts ...Option) error {
	started := time.Now()
	err := fn()
	record(ctx, op, nil, started, exitCodeFor(err), nil, err, opts...)
	return err
}

func run(ctx context.Context, op string, cmd *exec.Cmd, captureStdout bool, opts ...Option) ([]byte, error) {
	stderrTail := newTailBuffer(stderrTailBytes)
	if cmd.Stderr == nil {
		cmd.Stderr = stderrTail
	} else {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, stderrTail)
	}

	var stdoutBuf *bytes.Buffer
	if captureStdout {
		stdoutBuf = &bytes.Buffer{}
		if cmd.Stdout == nil {
			cmd.Stdout = stdoutBuf
		} else {
			cmd.Stdout = io.MultiWriter(cmd.Stdout, stdoutBuf)
		}
	}

	started := time.Now()
	err := cmd.Run()
	record(ctx, op, argsAfterProgram(cmd), started, exitCodeFor(err), stderrTail.Bytes(), err, opts...)

	if captureStdout {
		return stdoutBuf.Bytes(), err
	}
	return nil, err
}

func record(ctx context.Context, op string, args []string, started time.Time, exitCode int, stderrTail []byte, err error, opts ...Option) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	if o.repoURL != "" {
		o.repoHost, o.repoPath = repoFromArgs([]string{o.repoURL})
	}
	if o.repoHost == "" {
		o.repoHost, o.repoPath = repoFromArgs(args)
	}

	duration := time.Since(started)
	outcome := outcomeSuccess
	if err != nil {
		outcome = outcomeFailure
	}

	operationsTotal.WithLabelValues(op, outcome).Inc()
	operationDuration.WithLabelValues(op).Observe(duration.Seconds())

	fields := []any{
		"op", op,
		"args", sanitizeArgs(args),
		"repo_host", o.repoHost,
		"repo_path", o.repoPath,
		"duration_ms", duration.Milliseconds(),
		"exit_code", exitCode,
		"outcome", outcome,
	}
	if err != nil {
		fields = append(fields, "err", err.Error())
		if len(stderrTail) > 0 {
			fields = append(fields, "stderr_tail", string(stderrTail))
		}
		clog.ErrorContext(ctx, "git_operation", fields...)
		return
	}
	clog.InfoContext(ctx, "git_operation", fields...)
}

// argsAfterProgram returns cmd.Args without the leading program name.
// Returns nil if cmd is nil (Observe path).
func argsAfterProgram(cmd *exec.Cmd) []string {
	if cmd == nil || len(cmd.Args) == 0 {
		return nil
	}
	return cmd.Args[1:]
}

func exitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
