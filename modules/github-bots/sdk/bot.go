package sdk

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/clog/gcp"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/schemas"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-github/v61/github"
	"github.com/kelseyhightower/envconfig"
)

type Bot interface{ Name() string }

func Serve(b Bot) {
	var env struct {
		Port int `envconfig:"PORT" default:"8080" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		clog.Fatalf("failed to process env var: %s", err)
	}
	ctx := context.Background()

	slog.SetDefault(slog.New(gcp.NewHandler(slog.LevelInfo)))

	http.DefaultTransport = httpmetrics.Transport
	go httpmetrics.ServeMetrics()
	httpmetrics.SetupTracer(ctx)
	httpmetrics.SetBuckets(map[string]string{
		"api.github.com": "github",
		"octosts.dev":    "octosts",
	})

	c, err := cloudevents.NewClientHTTP(
		cloudevents.WithPort(env.Port),
		cloudevents.WithMiddleware(func(next http.Handler) http.Handler {
			return httpmetrics.HandlerFunc(b.Name(), func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}),
	)
	if err != nil {
		clog.Fatalf("failed to create event client, %v", err)
	}
	if err := c.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) error {
		clog.FromContext(ctx).With("event", event).Debugf("received event")

		defer func() {
			if err := recover(); err != nil {
				clog.Errorf("panic: %s", debug.Stack())
			}
		}()

		if prb, ok := b.(interface {
			OnPullRequest(ctx context.Context, pr *github.PullRequest) error
		}); ok && event.Type() == "dev.chainguard.github.pull_request" {
			log := clog.FromContext(ctx).With("bot", b.Name(), "event", event.Type())
			lctx := clog.WithLogger(ctx, log)

			// I don't love this. We decode the event into a wrapper, then marshal the body to json,
			// then unmarshal it into a github.PullRequest. There should be a better way to get
			// the .Body field of the original event.
			var pre schemas.Wrapper[schemas.PullRequestEvent]
			if err := event.DataAs(&pre); err != nil {
				clog.FromContext(lctx).Errorf("failed to unmarshal pull request: %v", err)
				return err
			}
			b, err := json.Marshal(pre.Body.PullRequest)
			if err != nil {
				clog.FromContext(lctx).Errorf("failed to marshal pull request: %v", err)
			}
			var pr github.PullRequest
			if err := json.Unmarshal(b, &pr); err != nil {
				clog.FromContext(lctx).Errorf("failed to unmarshal pull request: %v", err)
				return err
			}

			if err := prb.OnPullRequest(ctx, &pr); err != nil {
				clog.FromContext(lctx).Errorf("failed to handle pull request: %v", err)
				return err
			}
			return nil
		}

		if prb, ok := b.(interface {
			OnWorkflowRunEvent(ctx context.Context, wfr *github.WorkflowRunEvent) error
		}); ok && event.Type() == "dev.chainguard.github.workflow_run" {
			log := clog.FromContext(ctx).With("bot", b.Name(), "event", event.Type())
			lctx := clog.WithLogger(ctx, log)

			// I don't love this. We decode the event into a wrapper, then marshal the body to json,
			// then unmarshal it into a github.PullRequest. There should be a better way to get
			// the .Body field of the original event.
			var wfr schemas.Wrapper[schemas.WorkflowRunEvent]
			if err := event.DataAs(&wfr); err != nil {
				clog.FromContext(lctx).Errorf("failed to unmarshal workflow run event: %v", err)
				return err
			}
			b, err := json.Marshal(wfr.Body)
			if err != nil {
				clog.FromContext(lctx).Errorf("failed to marshal workflow run event: %v", err)
			}

			var wr github.WorkflowRunEvent
			if err := json.Unmarshal(b, &wr); err != nil {
				clog.FromContext(lctx).Errorf("failed to unmarshal workflow run event: %v", err)
				return err
			}

			if err := prb.OnWorkflowRunEvent(ctx, &wr); err != nil {
				clog.FromContext(lctx).Errorf("failed to handle workflow run event: %v", err)
				return err
			}
			return nil
		}

		clog.FromContext(ctx).With("event", event).Debugf("ignoring event type %s", event.Type())
		return nil
	}); err != nil {
		clog.Fatalf("failed to start event receiver, %v", err)
	}
}
