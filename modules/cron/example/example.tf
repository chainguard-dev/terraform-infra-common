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

  schedule = "*/8 * * * *"

  env = {
    EXAMPLE_ENV = "honk"
  }
}
