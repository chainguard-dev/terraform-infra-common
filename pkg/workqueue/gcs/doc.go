/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package gcs implements a Google Cloud Storage backed workqueue that provides
// reliable, persistent task processing with state management.
//
// # Overview
//
// The GCS workqueue uses object prefixes to organize tasks by their state within
// a single bucket. Tasks flow through three states: queued, in-progress, and
// dead-letter. The implementation provides priority-based processing, lease-based
// ownership, automatic retry with backoff, and dead letter handling.
//
// # Bucket Organization
//
// Tasks are organized using the following prefixes:
//
//   - queued/ - Tasks waiting to be processed
//   - in-progress/ - Tasks currently being processed by a worker
//   - dead-letter/ - Tasks that have failed after exceeding maximum retry attempts
//
// # Features
//
//   - Priority-based processing: Higher priority tasks are processed first
//   - Lease-based ownership: In-progress tasks have renewable leases to prevent
//     multiple workers from processing the same task
//   - Automatic retry with backoff: Failed tasks are automatically requeued with
//     exponential backoff
//   - Dead letter handling: Tasks exceeding retry limits are moved to a dead letter queue
//   - Orphan detection: Detects and handles tasks with expired leases
//   - Deduplication: Duplicate queue requests update priority/timing instead of
//     creating duplicates
//
// # Usage
//
// Create a new workqueue using NewWorkQueue with a GCS bucket handle:
//
//	client, err := storage.NewClient(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	wq := gcs.NewWorkQueue(client.Bucket("my-bucket"), 100)
//
// Queue a task:
//
//	err := wq.Queue(ctx, "task-key", workqueue.Options{
//	    Priority: 10,
//	})
//
// Enumerate and process tasks:
//
//	inProgress, queued, deadLettered, err := wq.Enumerate(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, key := range queued {
//	    owned, err := key.Start(ctx)
//	    if err != nil {
//	        continue
//	    }
//	    // Process the task...
//	    if err := owned.Complete(ctx); err != nil {
//	        log.Printf("failed to complete: %v", err)
//	    }
//	}
//
// # Configuration
//
// The package exposes the following configuration variables:
//
//   - RefreshInterval: The period on which leases are refreshed (default: 5 minutes)
//   - TrackWorkAttemptMinThreshold: Minimum attempts before tracking in metrics (default: 20)
//
// # Metrics
//
// The implementation exports Prometheus metrics for monitoring:
//
//   - workqueue_in_progress_keys: Number of keys currently being processed
//   - workqueue_queued_keys: Number of keys in the backlog
//   - workqueue_notbefore_keys: Number of keys waiting on a 'not before' time
//   - workqueue_dead_lettered_keys: Number of keys in the dead letter queue
//   - workqueue_process_latency_seconds: Duration taken to process a key
//   - workqueue_wait_latency_seconds: Duration the key waited to start
//   - workqueue_added_keys: Total number of queue requests
//   - workqueue_deduped_keys: Total number of deduplicated keys
//   - workqueue_max_attempts: Maximum attempts for any queued or in-progress task
//   - workqueue_time_to_completion_seconds: Time from first queue to final outcome
//
// All metrics include service_name and revision_name labels derived from
// K_SERVICE and K_REVISION environment variables.
//
// # Thread Safety
//
// The workqueue implementation is safe for concurrent use. Multiple goroutines
// can queue, enumerate, and process tasks simultaneously. The lease-based
// ownership model ensures that only one worker processes a given task at a time.
package gcs
