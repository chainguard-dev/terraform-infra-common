# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Stateful mock applies exercise Terraform's replacement lifecycle without
# creating cloud resources. Do not override computed IDs: a new mock ID is
# how these assertions observe whether an attachment was actually replaced.
mock_provider "google" {}

variables {
  project                  = "fixture-project"
  region                   = "us-central1"
  name                     = "fixture"
  network                  = "projects/fixture-project/global/networks/fixture"
  subnetwork               = "projects/fixture-project/regions/us-central1/subnetworks/frontend"
  proxy_only_subnet        = "projects/fixture-project/regions/us-central1/subnetworks/proxy"
  psc_nat_subnets          = ["projects/fixture-project/regions/us-central1/subnetworks/psc-nat"]
  cloud_run_service_name   = "fixture"
  consumer_accept_projects = ["consumer-project"]
  labels                   = { team = "team", product = "after" }
}

run "initial" {
  command = apply

  variables {
    labels = { team = "team", product = "before" }
  }
}

# A label-only update previously recreated the attachment and permanently
# closed its consumer connections, despite leaving the forwarding rule intact.
run "label_update_preserves_attachment" {
  command = apply

  assert {
    condition     = google_compute_forwarding_rule.this.labels["product"] == "after"
    error_message = "The forwarding rule must receive the updated label."
  }

  assert {
    condition     = output.service_attachment_id == run.initial.service_attachment_id
    error_message = "Updating forwarding-rule labels must preserve the service attachment."
  }
}

# Explicit replacement exercises the lifecycle even though the mock provider
# does not model Google's immutable fields. The name stays unchanged, as it
# does when allow_global_access or the frontend protocol forces replacement.
run "forwarding_rule_replacement_replaces_attachment" {
  command = apply

  plan_options {
    replace = [google_compute_forwarding_rule.this]
  }

  assert {
    condition     = output.service_attachment_id != run.label_update_preserves_attachment.service_attachment_id
    error_message = "Replacing the forwarding rule must also replace the attachment that references it."
  }
}
