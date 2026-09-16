# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only guard on the go_* relabel rules rendered into the otel sidecar config.

mock_provider "google-beta" {}

mock_provider "google" {}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
  }
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels = []
  team                  = "fixture"
  containers = {
    "main" = {
      image = "cgr.dev/chainguard/static:latest"
      ports = [{ container_port = 8080 }]
    }
  }
}

run "default_drops_all_go_series" {
  command = plan

  assert {
    condition = (
      strcontains(one([for e in google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].env : e.value if e.name == "OTEL_CONFIG"]), "regex: '^go_.*'\n            action: drop") &&
      !strcontains(one([for e in google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].env : e.value if e.name == "OTEL_CONFIG"]), "__tmp_keep_go")
    )
    error_message = "default otel config must drop every go_* series with the plain drop rule and no keep rules"
  }
}

run "keep_list_renders_tag_drop_labeldrop_in_order" {
  command = plan

  variables {
    keep_go_metrics = ["go_memstats_sys_bytes", "go_goroutines"]
  }

  assert {
    condition = (
      strcontains(
        one([for e in google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].env : e.value if e.name == "OTEL_CONFIG"]),
        join("\n", [
          "          - source_labels: [ __name__ ]",
          "            regex: '^(go_memstats_sys_bytes|go_goroutines)$'",
          "            target_label: __tmp_keep_go",
          "            replacement: keep",
          "          - source_labels: [ __name__, __tmp_keep_go ]",
          "            regex: '^go_.*;$'",
          "            action: drop",
          "          - regex: '^__tmp_keep_go$'",
          "            action: labeldrop",
        ]),
      ) &&
      !strcontains(one([for e in google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].env : e.value if e.name == "OTEL_CONFIG"]), "regex: '^go_.*'\n            action: drop")
    )
    error_message = "otel config must render tag, drop, labeldrop in that order for the keep list, and not the plain go_* drop rule"
  }
}

run "keep_list_rejects_regex_metacharacters" {
  command = plan

  variables {
    keep_go_metrics = ["go_.*"]
  }

  expect_failures = [var.keep_go_metrics]
}

run "keep_list_rejects_non_go_names" {
  command = plan

  variables {
    keep_go_metrics = ["process_resident_memory_bytes"]
  }

  expect_failures = [var.keep_go_metrics]
}
