/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package memusage logs Linux process and container memory high-water marks.
// Accounting is read on each call; no background sampling or shared state is used.
// Log is safe for concurrent use.
package memusage
