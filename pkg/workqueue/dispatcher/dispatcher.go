/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/clog"
	"golang.org/x/sync/errgroup"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

// Callback is the function that Handle calls to process a particular key.
type Callback func(ctx context.Context, key string, opts workqueue.Options) error

// ServiceCallback returns a Callback that invokes the given service.
func ServiceCallback(client workqueue.WorkqueueServiceClient) Callback {
	return func(ctx context.Context, key string, opts workqueue.Options) error {
		_, err := client.Process(ctx, &workqueue.ProcessRequest{
			Key:      key,
			Priority: opts.Priority,
		})
		return err
	}
}

// Future is a function that can be used to block on the result of a round of
// dispatching work.
type Future func() error

// Handle is a synchronous form of HandleAsync.
func Handle(ctx context.Context, wq workqueue.Interface, concurrency int, f Callback) error {
	return HandleAsync(ctx, wq, concurrency, f, 0)()
}

// HandleAsync initiates a single iteration of the dispatcher, possibly invoking
// the callback for several different keys.  It returns a future that can be
// used to block on the result.
func HandleAsync(ctx context.Context, wq workqueue.Interface, concurrency int, f Callback, maxRetry int) Future {
	// Enumerate the state of the queue.
	wip, next, err := wq.Enumerate(ctx)
	if err != nil {
		return func() error { return fmt.Errorf("enumerate() = %w", err) }
	}

	eg := errgroup.Group{}

	// Remove any orphaned work by returning it to the queue.
	activeKeys := make(map[string]struct{}, len(wip))
	for _, x := range wip {
		if !x.IsOrphaned() {
			activeKeys[x.Name()] = struct{}{}
			continue
		}
		eg.Go(func() error {
			return x.Requeue(ctx)
		})
	}

	// If our open slots are filled, then we can't launch any new work!
	// We explicitly check this here because if nWIP grows larger than
	// concurrency then the subtraction below will underflow and we'll
	// start to queue work without bounds.
	nWIP := len(activeKeys)
	if nWIP >= concurrency {
		return eg.Wait // Should generally be a no-op.
	}

	// Attempt to launch a new piece of work for each open slot we have available
	// which is: N - active.
	openSlots := concurrency - nWIP
	idx, launched := 0, 0
	for ; idx < len(next) && launched < openSlots; idx++ {
		nextKey := next[idx]

		// If the next key is already in progress, then move to the next candidate.
		if _, ok := activeKeys[nextKey.Name()]; ok {
			continue
		}

		// At this point, we know that nextKey gets launched.  There are two paths below:
		// 1. One is where we lose the race and someone else launches it, and
		// 2. The other is where we launch it.
		// By incrementing the counter here, we ensure we don't overlaunch keys due to a race.
		launched++

		// This is done in a Go routine so that we can process keys concurrently.
		eg.Go(func() error {
			// Start the work, moving it to be in-progress. If we are unsuccessful starting
			// the work, then someone beat us to it, so move on to the next key.
			oip, err := nextKey.Start(ctx)
			if err != nil {
				clog.DebugContextf(ctx, "Failed to start key %q: %v", nextKey.Name(), err)
				return nil
			}

			// Attempt to perform the actual reconciler invocation.
			if err := f(oip.Context(), oip.Name(), workqueue.Options{
				Priority: oip.Priority(),
			}); err != nil {
				clog.WarnContextf(ctx, "Failed callback for key %q: %v", oip.Name(), err)
				attempts := oip.GetAttempts()

				// If maxRetry is configured and we've reached or exceeded it, use Fail() instead of Requeue()
				if maxRetry > 0 && attempts >= maxRetry {
					clog.InfoContextf(ctx, "Key %q has reached max retry limit (%d/%d), failing permanently",
						oip.Name(), attempts, maxRetry)

					if err := oip.Fail(ctx); err != nil {
						return fmt.Errorf("fail(after reaching max retries) = %w", err)
					}
				} else {
					if err := oip.Requeue(ctx); err != nil {
						return fmt.Errorf("requeue(after failed callback) = %w", err)
					}
				}
				return nil // This isn't an error in the dispatcher itself.
			}
			// Delete the in-progress key (stops heartbeat).
			if err := oip.Complete(ctx); err != nil {
				return fmt.Errorf("complete() = %w", err)
			}
			return nil
		})
	}
	clog.InfoContextf(ctx, "Launched %d new keys (wip: %d)", launched, nWIP)

	// Return the future to wait on outstanding work.
	return eg.Wait
}
