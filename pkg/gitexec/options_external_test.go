/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"os/exec"
	"testing"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/gitexec"
)

// Type inference lets external callers wrap Option without naming its private parameter type.
func withOptionEffect[T any](option func(T), effect func()) func(T) {
	return func(value T) {
		effect()
		option(value)
	}
}

func TestExecOptionsAppliedOnce(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	for _, mode := range []struct {
		name    string
		capture bool
		custom  bool
	}{
		{name: "native run"},
		{name: "native output", capture: true},
		{name: "custom run", custom: true},
		{name: "custom output", capture: true, custom: true},
	} {
		for _, outcome := range []struct {
			name string
			arg  string
			fail bool
		}{
			{name: "success", arg: "--version"},
			{name: "failure", arg: "--gitexec-invalid-option", fail: true},
		} {
			t.Run(mode.name+"/"+outcome.name, func(t *testing.T) {
				var logs, combined bytes.Buffer
				ctx := clog.WithLogger(t.Context(), clog.New(slog.NewJSONHandler(&logs, nil)))
				cmd := exec.CommandContext(ctx, git, outcome.arg)
				cmd.Stdout, cmd.Stderr = &combined, &combined
				applications, runs, classifications := 0, 0, 0
				repo := "https://example.invalid/" + rand.Text()
				opts := []gitexec.Option{
					withOptionEffect(gitexec.WithRepoURL(repo), func() { applications++ }),
					gitexec.WithFailure(func(err error) gitexec.Failure {
						classifications++
						if err == nil {
							t.Error("failure classifier received nil")
						}
						return gitexec.Failure{Class: "fixture"}
					}),
				}
				if mode.custom {
					opts = append(opts, gitexec.WithCommandRunner(func(c *exec.Cmd) error {
						runs++
						if applications != 1 || c != cmd || c.Stdout != c.Stderr {
							t.Error("runner lost resolved options, command or shared-writer identity")
						}
						return c.Run()
					}))
				}
				var out []byte
				var runErr error
				if mode.capture {
					out, runErr = gitexec.Output(ctx, "option-fixture", cmd, opts...)
				} else {
					runErr = gitexec.Run(ctx, "option-fixture", cmd, opts...)
				}
				if (runErr != nil) != outcome.fail {
					t.Fatalf("execution error: got = %v, want failure = %t", runErr, outcome.fail)
				}
				if applications != 1 {
					t.Errorf("option applications: got = %d, want = 1", applications)
				}
				if mode.custom && runs != 1 || !mode.custom && runs != 0 {
					t.Errorf("custom runner calls: got = %d, selected = %t", runs, mode.custom)
				}
				if outcome.fail && classifications != 1 || !outcome.fail && classifications != 0 {
					t.Errorf("failure classifier calls: got = %d, failed = %t", classifications, outcome.fail)
				}
				if mode.capture && !bytes.Equal(out, combined.Bytes()) {
					t.Errorf("captured output: got = %q, want = %q", out, combined.Bytes())
				}
				var record struct {
					RepoHost string `json:"repo_host"`
					RepoPath string `json:"repo_path"`
				}
				if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
					t.Fatalf("decode single observation: %v", err)
				}
				if record.RepoHost != "example.invalid" || "https://"+record.RepoHost+"/"+record.RepoPath != repo {
					t.Errorf("recorded repository: got = %s/%s, want = %s", record.RepoHost, record.RepoPath, repo)
				}
			})
		}
	}
}

type mutableStatusError struct{ code int }

func (*mutableStatusError) Error() string   { return "mutable status" }
func (e *mutableStatusError) ExitCode() int { return e.code }

func TestObserveAttributionPrecedesOptionEffects(t *testing.T) {
	var logs bytes.Buffer
	ctx := clog.WithLogger(t.Context(), clog.New(slog.NewJSONHandler(&logs, nil)))
	outcome := &mutableStatusError{code: 7}
	applications := 0
	err := gitexec.Observe(ctx, "option-fixture", func() error { return outcome },
		withOptionEffect(gitexec.WithRepoURL("https://example.invalid/repo"), func() {
			applications++
			outcome.code = 9
		}))
	if !errors.Is(err, outcome) || applications != 1 || outcome.code != 9 {
		t.Fatalf("Observe: error = %v, applications = %d, mutated code = %d", err, applications, outcome.code)
	}
	var record struct {
		ExitCode int `json:"exit_code"`
	}
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.ExitCode != 7 {
		t.Errorf("recorded exit code: got = %d, want = 7 before option effects", record.ExitCode)
	}
}

func TestObserveOptionsApplyOnceAfterCallback(t *testing.T) {
	failure := errors.New("fixture failure")
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "failure", err: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			applications, classifications := 0, 0
			finished := false
			err := gitexec.Observe(t.Context(), "option-fixture", func() error {
				if applications != 0 {
					t.Error("options applied before callback")
				}
				finished = true
				return tc.err
			}, withOptionEffect(gitexec.WithRepoURL("https://example.invalid/repo"), func() {
				applications++
				if !finished {
					t.Error("options applied before callback completed")
				}
			}), gitexec.WithFailure(func(err error) gitexec.Failure {
				classifications++
				if !errors.Is(err, tc.err) {
					t.Errorf("classifier error: got = %v, want = %v", err, tc.err)
				}
				return gitexec.Failure{Class: "fixture"}
			}))
			if !errors.Is(err, tc.err) || applications != 1 {
				t.Errorf("Observe: error = %v, applications = %d; want = (%v, 1)", err, applications, tc.err)
			}
			if tc.err != nil && classifications != 1 || tc.err == nil && classifications != 0 {
				t.Errorf("failure classifier calls: got = %d, error = %v", classifications, tc.err)
			}
		})
	}
}
