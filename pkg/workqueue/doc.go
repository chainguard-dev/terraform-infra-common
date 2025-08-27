/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Generate the proto definitions
//go:generate protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative workqueue.proto

/*
Package workqueue provides a distributed workqueue abstraction for processing tasks
across multiple replicas with concurrency control, retry logic, and comprehensive monitoring.

# Overview

The workqueue package implements a work distribution system built on top of Google Cloud Storage
that enables stateless services to coordinate work processing across multiple replicas. It provides
features essential for reliable distributed task processing including priority queues, retry logic
with exponential backoff, dead letter queues, and detailed Prometheus metrics.

# Key Features

- Concurrency Control: Manages concurrent work processing across multiple replicas
- Priority Support: Tasks can be queued with different priority levels for processing order
- Retry Logic: Configurable retry attempts with exponential backoff for failed tasks
- Dead Letter Queue: Failed tasks exceeding retry limits are moved to dead letter queue
- Comprehensive Metrics: Detailed Prometheus metrics for monitoring and alerting
- Not Before Scheduling: Tasks can be scheduled to run at specific future times

# Basic Usage

	// Create a workqueue using GCS backend
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	wq := gcs.NewWorkQueue(client.Bucket("my-workqueue-bucket"), 10)

	// Queue work with options
	err = wq.Queue(ctx, "task-key", workqueue.Options{
		Priority: 100,
		NotBefore: time.Now().Add(5 * time.Minute),
	})

	// Enumerate and process work
	inProgress, queued, err := wq.Enumerate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	for _, key := range queued {
		owned, err := key.Own(ctx)
		if err != nil {
			continue
		}

		// Process the work...
		if err := processWork(owned.Name()); err != nil {
			owned.Requeue(ctx) // Will retry with backoff
		} else {
			owned.Done(ctx) // Mark as complete
		}
	}

# Retry Behavior

The workqueue supports configurable maximum retry limits through the WORKQUEUE_MAX_RETRY
environment variable. When a task exceeds the maximum retry count, it is moved to a dead
letter queue instead of being requeued. Retry attempts use exponential backoff based on
the attempt count and the BackoffPeriod constant.

# Metrics and Monitoring

The GCS implementation exposes comprehensive Prometheus metrics including:

Queue State Metrics:
- workqueue_in_progress_keys: Number of keys currently being processed
- workqueue_queued_keys: Number of keys in the backlog
- workqueue_notbefore_keys: Number of keys waiting on a 'not before' timestamp
- workqueue_dead_lettered_keys: Number of keys in the dead letter queue

Retry Attempt Metrics:
- workqueue_max_attempts: Maximum number of attempts for any queued or in-progress task
- workqueue_task_max_attempts: Per-task attempt count for tasks with >20 attempts
- workqueue_retry_attempts: Histogram distribution of retry attempts across all tasks

Latency Metrics:
- workqueue_process_latency_seconds: Duration to process a key
- workqueue_wait_latency_seconds: Duration a key waited before processing

Throughput Metrics:
- workqueue_added_keys: Total number of queue requests
- workqueue_deduped_keys: Total number of keys that were deduplicated

Example Prometheus queries:

	# 90th percentile of retry attempts
	histogram_quantile(0.9, workqueue_retry_attempts)

	# Tasks with excessive retries
	workqueue_task_max_attempts > 50

# Integration

This package is designed to work with the terraform-infra-common workqueue module, which
provides the infrastructure setup including Cloud Storage buckets, Cloud Scheduler jobs,
Pub/Sub subscriptions, and Cloud Run services for processing work.

The dispatcher service uses this package to implement work distribution, while user services
implement the reconciler logic that processes individual work items.
*/
package workqueue
