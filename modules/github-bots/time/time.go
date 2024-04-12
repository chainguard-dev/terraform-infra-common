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

	bot := sdk.NewBot(name)
	bot.RegisterHandler(timeHandler(name))

	return bot
}

func timeHandler(name string) sdk.PullRequestHandler {
	return func(ctx context.Context, client sdk.GitHubClient, pr *github.PullRequest) error {
		return client.SetComment(ctx, pr, name, fmt.Sprintf("The time is now %s", time.Now().Format(time.RFC3339)))
	}
}
