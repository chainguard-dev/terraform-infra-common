/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package schemas defines Go types for GitHub webhook event payloads as
// stored in BigQuery.
//
// The [Wrapper] type wraps any event body with metadata including the
// delivery timestamp and GitHub webhook headers. Event-specific types such
// as [PullRequest], [Repository], [User], and [Installation] mirror the
// structure of GitHub webhook payloads using BigQuery-compatible field types.
package schemas
