# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Calls regional-go-cron with a tag value created in the same apply, so the
# value's ID is unknown at plan time.

terraform {
  required_version = ">= 1.3"

  required_providers {
    google      = { source = "hashicorp/google", version = ">= 8.7.0" }
    google-beta = { source = "hashicorp/google-beta", version = ">= 8.7.0" }
    ko          = { source = "ko-build/ko", version = ">= 0.0.21" }
    cosign      = { source = "chainguard-dev/cosign", version = ">= 0.5.5" }
  }
}

resource "google_tags_tag_key" "this" {
  parent     = "projects/fixture-project"
  short_name = "fixture"
}

resource "google_tags_tag_value" "this" {
  parent     = google_tags_tag_key.this.id
  short_name = "denied"
}

module "cron" {
  source = "../../"

  project_id      = "fixture-project"
  name            = "fixture"
  service_account = "fixture@fixture-project.iam.gserviceaccount.com"
  team            = "fixture"
  regions = {
    us-central1 = {}
  }
  regional-cronspec = {
    us-central1 = { schedule = "0 * * * *" }
  }
  resource_manager_tag_values = {
    "execution-overrides" = google_tags_tag_value.this.id
  }
}

output "binding_parents" {
  value = module.cron.tag_binding_parents
}
