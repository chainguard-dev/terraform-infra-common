package main

import (
	"context"
	"fmt"
	"time"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
	"github.com/google/go-github/v61/github"
)

func New() sdk.Bot {
	name := "time"

	handler := sdk.PullRequestHandler(func(ctx context.Context, pre github.PullRequestEvent) error {
		cli := sdk.NewGitHubClient(ctx, *pre.Repo.Owner.Login, *pre.Repo.Name, name)
		defer cli.Close(ctx)

		return cli.SetComment(ctx, pre.PullRequest, name, fmt.Sprintf("The time is now %s", time.Now().Format(time.RFC3339)))
	})

	return sdk.NewBot(name,
		sdk.BotWithHandler(handler),
	)
}
