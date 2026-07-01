/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package pubsub

import (
	"context"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/chainguard-dev/clog"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/types"
)

// standardCEAttributes is the set of attribute keys populated from standard
// CloudEvent fields. Extensions with these names are skipped to prevent
// accidental overwrites.
var standardCEAttributes = map[string]struct{}{
	"ce-id":          {},
	"ce-specversion": {},
	"ce-type":        {},
	"ce-source":      {},
	"ce-subject":     {},
	"ce-time":        {},
	"content-type":   {},
}

// partitionKeyExtension is the CloudEvents Partitioning extension key.
// See https://github.com/cloudevents/spec/blob/main/cloudevents/extensions/partitioning.md.
const partitionKeyExtension = "partitionkey"

// FromCloudEvent converts a CloudEvent into a Pub/Sub message. Standard
// CloudEvent fields become message attributes and the event data becomes
// the body. The message carries no OrderingKey; use
// [FromCloudEventWithOrdering] to derive one from the event.
func FromCloudEvent(ctx context.Context, event cloudevents.Event) *pubsub.Message {
	return fromCloudEvent(ctx, event, false)
}

// FromCloudEventWithOrdering converts a CloudEvent into a Pub/Sub message
// like [FromCloudEvent], and additionally sets the message OrderingKey from
// the event's CloudEvents Partitioning extension (partitionkey), per the
// CloudEvents Pub/Sub protocol binding.
//
// Publishing a message with an OrderingKey requires EnableMessageOrdering on
// the publisher; without it, Publish rejects the message outright. An event
// without a partitionkey produces a message with no OrderingKey, which
// publishes fine either way.
func FromCloudEventWithOrdering(ctx context.Context, event cloudevents.Event) *pubsub.Message {
	return fromCloudEvent(ctx, event, true)
}

func fromCloudEvent(ctx context.Context, event cloudevents.Event, ordered bool) *pubsub.Message {
	attributes := map[string]string{
		"ce-id":          event.ID(),
		"ce-specversion": event.SpecVersion(),
		"ce-type":        event.Type(),
		"ce-source":      event.Source(),
		"ce-subject":     event.Subject(),
		"ce-time":        event.Time().UTC().Format(time.RFC3339),
		"content-type":   event.DataContentType(),
	}

	var orderingKey string
	for k, v := range event.Extensions() {
		key := "ce-" + k
		if _, reserved := standardCEAttributes[key]; reserved {
			clog.WarnContextf(ctx, "skipping extension %q: conflicts with standard CE attribute", k)
			continue
		}
		sv, err := types.ToString(v)
		if err != nil {
			clog.WarnContextf(ctx, "skipping non-string extension %q: %v", k, err)
			continue
		}
		attributes[key] = sv
		if ordered && k == partitionKeyExtension {
			orderingKey = sv
		}
	}

	return &pubsub.Message{
		Attributes:  attributes,
		Data:        event.Data(),
		OrderingKey: orderingKey,
	}
}
