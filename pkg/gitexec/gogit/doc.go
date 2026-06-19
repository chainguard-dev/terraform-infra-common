/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package gogit is a thin observability shim over go-git: a drop-in for the
// subset of github.com/go-git/go-git/v5 that performs network I/O, so callers
// keep writing ordinary go-git and every clone, fetch, and push is recorded
// through gitexec without per-call instrumentation.
//
// Swap the constructor that produces the object you call network methods on,
// and the calls themselves read exactly as they did against go-git:
//
//	repo, err := gogit.PlainCloneContext(ctx, dir, false, &git.CloneOptions{URL: url})
//	...
//	err = repo.FetchContext(ctx, &git.FetchOptions{RemoteName: "origin"})
//	err = repo.PushContext(ctx, &git.PushOptions{RemoteName: "origin", RefSpecs: refs})
//
// A *Repository embeds *git.Repository, so every local operation (Worktree,
// Head, Reference, CommitObject, Storer, ...) is promoted unchanged; only the
// network methods are shadowed by observed versions of the same signature. Use
// Wrap to adapt a *git.Repository built elsewhere (e.g. git.Open over a custom
// storer) and NewRemote in place of git.NewRemote.
//
// # Coverage and its limits
//
// Observed entry points: PlainCloneContext, CloneContext,
// (*Repository).FetchContext, (*Repository).PushContext,
// (*Remote).FetchContext, (*Remote).PushContext.
//
// Coverage is deliberately partial: it spans the operations mono uses today,
// not all of go-git. These also touch the network but are NOT yet shadowed
// here, so they currently run unobserved:
//
//   - the non-context variants Fetch and Push on *Repository and *Remote.
//     Because the wrapper types embed go-git's, these promote to the embedded
//     method and compile cleanly — a silent bypass, not a build error.
//   - (*Remote).List and ListContext (ls-remote).
//   - (*Worktree).Pull and PullContext (Worktree is not wrapped at all).
//   - the non-context Clone and PlainClone constructors.
//
// When you need one of these observed, ADD the matching wrapper to this
// package — shadow the method on the wrapper type, or add the constructor,
// routing it through observe. Do NOT hand-wrap the call in gitexec.Observe at
// the call site: that per-call instrumentation is exactly what this shim
// exists to remove, and scattering it again splits go-git observability back
// across two patterns.
//
// Wrapping is also only as complete as the objects you obtain through this
// package. A network method reached via a *git.Repository or *git.Remote that
// did not come from a gogit constructor (or Wrap) is the plain go-git method
// and is not observed. In particular (*Repository).Remote returns a plain
// *git.Remote, so fetch or push on it bypasses this shim — build such remotes
// with NewRemote.
//
// repo_host and repo_path are derived automatically — from CloneOptions.URL for
// clones and from the remote's configured URL for fetch and push — so callers
// do not need gitexec.WithRepoURL.
//
// NoErrAlreadyUpToDate is returned to the caller unchanged but recorded as a
// successful no-op: the network round-trip happened and found nothing to
// transfer. This matches how hand-written gitexec.Observe call sites treated it.
package gogit
