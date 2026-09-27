/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package memory implements a process-local lease pool for tests and local use.
// Share one pool among participants; separate pools have independent namespaces.
// It uses the same renewal, expiration, and cancellation protocol as GCS.
package memory
