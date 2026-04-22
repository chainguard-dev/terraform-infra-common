/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package check provides utilities for creating and updating GitHub Check Runs.
//
// Use [NewBuilder] to construct a check run result incrementally. Write
// formatted output with [Builder.Writef], set the [Builder.Status] and
// [Builder.Conclusion] fields, then call [Builder.CheckRunCreate] or
// [Builder.CheckRunUpdate] to produce the GitHub API options struct.
//
// Output is automatically truncated to GitHub's maximum check run output
// length of 65536 bytes.
package check
