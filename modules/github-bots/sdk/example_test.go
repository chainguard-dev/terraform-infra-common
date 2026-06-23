/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package sdk_test

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
	"github.com/google/go-github/v88/github"
)

func ExampleNewBot() {
	bot := sdk.NewBot("my-bot",
		sdk.BotWithHandler(
			sdk.PullRequestHandler(func(_ context.Context, pre github.PullRequestEvent) error {
				fmt.Printf("handling PR #%d\n", pre.GetNumber())
				return nil
			}),
		),
	)
	fmt.Println(bot.Name)
	// Output: my-bot
}

func ExampleBot_RegisterHandler() {
	bot := sdk.NewBot("my-bot")
	bot.RegisterHandler(
		sdk.PushHandler(func(_ context.Context, _ github.PushEvent) error {
			return nil
		}),
	)
	fmt.Println(len(bot.Handlers))
	// Output: 1
}

func ExampleAttributeFromContext() {
	ctx := context.Background()
	val := sdk.AttributeFromContext(ctx, "missing-key")
	fmt.Println(val)
	// Output: <nil>
}
