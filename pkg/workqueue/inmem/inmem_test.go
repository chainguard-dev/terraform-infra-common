/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package inmem

import (
	"testing"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/conformance"
)

func TestWorkQueue(t *testing.T) {
	// Adjust this to a suitable period for testing things.
	// The conformance tests own adjusting MaximumBackoffPeriod.
	workqueue.BackoffPeriod = 1 * time.Second

	conformance.TestSemantics(t, NewWorkQueue)

	conformance.TestConcurrency(t, NewWorkQueue)

	conformance.TestMaxRetry(t, NewWorkQueue)
}
