/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package leasepool

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrBusy means another attempt owns the named lease. It is not a storage error.
	ErrBusy = errors.New("lease is busy")
	// ErrLost means ownership expired, changed, or could not be safely renewed.
	ErrLost = errors.New("lease ownership lost")
	// ErrReleased is the context cancellation cause after explicit release.
	ErrReleased = errors.New("lease released")
)

// Interface is a concurrency-safe pool of independent named leases.
type Interface interface {
	// TryAcquire makes one acquisition attempt, without waiting for an owner to
	// release. Names must be nonempty and at most 512 bytes. Storage operations
	// obey ctx. ErrBusy denotes contention. ErrLost can occur if a confirmed
	// acquisition expires before it can be handed to the caller. Such confirmed
	// acquisitions are conditionally released; an ambiguous write may leave a
	// lease busy until its TTL expires.
	TryAcquire(ctx context.Context, name string) (Lease, error)
}

// Lease renews automatically until its parent context ends, ownership is lost,
// or Release is called. Filesystem leases use OS locks without renewal or expiry.
// Methods are safe for concurrent use.
type Lease interface {
	// Context is canceled on ownership loss or release, or with the parent.
	// context.Cause distinguishes these cases. Cancellation is cooperative; it
	// cannot stop an arbitrarily paused process or revoke external API requests.
	Context() context.Context
	// Release stops renewal and conditionally releases this acquisition. Call it
	// only after protected work has stopped, using a fresh bounded context if
	// the lease context has ended. Repeated calls return the first result.
	// The first call's context bounds release. Renewal drain and the subsequent
	// conditional write each also have an independent AcquireTimeout budget.
	// Failure leaves the record to expire; it never releases a successor's lease.
	Release(context.Context) error
}

// Config controls lease timing. Zero fields select defaults. All participants
// must agree on ClockSkew, an upper bound on pairwise wall-clock difference.
// Lease holders use monotonic local deadlines; contenders wait ClockSkew beyond
// the recorded expiration. Clock jumps beyond that assumption are unsupported.
type Config struct {
	AcquireTimeout time.Duration // Default 5 seconds; bounds acquisition I/O and each release phase.
	TTL            time.Duration // Default 30 seconds.
	RenewInterval  time.Duration // Default TTL / 3.
	ClockSkew      time.Duration // Default 2 seconds.
}

// Run acquires a lease, runs work with its ownership context, and releases it
// after work returns. Bound tenure with ctx; work must honor cancellation.
// ErrBusy is returned without calling work. Renewal uncertainty fails closed.
func Run(ctx context.Context, pool Interface, name string, work func(context.Context) error) (retErr error) {
	l, err := pool.TryAcquire(ctx, name)
	if err != nil {
		return err
	}
	defer func() {
		cause := context.Cause(l.Context())
		// Release bounds its drain and write independently. An outer timeout
		// here would spend the write budget waiting for an in-flight renewal.
		releaseErr := l.Release(context.WithoutCancel(ctx))
		if errors.Is(cause, ErrLost) && errors.Is(releaseErr, ErrLost) {
			releaseErr = nil
		}
		retErr = errors.Join(retErr, cause, releaseErr)
	}()
	return work(l.Context())
}
