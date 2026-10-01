/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package filesystem coordinates leasepool callers through advisory file locks
// on one host. All participants must use the same trusted, stable directory on
// a local filesystem. Locks last until Release or process death; they have no
// TTL and remain held after parent cancellation while protected work stops.
// Do not unlink lock files or replace the directory while participants run.
// This backend supports Linux and macOS; it is not distributed coordination.
package filesystem
