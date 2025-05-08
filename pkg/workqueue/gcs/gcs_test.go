/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue/conformance"
)

func TestWorkQueue(t *testing.T) {
	bucket, ok := os.LookupEnv("WORKQUEUE_GCS_TEST_BUCKET")
	if !ok {
		t.Skip("WORKQUEUE_GCS_TEST_BUCKET not set")
	}
	// Adjust this to a suitable period for testing things.
	// The conformance tests own adjusting MaximumBackoffPeriod.
	workqueue.BackoffPeriod = 10 * time.Second

	client, err := storage.NewClient(context.Background())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	conformance.TestSemantics(t, func(u int) workqueue.Interface {
		return NewWorkQueue(client.Bucket(bucket), u)
	})

	conformance.TestConcurrency(t, func(u int) workqueue.Interface {
		return NewWorkQueue(client.Bucket(bucket), u)
	})

	conformance.TestDurability(t, func(u int) workqueue.Interface {
		return NewWorkQueue(client.Bucket(bucket), u)
	})

	conformance.TestMaxRetry(t, func(u int) workqueue.Interface {
		return NewWorkQueue(client.Bucket(bucket), u)
	})
}
