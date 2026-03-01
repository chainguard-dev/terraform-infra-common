/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package conformance provides a comprehensive test suite for validating
// workqueue.Interface implementations.
//
// # Overview
//
// This package contains conformance tests that verify workqueue implementations
// adhere to the expected semantics defined by the workqueue.Interface contract.
// The tests cover queue ordering, deduplication, priority handling, backoff
// behavior, concurrency limits, and durability guarantees.
//
// # Features
//
// The conformance test suite validates:
//
//   - Queue ordering: FIFO ordering for items with equal priority
//   - Deduplication: Duplicate keys are merged, not duplicated
//   - Priority handling: Higher priority items are processed first
//   - NotBefore scheduling: Items are not processed before their scheduled time
//   - Backoff behavior: Requeued items use exponential backoff with configurable limits
//   - Concurrency limits: The queue respects configured concurrency limits
//   - State transitions: Keys move correctly between queued, in-progress, and completed states
//   - Context lifecycle: Contexts are properly managed and canceled
//   - Retry limits: Failed items can be dead-lettered after max retries
//   - Durability: Persistent implementations retain state across restarts (optional)
//
// # Usage
//
// To test a workqueue implementation, call TestSemantics with a constructor
// function that creates instances of your workqueue:
//
//	func TestMyWorkQueue(t *testing.T) {
//	    conformance.TestSemantics(t, func(concurrency int) workqueue.Interface {
//	        return myworkqueue.New(concurrency)
//	    })
//	}
//
// For implementations with durability guarantees, also run TestDurability:
//
//	func TestMyDurableWorkQueue(t *testing.T) {
//	    conformance.TestDurability(t, func(concurrency int) workqueue.Interface {
//	        return myworkqueue.NewDurable(concurrency)
//	    })
//	}
//
// For implementations that support concurrent dispatch, run TestConcurrency:
//
//	func TestMyWorkQueueConcurrency(t *testing.T) {
//	    conformance.TestConcurrency(t, func(concurrency int) workqueue.Interface {
//	        return myworkqueue.New(concurrency)
//	    })
//	}
//
// For implementations that support retry limits and dead-lettering, run TestMaxRetry:
//
//	func TestMyWorkQueueMaxRetry(t *testing.T) {
//	    conformance.TestMaxRetry(t, func(concurrency int) workqueue.Interface {
//	        return myworkqueue.New(concurrency)
//	    })
//	}
//
// # Integration Patterns
//
// The conformance tests are designed to work with any workqueue.Interface
// implementation. The constructor function pattern allows tests to create
// fresh instances for each scenario, ensuring test isolation.
//
// Tests automatically clean up by draining the queue after each scenario,
// making them safe to run against durable implementations that persist state.
//
// The test suite adjusts timing parameters (workqueue.BackoffPeriod and
// workqueue.MaximumBackoffPeriod) to ensure tests complete in a reasonable
// time while still validating backoff behavior.
//
// # Test Scenarios
//
// TestSemantics includes scenarios for:
//
//   - Simple queue ordering
//   - Queue more than concurrency limit
//   - Simple deduplication
//   - Priority ordering
//   - Start and complete with context check
//   - Start and requeue
//   - Start and queue
//   - Start queue and requeue with priority
//   - Simple not before
//   - Queue not before with priorities
//   - Requeue doesn't reset not before
//   - Requeuing a priority task has backoff
//   - Get key states
//   - Queued and in-progress
//
// TestConcurrency validates that the workqueue respects concurrency limits
// when processing items concurrently with a dispatcher.
//
// TestDurability validates that queued items persist across workqueue
// instance restarts.
//
// TestMaxRetry validates that items can be dead-lettered after exceeding
// retry limits and that attempt counts are tracked correctly.
package conformance
