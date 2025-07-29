/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package pubsub

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/types"
)

func FromCloudEvent(_ context.Context, event cloudevents.Event) *pubsub.Message {
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
		sv, err := types.ToString(v)
		if err != nil {
			log.Printf("encountered non-string extension %q: %v", k, err)
			continue
		}
		attributes["ce-"+k] = sv
	}

	return &pubsub.Message{
		Attributes: attributes,
		Data:       event.Data(),
	}
}
