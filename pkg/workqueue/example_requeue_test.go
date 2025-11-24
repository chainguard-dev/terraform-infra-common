/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package workqueue_test

import (
	"context"
	"fmt"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

// ExampleWorker demonstrates how to implement a worker that can request
// custom requeue delays.
type ExampleWorker struct {
	workqueue.UnimplementedWorkqueueServiceServer
}

func (w *ExampleWorker) Process(_ context.Context, req *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
	// Example 1: Process successfully
	if req.Key == "success" {
		fmt.Println("Processing successful")
		return &workqueue.ProcessResponse{}, nil
	}

	// Example 2: Requeue with a 5-minute delay
	if req.Key == "rate-limited" {
		fmt.Println("Rate limited, requeueing after 5 minutes")
		return &workqueue.ProcessResponse{
			RequeueAfterSeconds: 300, // 5 minutes
		}, nil
	}

	// Example 3: Requeue with exponential backoff based on external state
	if req.Key == "backoff" {
		retryCount := getRetryCount(req.Key) // hypothetical function
		delay := time.Duration(retryCount) * time.Minute
		fmt.Printf("Requeueing with %v delay\n", delay)
		return &workqueue.ProcessResponse{
			RequeueAfterSeconds: int64(delay.Seconds()),
		}, nil
	}

	// Example 4: Traditional error handling (uses default backoff)
	if req.Key == "error" {
		return nil, fmt.Errorf("processing failed")
	}

	// Example 5: Non-retriable error
	if req.Key == "permanent-failure" {
		return nil, workqueue.NonRetriableError(
			fmt.Errorf("unrecoverable error"),
			"Resource does not exist",
		)
	}

	return &workqueue.ProcessResponse{}, nil
}

// ExampleRequeueAfter demonstrates how to use RequeueAfter in a callback.
func ExampleRequeueAfter() {
	callback := func(_ context.Context, _ string, _ workqueue.Options) error {
		// Do some work...

		// Request requeue with a 30-second delay
		return workqueue.RequeueAfter(30 * time.Second)
	}
	// This would be used in a dispatcher
	err := callback(context.Background(), "example-key", workqueue.Options{})
	delay, ok := workqueue.GetRequeueDelay(err)
	fmt.Printf("Requeue requested: %v, delay: %v\n", ok, delay)
	// Output: Requeue requested: true, delay: 30s
}

func getRetryCount(_ string) int {
	// This is a placeholder for demonstration
	return 1
}
