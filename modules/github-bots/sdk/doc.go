/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Package sdk provides a framework for building GitHub bots that receive and
// handle GitHub webhook events delivered as CloudEvents.
//
// # Bots
//
// A bot is created with [NewBot] and configured with handler functions for
// specific GitHub event types. Use [BotWithHandler] to register handlers, or
// call [Bot.RegisterHandler] directly.
//
// # Handlers
//
// Each handler type corresponds to a GitHub event type:
//   - [PullRequestHandler] — pull request events
//   - [WorkflowRunHandler] — workflow run events
//   - [IssueCommentHandler] — issue comment events
//   - [PushHandler] — push events
//   - [CheckRunHandler] — check run events
//   - [CheckSuiteHandler] — check suite events
//
// # Serving
//
// Call [Serve] to start the bot's CloudEvents HTTP receiver. The port defaults
// to the PORT environment variable, or 8080 if unset. Use [WithPort] to
// override the port programmatically.
//
// # GitHub Clients
//
// [NewGitHubClient] creates an authenticated GitHub API client using OctoSTS
// for token management. [NewInstallationClient] creates a client using a
// GitHub App installation transport.
package sdk
