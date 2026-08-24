/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package tracesplit re-roots a unit of work into its own trace, paired with
// a handoff span in the caller's trace.
package tracesplit
