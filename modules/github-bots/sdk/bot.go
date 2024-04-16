package sdk

import (
	"context"
	"encoding/json"
	"fmt"
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

type Bot struct {
	Name     string
	Handlers map[EventType]EventHandlerFunc
}

type BotOptions func(*Bot)

func NewBot(name string, opts ...BotOptions) Bot {
	bot := Bot{
		Name:     name,
		Handlers: make(map[EventType]EventHandlerFunc),
	}

	for _, opt := range opts {
		opt(&bot)
	}

	return bot
}

func BotWithHandler(handler EventHandlerFunc) BotOptions {
	return func(b *Bot) {
		b.RegisterHandler(handler)
	}
}

func (b *Bot) RegisterHandler(handler EventHandlerFunc) {
	etype := handler.EventType()
	if _, ok := b.Handlers[etype]; ok {
		panic(fmt.Sprintf("handler for event type %s already registered", etype))
	}
	b.Handlers[etype] = handler
}

func Serve(b Bot) {
	var env struct {
		Port int `envconfig:"PORT" default:"8080" required:"true"`
	}
	if err := envconfig.Process("", &env); err != nil {
		clog.Fatalf("failed to process env var: %s", err)
	}
	ctx := context.Background()

	slog.SetDefault(slog.New(gcp.NewHandler(slog.LevelInfo)))

	logger := clog.FromContext(ctx)

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
			return httpmetrics.HandlerFunc(b.Name, func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		}),
	)
	if err != nil {
		clog.Fatalf("failed to create event client, %v", err)
	}

	logger.Infof("starting bot %s receiver on port %d", b.Name, env.Port)
	if err := c.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) error {
		clog.FromContext(ctx).With("event", event).Debugf("received event")

		defer func() {
			if err := recover(); err != nil {
				clog.Errorf("panic: %s", debug.Stack())
			}
		}()

		logger.Info("handling event", "type", event.Type())

		// dispatch event to n handlers
		if handler, ok := b.Handlers[EventType(event.Type())]; ok {
			switch h := handler.(type) {
			case WorkflowRunHandler:
				logger.Debug("handling workflow run event")

				var wre schemas.Wrapper[github.WorkflowRunEvent]
				if err := event.DataAs(&wre); err != nil {
					return err
				}

				wr := &github.WorkflowRun{}
				if err := marshalTo(wre.Body.WorkflowRun, wr); err != nil {
					return err
				}

				return h(ctx, wre.Body, wr)

			case PullRequestHandler:
				logger.Debug("handling pull request event")

				var pre schemas.Wrapper[github.PullRequestEvent]
				if err := event.DataAs(&pre); err != nil {
					return err
				}

				pr := &github.PullRequest{}
				if err := marshalTo(pre.Body.PullRequest, pr); err != nil {
					return err
				}

				return h(ctx, pre.Body, pr)
			}
		}

		clog.FromContext(ctx).With("event", event).Debugf("ignoring event")
		return nil
	}); err != nil {
		clog.Fatalf("failed to start event receiver, %v", err)
	}
}

func marshalTo(source any, target any) error {
	b, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, target)
}
