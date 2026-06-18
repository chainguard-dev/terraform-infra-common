/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package gitexec wraps local git invocations so every operation emits one
// structured clog line and updates a small set of Prometheus metrics.
//
// The package is intended for any code in mono that shells out to git or
// drives go-git, so we can measure how often we clone, fetch, push, pull,
// and so on, and which services drive that traffic.
//
// # Exec form
//
//	cmd := gitexec.CommandContext(ctx, "clone",
//		"--filter=blob:none", "--depth=1", repoURL, dir)
//	if err := gitexec.Run(ctx, "clone", cmd, gitexec.WithRepoURL(repoURL)); err != nil {
//		return err
//	}
//
// # go-git form
//
// Prefer the gogit subpackage: it is a drop-in for the network-performing
// subset of go-git, so callers keep writing ordinary go-git and clone, fetch,
// and push are observed without per-call instrumentation.
//
//	repo, err := gogit.PlainCloneContext(ctx, dir, false, &git.CloneOptions{URL: repoURL})
//	...
//	err = repo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin"})
//
// Observe is the low-level primitive gogit is built on. Reach for it directly
// only to record a go-git operation the shim does not yet cover:
//
//	err := gitexec.Observe(ctx, "fetch", func() error {
//		return repo.FetchContext(ctx, &git.FetchOptions{...})
//	}, gitexec.WithRepoURL(repoURL))
//
// # Emitted observability
//
// Each call records:
//
//   - One clog line with message "git_operation" carrying: op, args,
//     repo_host, repo_path, duration_ms, exit_code, outcome.
//     On failure the line also carries err and stderr_tail.
//
//   - Counter git_operations_total{op,outcome}.
//
//   - Histogram git_operation_duration_seconds{op}.
//
// Cardinality is deliberately bounded: only op and outcome are metric
// labels. Per-repo detail lives in logs only.
package gitexec
