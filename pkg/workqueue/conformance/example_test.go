/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package conformance_test

import (
	"testing"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/conformance"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/inmem"
)

// ExampleTestSemantics demonstrates how to use the conformance test suite
// to validate a workqueue implementation's semantic behavior.
func ExampleTestSemantics() {
	// Create a test instance (in real usage, this would be *testing.T from a test function)
	var t *testing.T

	// Define a constructor function that creates instances of your workqueue
	ctor := inmem.NewWorkQueue

	// Run the conformance test suite
	conformance.TestSemantics(t, ctor)
}

// ExampleTestConcurrency demonstrates how to test that a workqueue
// implementation correctly handles concurrent processing.
func ExampleTestConcurrency() {
	// Create a test instance (in real usage, this would be *testing.T from a test function)
	var t *testing.T

	// Define a constructor function that creates instances of your workqueue
	ctor := inmem.NewWorkQueue

	// Run the concurrency test
	conformance.TestConcurrency(t, ctor)
}

// ExampleTestDurability demonstrates how to test that a durable workqueue
// implementation persists state across restarts.
func ExampleTestDurability() {
	// Create a test instance (in real usage, this would be *testing.T from a test function)
	var t *testing.T

	// Define a constructor function that creates instances of your durable workqueue
	// Note: This example uses inmem which is NOT durable and will fail this test.
	// Use a durable implementation like GCS-backed workqueue instead.
	// In practice, use a durable implementation:
	// ctor := func(concurrency int) workqueue.Interface {
	//     return gcs.New(ctx, bucket, prefix, concurrency)
	// }
	ctor := inmem.NewWorkQueue

	// Run the durability test
	conformance.TestDurability(t, ctor)
}

// ExampleTestMaxRetry demonstrates how to test that a workqueue
// implementation correctly handles retry limits and dead-lettering.
func ExampleTestMaxRetry() {
	// Create a test instance (in real usage, this would be *testing.T from a test function)
	var t *testing.T

	// Define a constructor function that creates instances of your workqueue
	ctor := inmem.NewWorkQueue

	// Run the max retry test
	conformance.TestMaxRetry(t, ctor)
}

// Example_fullTestSuite demonstrates how to use all conformance tests
// together in a complete test suite for a workqueue implementation.
func Example_fullTestSuite() {
	// In a real test file, you would have multiple test functions:
	//
	// func TestMyWorkQueueSemantics(t *testing.T) {
	//     conformance.TestSemantics(t, func(concurrency int) workqueue.Interface {
	//         return myworkqueue.New(concurrency)
	//     })
	// }
	//
	// func TestMyWorkQueueConcurrency(t *testing.T) {
	//     conformance.TestConcurrency(t, func(concurrency int) workqueue.Interface {
	//         return myworkqueue.New(concurrency)
	//     })
	// }
	//
	// func TestMyWorkQueueDurability(t *testing.T) {
	//     conformance.TestDurability(t, func(concurrency int) workqueue.Interface {
	//         return myworkqueue.NewDurable(concurrency)
	//     })
	// }
	//
	// func TestMyWorkQueueMaxRetry(t *testing.T) {
	//     conformance.TestMaxRetry(t, func(concurrency int) workqueue.Interface {
	//         return myworkqueue.New(concurrency)
	//     })
	// }
}
