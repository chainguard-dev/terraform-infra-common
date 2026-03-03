/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs_test

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/storage"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/gcs"
)

// ExampleNewWorkQueue demonstrates creating a new GCS-backed workqueue.
func ExampleNewWorkQueue() {
	ctx := context.Background()

	// Create a GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a workqueue with a limit of 100 items
	wq := gcs.NewWorkQueue(client.Bucket("my-workqueue-bucket"), 100)

	// Queue a task with priority
	if err := wq.Queue(ctx, "task-123", workqueue.Options{
		Priority: 10,
	}); err != nil {
		log.Print(err)
		return
	}

	fmt.Println("Task queued successfully")
}

// Example_processTasks demonstrates the typical workflow for processing tasks
// from the workqueue.
func Example_processTasks() {
	ctx := context.Background()

	// Create a GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a workqueue
	wq := gcs.NewWorkQueue(client.Bucket("my-workqueue-bucket"), 100)

	// Enumerate tasks
	inProgress, queued, deadLettered, err := wq.Enumerate(ctx)
	if err != nil {
		log.Print(err)
		return
	}

	fmt.Printf("In-progress: %d, Queued: %d, Dead-lettered: %d\n",
		len(inProgress), len(queued), len(deadLettered))

	// Process queued tasks
	for _, key := range queued {
		// Start processing the task
		owned, err := key.Start(ctx)
		if err != nil {
			log.Printf("Failed to start task %s: %v", key.Name(), err)
			continue
		}

		// Process the task using the owned context
		// The context is cancelled if the lease is lost
		select {
		case <-owned.Context().Done():
			log.Printf("Lost ownership of task %s", key.Name())
			continue
		default:
			// Do work here...
		}

		// Mark the task as complete
		if err := owned.Complete(ctx); err != nil {
			log.Printf("Failed to complete task %s: %v", key.Name(), err)
		}
	}
}

// Example_requeueTask demonstrates how to requeue a task for later processing.
func Example_requeueTask() {
	ctx := context.Background()

	// Create a GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a workqueue
	wq := gcs.NewWorkQueue(client.Bucket("my-workqueue-bucket"), 100)

	// Enumerate and get a queued task
	_, queued, _, err := wq.Enumerate(ctx)
	if err != nil {
		log.Print(err)
		return
	}

	if len(queued) == 0 {
		fmt.Println("No tasks to process")
		return
	}

	// Start the task
	owned, err := queued[0].Start(ctx)
	if err != nil {
		log.Print(err)
		return
	}

	// If processing fails, requeue the task
	if err := owned.Requeue(ctx); err != nil {
		log.Printf("Failed to requeue: %v", err)
	}

	fmt.Println("Task requeued for retry")
}

// Example_handleOrphanedTasks demonstrates detecting and handling orphaned tasks.
func Example_handleOrphanedTasks() {
	ctx := context.Background()

	// Create a GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a workqueue
	wq := gcs.NewWorkQueue(client.Bucket("my-workqueue-bucket"), 100)

	// Enumerate tasks
	inProgress, _, _, err := wq.Enumerate(ctx)
	if err != nil {
		log.Print(err)
		return
	}

	// Check for orphaned tasks (tasks with expired leases)
	for _, key := range inProgress {
		if key.IsOrphaned() {
			fmt.Printf("Found orphaned task: %s\n", key.Name())
			// Requeue the orphaned task
			if err := key.Requeue(ctx); err != nil {
				log.Printf("Failed to requeue orphaned task: %v", err)
			}
		}
	}
}

// Example_getTaskState demonstrates retrieving the state of a specific task.
func Example_getTaskState() {
	ctx := context.Background()

	// Create a GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Create a workqueue
	wq := gcs.NewWorkQueue(client.Bucket("my-workqueue-bucket"), 100)

	// Get the state of a specific task
	state, err := wq.Get(ctx, "task-123")
	if err != nil {
		log.Printf("Task not found: %v", err)
		return
	}

	fmt.Printf("Task %s: status=%v, attempts=%d, priority=%d\n",
		state.Key, state.Status, state.Attempts, state.Priority)
}
