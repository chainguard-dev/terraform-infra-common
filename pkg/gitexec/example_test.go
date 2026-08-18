/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec_test

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/pkg/gitexec"
)

func ExampleCommandContext() {
	ctx := context.Background()
	cmd := gitexec.CommandContext(ctx, "clone", "--depth=1", "https://github.com/example/repo", "/tmp/repo")
	_ = cmd
}

func ExampleRun() {
	ctx := context.Background()
	repoURL := "https://github.com/example/repo"
	cmd := gitexec.CommandContext(ctx, "clone", "--depth=1", repoURL, "/tmp/repo")
	if err := gitexec.Run(ctx, "clone", cmd, gitexec.WithRepoURL(repoURL)); err != nil {
		fmt.Println("clone failed:", err)
	}
}

func ExampleOutput() {
	ctx := context.Background()
	cmd := gitexec.CommandContext(ctx, "rev-parse", "HEAD")
	cmd.Dir = "/tmp/repo"
	out, err := gitexec.Output(ctx, "rev-parse", cmd)
	if err != nil {
		fmt.Println("rev-parse failed:", err)
		return
	}
	fmt.Printf("HEAD: %s", out)
}

func ExampleObserve() {
	ctx := context.Background()
	// Observe wraps a non-exec git operation — e.g. a go-git call not yet
	// covered by the gogit shim — and records the same observation shape as Run.
	err := gitexec.Observe(ctx, "custom-op", func() error {
		// perform the operation
		return nil
	}, gitexec.WithRepoURL("https://github.com/example/repo"))
	if err != nil {
		fmt.Println("operation failed:", err)
	}
}

func ExampleWithRepoURL() {
	ctx := context.Background()
	repoURL := "https://github.com/example/repo"
	cmd := gitexec.CommandContext(ctx, "fetch", "origin")
	cmd.Dir = "/tmp/repo"
	// WithRepoURL enriches the log line with repo_host and repo_path when the
	// command argv contains a local path rather than a URL.
	if err := gitexec.Run(ctx, "fetch", cmd, gitexec.WithRepoURL(repoURL)); err != nil {
		fmt.Println("fetch failed:", err)
	}
}

func ExampleWithLoggedArgs() {
	ctx := context.Background()
	sensitivePattern := "secret-path"
	cmd := gitexec.CommandContext(ctx, "log", "--", sensitivePattern)
	cmd.Dir = "/tmp/repo"
	// WithLoggedArgs substitutes the logged argv to redact sensitive values.
	if err := gitexec.Run(ctx, "log", cmd,
		gitexec.WithLoggedArgs([]string{"log", "--", "[REDACTED]"}),
		gitexec.WithoutStderrTail(),
	); err != nil {
		fmt.Println("log failed:", err)
	}
}

func ExampleWithoutStderrTail() {
	ctx := context.Background()
	sensitivePattern := "secret-path"
	cmd := gitexec.CommandContext(ctx, "log", "--", sensitivePattern)
	cmd.Dir = "/tmp/repo"
	// WithoutStderrTail omits stderr_tail from failure log lines to prevent
	// sensitive values from leaking via git's error messages.
	if err := gitexec.Run(ctx, "log", cmd,
		gitexec.WithLoggedArgs([]string{"log", "--", "[REDACTED]"}),
		gitexec.WithoutStderrTail(),
	); err != nil {
		fmt.Println("log failed:", err)
	}
}
