/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package pubsub_test

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/chainguard-dev/terraform-infra-common/pkg/pubsub"
)

func ExampleFromCloudEvent() {
	ctx := context.Background()
	event := cloudevents.NewEvent()
	event.SetID("example-id")
	event.SetSource("example/source")
	event.SetType("example.type")

	msg := pubsub.FromCloudEvent(ctx, event)
	_ = msg
}
