/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package workqueue

import (
	"context"
	"time"
)

// Note that these are variables, so that they can be modified by tests and
// made flags in binary entrypoints.
var (
	// BackoffPeriod is the unit of backoff used when requeueing keys.
	// This unit is combined with the number of attempts to determine the
	// wait period before a key should be reprocessed.
	BackoffPeriod = 30 * time.Second

	// MaximumBackoffPeriod is a cap on the period a key must wait before
	// being retried.
	MaximumBackoffPeriod = 10 * time.Minute
)

// Interface is the interface that workqueue implementations must implement.
type Interface interface {
	// Queue adds an item to the workqueue.
	Queue(ctx context.Context, key string, opts Options) error

	// Enumerate returns:
	// - a list of all of the in-progress keys,
	// - a list of the next "N" keys in the queue (according to its configured ordering),
	// - a list of all dead-lettered keys, or
	// - an error if the workqueue is unable to enumerate the keys.
	Enumerate(ctx context.Context) ([]ObservedInProgressKey, []QueuedKey, []DeadLetteredKey, error)

	// Get retrieves the current state and metadata for a specific key.
	Get(ctx context.Context, key string) (*KeyState, error)
}

// Options is a set of options that can be passed when queuing a key.
type Options struct {
	// Priority is the priority of the key.
	// Higher values are processed first.
	Priority int64

	// NotBefore is the earliest time that the key should be processed.
	// When deduplicating, the oldest time is used.
	NotBefore time.Time

	// Delay is an optional duration to wait before processing the key.
	// This is used when requeueing with a custom delay.
	Delay time.Duration
}

// Key is a shared interface that all key types must implement.
type Key interface {
	// Name is the name of the key.
	Name() string

	// Priority is the priority of the key.
	Priority() int64
}

// QueuedKey is a key that is in the queue, waiting to be processed.
type QueuedKey interface {
	Key

	// Start initiates processing of the key, returning an OwnedInProgressKey
	// on success and an error on failure.
	Start(context.Context) (OwnedInProgressKey, error)
}

// InProgressKey is a shared interface that all in-progress key types must implement.
type InProgressKey interface {
	Key

	// Requeue returns this key to the queue.
	Requeue(context.Context) error

	// RequeueWithOptions returns this key to the queue with custom options.
	RequeueWithOptions(context.Context, Options) error
}

// ObservedInProgressKey is a key that we have observed to be in progress,
// but that we are not the owner of.
type ObservedInProgressKey interface {
	InProgressKey

	// IsOrphaned checks whether the key has been orphaned by it's owner.
	IsOrphaned() bool
}

// OwnedInProgressKey is an in-progress key where we have initiated the work,
// and own until it completes either successfully (Complete),
// or unsuccessfully (Requeue or Fail).
type OwnedInProgressKey interface {
	InProgressKey

	// Complete marks the key as successfully completed, and removes it from
	// the in-progress key set.
	Complete(context.Context) error

	// Deadletter permanently removes this key from the queue, indicating it has
	// failed after exceeding the maximum retry attempts.
	Deadletter(context.Context) error

	// GetAttempts returns the current attempt count for the key.
	GetAttempts() int

	// Context is the context of the process heartbeating the key.
	Context() context.Context
}

// DeadLetteredKey is a key that has been moved to the dead-letter queue
// after exceeding the maximum retry attempts.
type DeadLetteredKey interface {
	Key

	// GetFailedTime returns the time when the key was dead-lettered.
	GetFailedTime() time.Time

	// GetAttempts returns the number of attempts before the key was dead-lettered.
	GetAttempts() int
}
