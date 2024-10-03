/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package workqueue

import "context"

// Interface is the interface that workqueue implementations must implement.
type Interface interface {
	// Queue adds an item to the workqueue.
	Queue(ctx context.Context, key string) error

	// Enumerate returns:
	// - a list of all of the in-progress keys, and
	// - a list of the next "N" keys in the queue (according to its configured ordering), or
	// - an error if the workqueue is unable to enumerate the keys.
	Enumerate(ctx context.Context) ([]ObservedInProgressKey, []QueuedKey, error)
}

// Key is a shared interface that all key types must implement.
type Key interface {
	// Name is the name of the key.
	Name() string
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
// or unsuccessfully (Requeue).
type OwnedInProgressKey interface {
	InProgressKey

	// Complete marks the key as successfully completed, and removes it from
	// the in-progress key set.
	Complete(context.Context) error

	// Context is the context of the process heartbeating the key.
	Context() context.Context
}
