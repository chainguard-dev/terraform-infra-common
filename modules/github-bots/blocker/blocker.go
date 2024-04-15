package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk"
	"github.com/google/go-github/v61/github"
)

// This bot is responsible for passing/failing a check run based on the presence of any "blocking/*" label.
//
// This check can be made required so that blockers block merging of PRs.

func New() sdk.Bot {
	name := "blocker"

	handler := sdk.PullRequestHandler(func(ctx context.Context, pre github.PullRequestEvent, pr *github.PullRequest) error {
		log := clog.FromContext(ctx)

		cli := sdk.NewGitHubClient(ctx, *pre.Repo.Owner.Login, *pre.Repo.Name, name)
		defer cli.Close(ctx)

		owner, repo := *pr.Base.Repo.Owner.Login, *pr.Base.Repo.Name

		hasBlockingLabel := slices.ContainsFunc(pr.Labels, func(l *github.Label) bool {
			return strings.HasPrefix(*l.Name, "blocking/")
		})

		conclusion := "success"
		if hasBlockingLabel {
			conclusion = "failure"
			log.Debug("PR %d has blocking label")
		}
		sha := *pr.Head.SHA

		// If there are no check runs for the current head SHA, create one with the conclusion.
		crs, _, err := cli.Client().Checks.ListCheckRunsForRef(ctx, owner, repo, sha, &github.ListCheckRunsOptions{CheckName: github.String(name)})
		if err != nil {
			return fmt.Errorf("listing check runs: %w", err)
		}
		if len(crs.CheckRuns) == 0 {
			log.Infof("Creating CheckRun for PR %d sha %s with conclusion %s", *pr.Number, sha, conclusion)
			if _, _, err := cli.Client().Checks.CreateCheckRun(ctx, owner, repo, github.CreateCheckRunOptions{
				Name:       name,
				HeadSHA:    sha,
				Status:     github.String("completed"),
				Conclusion: &conclusion,
			}); err != nil {
				return fmt.Errorf("creating check run: %w", err)
			}
			return nil
		}

		// No change, nothing else to do.
		if *crs.CheckRuns[0].Conclusion == conclusion {
			log.Debugf("CheckRun for PR %d sha %s already has conclusion %s", *pr.Number, sha, conclusion)
			return nil
		}

		// If there's already a check run, update its conclusion.
		log.Infof("Updating CheckRun for PR %d sha %s with conclusion %s", *pr.Number, sha, conclusion)
		if _, _, err = cli.Client().Checks.UpdateCheckRun(ctx, owner, repo, *crs.CheckRuns[0].ID, github.UpdateCheckRunOptions{
			Name:       name,
			Status:     github.String("completed"),
			Conclusion: &conclusion,
			Output: &github.CheckRunOutput{
				Title:   github.String("Blocking label check"),
				Summary: github.String("This PR has a blocking label"),
			},
		}); err != nil {
			return fmt.Errorf("updating check run %d: %w", *crs.CheckRuns[0].ID, err)
		}
		return nil
	})

	return sdk.NewBot(name,
		sdk.BotWithHandler(handler),
	)
}
