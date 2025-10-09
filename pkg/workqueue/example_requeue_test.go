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

	// Example 2: Requeue with a 5-minute delay for polling
	if req.Key == "polling" {
		fmt.Println("Polling, requeueing after 5 minutes")
		return &workqueue.ProcessResponse{
			RequeueAfterSeconds: 300, // 5 minutes
		}, nil
	}

	// Example 3: Retry with backoff due to rate limiting (error scenario)
	if req.Key == "rate-limited" {
		fmt.Println("Rate limited, retrying after 5 minutes")
		return &workqueue.ProcessResponse{
			RequeueAfterSeconds: 300, // 5 minutes
		}, nil
	}

	// Example 4: Requeue with exponential backoff based on external state
	if req.Key == "backoff" {
		retryCount := getRetryCount(req.Key) // hypothetical function
		delay := time.Duration(retryCount) * time.Minute
		fmt.Printf("Requeueing with %v delay\n", delay)
		return &workqueue.ProcessResponse{
			RequeueAfterSeconds: int64(delay.Seconds()),
		}, nil
	}

	// Example 5: Traditional error handling (uses default backoff)
	if req.Key == "error" {
		return nil, fmt.Errorf("processing failed")
	}

	// Example 6: Non-retriable error
	if req.Key == "permanent-failure" {
		return nil, workqueue.NonRetriableError(
			fmt.Errorf("unrecoverable error"),
			"Resource does not exist",
		)
	}

	return &workqueue.ProcessResponse{}, nil
}

// ExampleRequeueAfter demonstrates how to use RequeueAfter for polling in a callback.
func ExampleRequeueAfter() {
	callback := func(_ context.Context, _ string, _ workqueue.Options) error {
		// Do some work...

		// Request requeue with a 30-second delay for polling
		return workqueue.RequeueAfter(30 * time.Second)
	}
	// This would be used in a dispatcher
	err := callback(context.Background(), "example-key", workqueue.Options{})
	delay, ok, isError := workqueue.GetRequeueDelay(err)
	fmt.Printf("Requeue requested: %v, delay: %v, isError: %v\n", ok, delay, isError)
	// Output: Requeue requested: true, delay: 30s, isError: false
}

// ExampleRetryAfter demonstrates how to use RetryAfter for error retry in a callback.
func ExampleRetryAfter() {
	callback := func(_ context.Context, _ string, _ workqueue.Options) error {
		// Attempt some work that may fail...

		// On error, request retry after 1 minute
		return workqueue.RetryAfter(time.Minute)
	}
	// This would be used in a dispatcher
	err := callback(context.Background(), "example-key", workqueue.Options{})
	delay, ok, isError := workqueue.GetRequeueDelay(err)
	fmt.Printf("Retry requested: %v, delay: %v, isError: %v\n", ok, delay, isError)
	// Output: Retry requested: true, delay: 1m0s, isError: true
}

func getRetryCount(_ string) int {
	// This is a placeholder for demonstration
	return 1
}
