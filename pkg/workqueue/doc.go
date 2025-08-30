/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Generate the proto definitions
//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative workqueue.proto

// Package workqueue contains an interface for a simple key
// workqueue abstraction.
//
// # Metrics
//
// The GCS implementation exports the following Prometheus metrics:
//
//   - workqueue_in_progress_keys: The number of keys currently being processed
//   - workqueue_queued_keys: The number of keys currently in the backlog
//   - workqueue_notbefore_keys: The number of keys waiting on a 'not before' time
//   - workqueue_max_attempts: The maximum number of attempts for any queued or in-progress task
//   - workqueue_task_max_attempts: The maximum number of attempts for a given task above 20
//   - workqueue_process_latency_seconds: The duration taken to process a key
//   - workqueue_wait_latency_seconds: The duration the key waited to start
//   - workqueue_added_keys: The total number of queue requests
//   - workqueue_deduped_keys: The total number of keys that were deduped
//   - workqueue_attempts_at_completion: The number of attempts for successfully completed tasks
//   - workqueue_dead_lettered_keys: The number of keys currently in the dead letter queue
//   - workqueue_time_to_completion_seconds: The time from first queue to final outcome (success or dead-letter). The metric captures the full lifecycle duration including all retry attempts and backoff delays.
//
// All metrics include service_name and revision_name labels. Additional labels vary by metric.
package workqueue
