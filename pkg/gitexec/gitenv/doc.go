/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package gitenv builds the process environment for a git subprocess that
// trusts nothing on the host and carries its credential only to the remote
// it is meant for.
//
// The environment disables system and global git configuration, so only
// command-line flags, the configuration passed through the environment and
// the repository's own .git/config apply; disables terminal prompting; pins
// the transport protocols git may speak; and neutralises hooks, the
// credential helper and the HTTP proxy through configuration passed in the
// environment (GIT_CONFIG_COUNT with its numbered keys and values), which git
// reads at command scope, outranking the repository's config. A remote's
// credential travels as an Authorization header scoped to that remote's
// URL, so a url.<base>.insteadOf rewrite planted in .git/config cannot carry
// it to another host.
//
// Configuration through the environment needs git 2.31, and
// GIT_CONFIG_GLOBAL git 2.32; older git ignores both silently, so the hook
// and config defences would fail open rather than closed. CheckVersion
// refuses such a git; callers run it once at startup and refuse to serve.
package gitenv
