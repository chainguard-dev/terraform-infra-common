/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package memusage reads and logs Linux process and container memory accounting.
// Log and ReadContainer read accounting on each call and keep no shared state.
// Heartbeat samples in the background and logs while the container is at or
// over half its memory limit. All functions are safe for concurrent use.
package memusage
