# Example: GitHub Pull Request Processor using CloudEvents

# This example shows how to process GitHub pull request events
# using the cloudevents-workqueue module.

module "github_pr_processor" {
  source = "../"

  project_id            = var.project_id
  name                  = "github-pr-processor"
  regions               = var.regions
  team                  = var.team
  notification_channels = var.notification_channels

  # Subscribe to events from the broker
  broker = var.broker

  # Subscribe to all PR-related CloudEvent types
  filters = [
    { "type" = "dev.chainguard.github.pull_request" },
    { "type" = "dev.chainguard.github.pull_request_review" },
    { "type" = "dev.chainguard.github.pull_request_review_comment" },
    { "type" = "dev.chainguard.github.issue_comment" }, # For comments on PRs
    { "type" = "dev.chainguard.github.check_run" },     # For CI status updates
    { "type" = "dev.chainguard.github.check_suite" },
  ]

  # Use the pullrequesturl extension as the workqueue key
  # This will enqueue work items with the PR URL (e.g., https://github.com/owner/repo/pull/123)
  extension_key = "pullrequesturl"

  # Send work items to this workqueue
  workqueue = {
    name = var.workqueue_dispatcher_name
  }
}

# Example for processing GitHub issues
module "github_issue_processor" {
  source = "../"

  project_id            = var.project_id
  name                  = "github-issue-processor"
  regions               = var.regions
  team                  = var.team
  notification_channels = var.notification_channels

  broker = var.broker

  # Subscribe to issue-related CloudEvent types
  filters = [
    { "type" = "dev.chainguard.github.issues" },
    { "type" = "dev.chainguard.github.issue_comment" }, # Only for comments on issues, not PRs
  ]

  # Use the issueurl extension as the workqueue key
  # This will enqueue work items with the issue URL (e.g., https://github.com/owner/repo/issues/456)
  extension_key = "issueurl"

  workqueue = {
    name = var.workqueue_dispatcher_name
  }
}

# Example: Process only merged PRs
module "github_merged_pr_processor" {
  source = "../"

  project_id            = var.project_id
  name                  = "github-merged-pr-processor"
  regions               = var.regions
  team                  = var.team
  notification_channels = var.notification_channels

  broker = var.broker

  # Advanced filter: Only process closed PRs that were merged
  filters = [
    {
      "type"   = "dev.chainguard.github.pull_request"
      "action" = "closed"
      "merged" = "true"
    }
  ]

  extension_key = "pullrequesturl"

  workqueue = {
    name = var.workqueue_dispatcher_name
  }
}

# Example: Process events from specific repositories
module "github_specific_repo_processor" {
  source = "../"

  project_id            = var.project_id
  name                  = "github-myrepo-processor"
  regions               = var.regions
  team                  = var.team
  notification_channels = var.notification_channels

  broker = var.broker

  # Filter events from specific repositories
  # Note: Each filter must match ALL attributes, so we need separate filters
  # for each event type from the specific repo
  filters = [
    {
      "type"    = "dev.chainguard.github.pull_request"
      "subject" = "myorg/myrepo"
    },
    {
      "type"    = "dev.chainguard.github.issues"
      "subject" = "myorg/myrepo"
    },
    {
      "type"    = "dev.chainguard.github.pull_request_review"
      "subject" = "myorg/myrepo"
    }
  ]

  # This example could use either extension depending on the events
  extension_key = "pullrequesturl"

  workqueue = {
    name = var.workqueue_dispatcher_name
  }
}

# Example: Process GitHub CI/CD events (check_run and check_suite)
# These can now be handled in a single module instance
module "github_ci_processor" {
  source = "../"

  project_id            = var.project_id
  name                  = "github-ci-processor"
  regions               = var.regions
  team                  = var.team
  notification_channels = var.notification_channels

  broker = var.broker

  # Process both check_run and check_suite events
  filters = [
    { "type" = "dev.chainguard.github.check_run" },
    { "type" = "dev.chainguard.github.check_suite" }
  ]

  extension_key = "pullrequesturl"

  workqueue = {
    name = var.workqueue_dispatcher_name
  }
}

# Example variables
variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "team" {
  description = "Team label to apply to resources"
  type        = string
}

variable "regions" {
  description = "Regions to deploy in"
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "notification_channels" {
  description = "Notification channels for alerts"
  type        = list(string)
}

variable "broker" {
  description = "A map from each of the input region names to the name of the Broker topic in that region"
  type        = map(string)
}

variable "workqueue_dispatcher_name" {
  description = "Name of the workqueue dispatcher service"
  type        = string
}
