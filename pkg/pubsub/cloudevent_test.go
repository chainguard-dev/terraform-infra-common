/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package pubsub

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestFromCloudEvent(t *testing.T) {
	now := time.Unix(123456789, 0)
	tests := []struct {
		name string
		in   cloudevents.Event
		out  *pubsub.Message
	}{{
		name: "simple empty payload",
		in: func() cloudevents.Event {
			event := cloudevents.NewEvent()
			event.SetID("id")
			event.SetSource("source")
			event.SetType("type")
			event.SetSubject("subject")
			event.SetTime(now)

			event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{})
			return event
		}(),
		out: &pubsub.Message{
			Attributes: map[string]string{
				"ce-id":          "id",
				"ce-source":      "source",
				"ce-specversion": "1.0",
				"ce-type":        "type",
				"ce-subject":     "subject",
				"ce-time":        "1973-11-29T21:33:09Z",
				"content-type":   "application/json",
			},
			Data: []byte("{}"),
		},
	}, {
		name: "non-empty payload",
		in: func() cloudevents.Event {
			event := cloudevents.NewEvent()
			event.SetID("id")
			event.SetSource("source")
			event.SetType("another-type")
			event.SetSubject("subject")
			event.SetTime(now)

			event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
				"foo": "bar",
				"baz": 3,
			})
			return event
		}(),
		out: &pubsub.Message{
			Attributes: map[string]string{
				"ce-id":          "id",
				"ce-source":      "source",
				"ce-specversion": "1.0",
				"ce-type":        "another-type",
				"ce-subject":     "subject",
				"ce-time":        "1973-11-29T21:33:09Z",
				"content-type":   "application/json",
			},
			Data: []byte(`{"baz":3,"foo":"bar"}`),
		},
	}, {
		name: "with extensions",
		in: func() cloudevents.Event {
			event := cloudevents.NewEvent()
			event.SetID("id")
			event.SetSource("source")
			event.SetType("another-type")
			event.SetSubject("subject")
			event.SetTime(now)

			event.SetData(cloudevents.ApplicationJSON, map[string]interface{}{})

			event.SetExtension("ext1", "value1")
			event.SetExtension("ext2", "value2")
			return event
		}(),
		out: &pubsub.Message{
			Attributes: map[string]string{
				"ce-id":          "id",
				"ce-source":      "source",
				"ce-specversion": "1.0",
				"ce-type":        "another-type",
				"ce-subject":     "subject",
				"ce-time":        "1973-11-29T21:33:09Z",
				"ce-ext1":        "value1",
				"ce-ext2":        "value2",
				"content-type":   "application/json",
			},
			Data: []byte("{}"),
		},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := FromCloudEvent(context.Background(), test.in)
			if diff := cmp.Diff(out, test.out, cmpopts.IgnoreUnexported(pubsub.Message{})); diff != "" {
				t.Errorf("(-got, +want): %s", diff)
			}
		})
	}
}
