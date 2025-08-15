provider "google" {
  project = var.project_id
}

variable "project_id" {
  type        = string
  description = "The project that will host the cron job."
}

resource "google_service_account" "this" {
  project    = var.project_id
  account_id = "cron-example"
}

module "cron" {
  source = "../"

  project_id      = var.project_id
  name            = "example"
  service_account = google_service_account.this.email

  importpath  = "github.com/chainguard-dev/terraform-infra-common/modules/cron/example"
  working_dir = path.module

  schedule = "*/8 * * * *" # Every 8 minutes

  env = {
    EXAMPLE_ENV = "honk"
  }

  # Alert configuration example:
  # Check for successful executions in a 30-minute window
  # But only alert after 15 minutes of absence (faster detection)
  # This helps detect issues quickly while checking a broader context window
  success_alert_alignment_period_seconds = 1800 # 30 minutes
  success_alert_duration_seconds         = 900  # 15 minutes

  # Note: Set notification_channels to actual channel IDs to receive alerts
  notification_channels = []

  # Required squad label for the job
  squad = "example-team"
}
