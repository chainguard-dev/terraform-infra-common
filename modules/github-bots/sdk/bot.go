package sdk

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/chainguard-dev/clog"
	_ "github.com/chainguard-dev/clog/gcp/init" // enable GCP logging
	"github.com/chainguard-dev/terraform-infra-common/modules/github-events/schemas"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	mce "github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-github/v71/github"
	"github.com/sethvargo/go-envconfig"
)

// Define a type for keys used in context to prevent key collisions.
type contextKey string

// Define constants for the keys to use with context.WithValue.
const (
	ContextKeyAttributes contextKey = "ce-attributes"
	ContextKeyType       contextKey = "ce-type"
	ContextKeySubject    contextKey = "ce-subject"
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

var env = envconfig.MustProcess(context.Background(), &struct {
	Port int `env:"PORT, default=8080"`
}{})

func Serve(b Bot) {
	ctx := context.Background()

	log := clog.FromContext(ctx)

	http.DefaultTransport = httpmetrics.Transport
	go httpmetrics.ServeMetrics()
	defer httpmetrics.SetupTracer(ctx)()
	httpmetrics.SetBuckets(map[string]string{
		"api.github.com": "github",
		"octo-sts.dev":   "octosts",
	})

	c, err := mce.NewClientHTTP(b.Name,
		cloudevents.WithPort(env.Port),
	)
	if err != nil {
		clog.Fatalf("failed to create event client, %v", err)
	}

	log.Infof("starting bot %s receiver on port %d", b.Name, env.Port)
	if err := c.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) error {
		clog.FromContext(ctx).With("event", event).Debugf("received event")

		defer func() {
			if err := recover(); err != nil {
				clog.Errorf("panic: %s", debug.Stack())
			}
		}()

		log.With("type", event.Type(),
			"subject", event.Subject(),
			"action", event.Extensions()["action"]).Debug("handling event")

		// dispatch event to n handlers
		if handler, ok := b.Handlers[EventType(event.Type())]; ok {
			// loop over all event headers and add them to the context so they can be used by the handlers
			for k, v := range event.Context.GetExtensions() {
				ctx = context.WithValue(ctx, contextKey(k), v)
			}

			// add existing event attributes to context so they can be used by the handlers
			ctx = context.WithValue(ctx, ContextKeyAttributes, event.Extensions())
			ctx = context.WithValue(ctx, ContextKeyType, event.Type())
			ctx = context.WithValue(ctx, ContextKeySubject, event.Subject())

			switch h := handler.(type) {
			case WorkflowRunArtifactHandler:
				log.Debug("handling workflow run artifact event")

				var wre schemas.Wrapper[github.WorkflowRunEvent]
				if err := event.DataAs(&wre); err != nil {
					log.Errorf("failed to unmarshal workflow run event: %v", err)
					return err
				}

				if err := h(ctx, wre.Body); err != nil {
					log.Errorf("failed to handle workflow run event: %v", err)
					return err
				}
				return nil

			case WorkflowRunHandler:
				log.Debug("handling workflow run event")

				var wre schemas.Wrapper[github.WorkflowRunEvent]
				if err := event.DataAs(&wre); err != nil {
					log.Errorf("failed to unmarshal workflow run event: %v", err)
					return err
				}

				if err := h(ctx, wre.Body); err != nil {
					log.Errorf("failed to handle workflow run event: %v", err)
					return err
				}
				return nil

			case WorkflowRunLogsHandler:
				log.Debug("handling workflow run logs event")

				var wre schemas.Wrapper[github.WorkflowRunEvent]
				if err := event.DataAs(&wre); err != nil {
					log.Errorf("failed to unmarshal workflow run with logs event: %v", err)
					return err
				}

				if err := h(ctx, wre.Body); err != nil {
					log.Errorf("failed to handle workflow run with logs event: %v", err)
					return err
				}
				return nil

			case PullRequestHandler:
				log.Debug("handling pull request event")

				var pre schemas.Wrapper[github.PullRequestEvent]
				if err := event.DataAs(&pre); err != nil {
					log.Errorf("failed to unmarshal pull request event: %v", err)
					return err
				}

				if err := h(ctx, pre.Body); err != nil {
					log.Errorf("failed to handle pull request event: %v", err)
					return err
				}
				return nil

			case IssuesHandler:
				log.Debug("handling issue event")

				var ie schemas.Wrapper[github.IssueEvent]
				if err := event.DataAs(&ie); err != nil {
					log.Errorf("failed to unmarshal issue event: %v", err)
					return err
				}

				if err := h(ctx, ie.Body); err != nil {
					log.Errorf("failed to handle issue event: %v", err)
					return err
				}
				return nil

			case IssueCommentHandler:
				log.Debug("handling issue comment event")

				var ice schemas.Wrapper[github.IssueCommentEvent]
				if err := event.DataAs(&ice); err != nil {
					log.Errorf("failed to unmarshal issue comment event: %v", err)
					return err
				}

				if err := h(ctx, ice.Body); err != nil {
					log.Errorf("failed to handle issue comment event: %v", err)
					return err
				}
				return nil

			case PushHandler:
				log.Debug("handling push event")

				var pe schemas.Wrapper[github.PushEvent]
				if err := event.DataAs(&pe); err != nil {
					log.Errorf("failed to unmarshal push event: %v", err)
					return err
				}

				if err := h(ctx, pe.Body); err != nil {
					log.Errorf("failed to handle push event: %v", err)
					return err
				}
				return nil

			case CheckRunHandler:
				log.Debug("handling check_run event")

				var pe schemas.Wrapper[github.CheckRunEvent]
				if err := event.DataAs(&pe); err != nil {
					log.Errorf("failed to unmarshal push event: %v", err)
					return err
				}

				if err := h(ctx, pe.Body); err != nil {
					log.Errorf("failed to handle check_run event: %v", err)
					return err
				}
				return nil

			case CheckSuiteHandler:
				log.Debug("handling check_suite event")

				var pe schemas.Wrapper[github.CheckSuiteEvent]
				if err := event.DataAs(&pe); err != nil {
					log.Errorf("failed to unmarshal check_suite event: %v", err)
					return err
				}

				if err := h(ctx, pe.Body); err != nil {
					log.Errorf("failed to handle check_suite event: %v", err)
					return err
				}
				return nil

			case ProjectsV2ItemHandler:
				log.Debug("handling projects_v2_item event")

				var pie schemas.Wrapper[ProjectsV2ItemEvent]
				if err := event.DataAs(&pie); err != nil {
					log.Errorf("failed to unmarshal projects_v2_item event: %v", err)
					return err
				}

				if err := h(ctx, pie.Body); err != nil {
					log.Errorf("failed to handle projects_v2_item event: %v", err)
					return err
				}
				return nil

			default:
				return fmt.Errorf("unknown handler type %T", handler)
			}
		}

		clog.FromContext(ctx).With("event", event).Debugf("ignoring event")
		return nil
	}); err != nil {
		clog.Fatalf("failed to start event receiver, %v", err)
	}
}

// AttributeFromContext retrieves an attribute by key from the context.
// Returns nil if the attribute does not exist.
func AttributeFromContext(ctx context.Context, key string) interface{} {
	return ctx.Value(contextKey(key))
}
