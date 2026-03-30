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

// FromCloudEvent converts a CloudEvent into a Pub/Sub message with standard
// CE attributes mapped to message attributes and the event data as the body.
func FromCloudEvent(ctx context.Context, event cloudevents.Event) *pubsub.Message {
	attributes := map[string]string{
		"ce-id":          event.ID(),
		"ce-specversion": event.SpecVersion(),
		"ce-type":        event.Type(),
		"ce-source":      event.Source(),
		"ce-subject":     event.Subject(),
		"ce-time":        event.Time().UTC().Format(time.RFC3339),
		"content-type":   event.DataContentType(),
	}

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
	}

	return &pubsub.Message{
		Attributes: attributes,
		Data:       event.Data(),
	}
}
