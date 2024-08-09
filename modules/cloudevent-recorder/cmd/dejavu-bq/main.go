package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/chainguard-dev/clog"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/api/iterator"
)

const (
	retryDelay = 10 * time.Millisecond
	maxRetry   = 3
)

var env = envconfig.MustProcess(context.Background(), &struct {
	Host string `envconfig:"HOST" default:"http://0.0.0.0" required:"true"`
	Port int    `envconfig:"PORT" default:"8080" required:"true"`

	EventType   string `envconfig:"EVENT_TYPE" default:"dev.chainguard.not_specified.not_specified" required:"true"`
	EventSource string `envconfig:"EVENT_SOURCE" default:"github.com" required:"true"`

	// Project is the GCP project where the dataset lives
	Project string `envconfig:"PROJECT" required:"true"`

	// QueryWindow is the window to look for release failures
	Query string `envconfig:"QUERY" required:"true"`
}{})

func Publish(ctx context.Context, event cloudevents.Event) error {
	// TODO: Add idtoken back?
	ceclient, err := cloudevents.NewClientHTTP(
		cloudevents.WithTarget(fmt.Sprintf("%s:%d", env.Host, env.Port)),
		cehttp.WithClient(http.Client{}))
	if err != nil {
		return fmt.Errorf("failed to create cloudevents client: %w", err)
	}

	rctx := cloudevents.ContextWithRetriesExponentialBackoff(context.WithoutCancel(ctx), retryDelay, maxRetry)
	ceresult := ceclient.Send(rctx, event)
	if cloudevents.IsUndelivered(ceresult) || cloudevents.IsNACK(ceresult) {
		return fmt.Errorf("failed to deliver event: %w", ceresult)
	}

	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	log := clog.FromContext(ctx)

	// TODO: Cache older queries, presumably a use case is replaying the same
	// data over and over so we can cache it so we avoid hitting bq every time

	// BigQuery client
	client, err := bigquery.NewClient(ctx, env.Project)
	if err != nil {
		log.Errorf("failed to create bigquery client: %v", err)
		return
	}
	defer client.Close()

	q := client.Query(env.Query)
	it, err := q.Read(ctx)
	if err != nil {
		log.Error(env.Query)
		log.Errorf("failed to run thresholdQuery, %v", err)
		return
	}

	// Iterate through each row in the returned query and handle every module
	// that is above the failure threshold.
	for {
		var row map[string]bigquery.Value
		err = it.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			log.Errorf("failed to read row, %v", err)
		}

		log.Infof("%+v\n", row)

		body, err := json.Marshal(row)
		if err != nil {
			log.Errorf("marshaling row: %v", err)
			return
		}
		log.Info(string(body))

		// TODO: Extract event type intelligently
		log = log.With("event-type", env.EventType)
		log.Debugf("forwarding event: %s", env.EventType)

		event := cloudevents.NewEvent()
		event.SetType(env.EventType)
		event.SetSource(env.EventSource)
		if err := event.SetData(cloudevents.ApplicationJSON, body); err != nil {
			log.Errorf("failed to set data: %v", err)
			return
		}

		// TODO: Time based publishing if we care about replaying using the
		// timestamps in the dataset
		if err := Publish(ctx, event); err != nil {
			log.Errorf("Publishing event: %v", err)
			// Try to process next events
		}
	}
}
