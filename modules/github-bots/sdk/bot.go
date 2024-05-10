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
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-github/v61/github"
	"github.com/kelseyhightower/envconfig"
)

// Define a type for keys used in context to prevent key collisions.
type contextKey string

// Define constants for the keys to use with context.WithValue.
const (
	ContextKeyAttributes contextKey = "ce-attributes"
	ContextKeyType       contextKey = "ce-type"
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
	defer httpmetrics.SetupTracer(ctx)()
	httpmetrics.SetBuckets(map[string]string{
		"api.github.com": "github",
		"octo-sts.dev":   "octosts",
	})

	c, err := mce.NewClientHTTP(
		cloudevents.WithPort(env.Port),
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
			// loop over all event headers and add them to the context so they can be used by the handlers
			for k, v := range event.Context.GetExtensions() {
				ctx = context.WithValue(ctx, contextKey(k), v)
			}

			// add existing event attributes to context so they can be used by the handlers
			ctx = context.WithValue(ctx, ContextKeyAttributes, event.Extensions())
			ctx = context.WithValue(ctx, ContextKeyType, event.Type())

			switch h := handler.(type) {
			case WorkflowRunArtifactHandler:
				logger.Debug("handling workflow run artifact event")

				var wre schemas.Wrapper[github.WorkflowRunEvent]
				if err := event.DataAs(&wre); err != nil {
					logger.Errorf("failed to unmarshal workflow run event: %v", err)
					return err
				}

				if err := h(ctx, wre.Body); err != nil {
					logger.Errorf("failed to handle workflow run event: %v", err)
					return err
				}
				return nil

			case WorkflowRunHandler:
				logger.Debug("handling workflow run event")

				var wre schemas.Wrapper[github.WorkflowRunEvent]
				if err := event.DataAs(&wre); err != nil {
					logger.Errorf("failed to unmarshal workflow run event: %v", err)
					return err
				}

				if err := h(ctx, wre.Body); err != nil {
					logger.Errorf("failed to handle workflow run event: %v", err)
					return err
				}
				return nil

			case PullRequestHandler:
				logger.Debug("handling pull request event")

				var pre schemas.Wrapper[github.PullRequestEvent]
				if err := event.DataAs(&pre); err != nil {
					logger.Errorf("failed to unmarshal pull request event: %v", err)
					return err
				}

				if err := h(ctx, pre.Body); err != nil {
					logger.Errorf("failed to handle pull request event: %v", err)
					return err
				}
				return nil

			case IssueCommentHandler:
				logger.Debug("handling issue comment event")

				var ice schemas.Wrapper[github.IssueCommentEvent]
				if err := event.DataAs(&ice); err != nil {
					logger.Errorf("failed to unmarshal issue comment event: %v", err)
					return err
				}

				if err := h(ctx, ice.Body); err != nil {
					logger.Errorf("failed to handle issue comment event: %v", err)
					return err
				}
				return nil
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

// AttributeFromContext retrieves an attribute by key from the context.
// Returns nil if the attribute does not exist.
func AttributeFromContext(ctx context.Context, key string) interface{} {
	return ctx.Value(contextKey(key))
}
