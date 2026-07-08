/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"testing"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// newTestClient returns a Pub/Sub client backed by an in-memory server.
func newTestClient(t *testing.T) (*pstest.Server, *pubsub.Client) {
	t.Helper()

	srv := pstest.NewServer()
	t.Cleanup(func() { srv.Close() })

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	client, err := pubsub.NewClient(t.Context(), "test-project", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("pubsub.NewClient: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	return srv, client
}

// newTestPublisher returns an ordering-enabled publisher backed by an
// in-memory Pub/Sub server. The server starts with auto-responses off so the
// test controls each publish outcome via AddPublishResponse.
func newTestPublisher(t *testing.T) (*pstest.Server, *pubsub.Publisher) {
	t.Helper()

	srv, client := newTestClient(t)

	const topicName = "projects/test-project/topics/events"
	if _, err := client.TopicAdminClient.CreateTopic(t.Context(), &pubsubpb.Topic{Name: topicName}); err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}

	topic := client.Publisher(topicName)
	topic.EnableMessageOrdering = true
	topic.PublishSettings.CountThreshold = 1
	t.Cleanup(topic.Stop)

	srv.SetAutoPublishResponse(false)

	return srv, topic
}

func baseEvent(t *testing.T) cloudevents.Event {
	t.Helper()

	event := cloudevents.NewEvent()
	event.SetID("id")
	event.SetSource("source")
	event.SetType("type")

	return event
}

func orderedEvent(t *testing.T, key string) cloudevents.Event {
	t.Helper()

	event := baseEvent(t)
	event.SetExtension("partitionkey", key)

	return event
}

// forwarded captures the parts of a published message that forward controls:
// the ordering key it derives from the event and the actor it stamps on.
type forwarded struct {
	OrderingKey string
	Actor       string
}

func published(srv *pstest.Server) []forwarded {
	msgs := srv.Messages()

	out := make([]forwarded, len(msgs))
	for i, m := range msgs {
		out[i] = forwarded{OrderingKey: m.OrderingKey, Actor: m.Attributes["ce-actor"]}
	}

	return out
}

// buildPublishers dedupes publishers by topic (an override pointing at the
// default reuses the default; two types sharing a topic share one publisher),
// and rejects an empty override topic name.
func TestBuildPublishers(t *testing.T) {
	_, client := newTestClient(t)

	t.Run("override equal to default reuses the default publisher", func(t *testing.T) {
		def, byType, all, err := buildPublishers(client, "shared", map[string]string{"t.default": "shared"})
		if err != nil {
			t.Fatalf("buildPublishers: %v", err)
		}
		if byType["t.default"] != def {
			t.Errorf("override pointing at the default topic: got a separate publisher, want the default")
		}
		if len(all) != 1 {
			t.Errorf("distinct publishers: got %d, want 1", len(all))
		}
	})

	t.Run("types sharing a topic share one publisher, distinct from default", func(t *testing.T) {
		def, byType, all, err := buildPublishers(client, "shared", map[string]string{"t.a": "ded", "t.b": "ded"})
		if err != nil {
			t.Fatalf("buildPublishers: %v", err)
		}
		if byType["t.a"] != byType["t.b"] {
			t.Errorf("types sharing a topic: got different publishers, want the same")
		}
		if byType["t.a"] == def {
			t.Errorf("dedicated publisher: got the default publisher, want a distinct one")
		}
		if len(all) != 2 {
			t.Errorf("distinct publishers (shared + ded): got %d, want 2", len(all))
		}
		// Every publisher must enable message ordering — the dedicated topics
		// carry partitionkey-derived ordering keys, and publishing an ordered
		// message without it fails at runtime.
		for _, p := range all {
			if !p.EnableMessageOrdering {
				t.Errorf("publisher %p: EnableMessageOrdering = false, want true", p)
			}
		}
	})

	t.Run("empty override topic name errors", func(t *testing.T) {
		if _, _, _, err := buildPublishers(client, "shared", map[string]string{"t.x": ""}); err == nil {
			t.Errorf("empty override topic: got nil error, want error")
		}
	})
}

// publisherForType routes an event to its type's dedicated publisher when one is
// configured and to the default publisher otherwise.
func TestPublisherForType(t *testing.T) {
	def := &pubsub.Publisher{}
	dedicated := &pubsub.Publisher{}

	byType := map[string]*pubsub.Publisher{"dev.chainguard.registry.pull.v1": dedicated}

	if got := publisherForType(byType, def, "dev.chainguard.registry.pull.v1"); got != dedicated {
		t.Errorf("routed type: got %p, want dedicated publisher %p", got, dedicated)
	}
	if got := publisherForType(byType, def, "dev.chainguard.registry.push.v1"); got != def {
		t.Errorf("unrouted type: got %p, want default publisher %p", got, def)
	}
	if got := publisherForType(map[string]*pubsub.Publisher{}, def, "anything"); got != def {
		t.Errorf("empty map: got %p, want default publisher %p", got, def)
	}
}

// A publish failure pauses the message's ordering key in the client, so every
// later message for that key fails until the key is resumed. forward must
// resume the key on failure: a second event sharing the key has to reach the
// topic once the transient error clears.
func TestForwardResumesOrderingKeyAfterFailure(t *testing.T) {
	srv, topic := newTestPublisher(t)

	srv.AddPublishResponse(&pubsubpb.PublishResponse{}, status.Error(codes.InvalidArgument, "boom"))

	event := orderedEvent(t, "key-1")

	if err := forward(t.Context(), topic, event, "actor@example.com"); err == nil {
		t.Fatal("forward: got nil, want error from injected publish failure")
	}

	srv.SetAutoPublishResponse(true)

	if err := forward(t.Context(), topic, event, "actor@example.com"); err != nil {
		t.Errorf("forward after resume: got %v, want nil", err)
	}

	want := []forwarded{{OrderingKey: "key-1", Actor: "actor@example.com"}}
	if diff := cmp.Diff(want, published(srv)); diff != "" {
		t.Errorf("published messages (-want +got):\n%s", diff)
	}
}

// A keyless event carries no ordering key, so the client never pauses publishing
// on its behalf and forward never resumes anything. A failed keyless publish
// must leave later keyless publishes free to reach the topic, which is what
// makes enabling message ordering on the publisher safe for unordered traffic.
func TestForwardKeylessNotPausedByFailure(t *testing.T) {
	srv, topic := newTestPublisher(t)

	srv.AddPublishResponse(&pubsubpb.PublishResponse{}, status.Error(codes.InvalidArgument, "boom"))

	event := baseEvent(t)

	if err := forward(t.Context(), topic, event, "actor@example.com"); err == nil {
		t.Fatal("forward: got nil, want error from injected publish failure")
	}

	srv.SetAutoPublishResponse(true)

	if err := forward(t.Context(), topic, event, "actor@example.com"); err != nil {
		t.Errorf("keyless forward after failure: got %v, want nil", err)
	}

	want := []forwarded{{OrderingKey: "", Actor: "actor@example.com"}}
	if diff := cmp.Diff(want, published(srv)); diff != "" {
		t.Errorf("published messages (-want +got):\n%s", diff)
	}
}

// forward stamps the authenticated caller as the ce-actor attribute. An event
// may already carry an actor extension, which FromCloudEventWithOrdering copies
// into ce-actor, so forward must overwrite it: a source cannot pass off another
// actor as its own.
func TestForwardOverwritesEventSuppliedActor(t *testing.T) {
	srv, topic := newTestPublisher(t)
	srv.SetAutoPublishResponse(true)

	event := baseEvent(t)
	event.SetExtension("actor", "attacker@evil.example")

	if err := forward(t.Context(), topic, event, "actor@example.com"); err != nil {
		t.Fatalf("forward: got %v, want nil", err)
	}

	want := []forwarded{{OrderingKey: "", Actor: "actor@example.com"}}
	if diff := cmp.Diff(want, published(srv)); diff != "" {
		t.Errorf("published messages (-want +got):\n%s", diff)
	}
}
