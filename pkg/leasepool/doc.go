/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package leasepool provides named, renewable leases for cooperative leadership.
// A pool contains independent leases; sharing a pool and name coordinates callers.
// Ownership is not a fencing guarantee for external side effects. Protected work
// must stop when the lease context is canceled, and finish before Release.
package leasepool
