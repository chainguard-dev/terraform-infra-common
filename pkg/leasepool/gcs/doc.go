/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package gcs implements named lease pools using conditional Cloud Storage writes.
// Each pool uses a dedicated bucket or prefix. Callers own the storage client.
package gcs
