# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only guard on the go_*/process_* relabel rules rendered into the otel sidecar config.

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

run "default_renders_the_inlined_rules_unchanged" {
  command = plan

  assert {
    condition = (
      strcontains(
        one([for e in google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].env : e.value if e.name == "OTEL_CONFIG"]),
        join("\n", [
          "        metric_relabel_configs:",
          "          - source_labels: [ __name__ ]",
          "            regex: '^prometheus_.*'",
          "            action: drop",
          "          # Drop process_* except process_start_time_seconds, which the",
          "          # metricstarttime processor below reads and then drops itself.",
          "          # Relabel regexes are RE2 (no lookahead), so mark the one to keep",
          "          # with a scratch label, drop the unmarked rest, then drop the label.",
          "          # The scratch label uses the reserved __tmp prefix so it can never",
          "          # collide with a label an application exports.",
          "          - source_labels: [ __name__ ]",
          "            regex: 'process_start_time_seconds'",
          "            target_label: __tmp_keep_process_start_time",
          "            replacement: 'true'",
          "          - source_labels: [ __name__, __tmp_keep_process_start_time ]",
          "            regex: 'process_.*;'",
          "            action: drop",
          "          - regex: __tmp_keep_process_start_time",
          "            action: labeldrop",
          "          - source_labels: [ __name__ ]",
          "            regex: '^go_.*'",
          "            action: drop",
          "",
          "processors:",
        ]),
      ) &&
      !strcontains(one([for e in google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].env : e.value if e.name == "OTEL_CONFIG"]), "__tmp_keep_go")
    )
    error_message = "with no keep list the otel config must render exactly the relabel rules the template inlined before, so opted-out services see no OTEL_CONFIG change"
  }
}

run "keep_list_is_added_to_process_start_time" {
  command = plan

  variables {
    keep_go_metrics = ["go_memstats_sys_bytes", "process_resident_memory_bytes"]
  }

  assert {
    condition = (
      strcontains(
        one([for e in google_cloud_run_v2_service.this["us-central1"].template[0].containers[1].env : e.value if e.name == "OTEL_CONFIG"]),
        join("\n", [
          "        metric_relabel_configs:",
          "          - source_labels: [ __name__ ]",
          "            regex: '^prometheus_.*'",
          "            action: drop",
          "          - source_labels: [ __name__ ]",
          "            regex: '^(process_start_time_seconds|go_memstats_sys_bytes|process_resident_memory_bytes)$'",
          "            target_label: __tmp_keep_go",
          "            replacement: keep",
          "          - source_labels: [ __name__, __tmp_keep_go ]",
          "            regex: '^(go|process)_.*;$'",
          "            action: drop",
          "          - regex: '^__tmp_keep_go$'",
          "            action: labeldrop",
        ]),
      )
    )
    error_message = "otel config must render the keep list after process_start_time_seconds in the tag rule, then the drop and labeldrop rules"
  }
}

run "keep_list_rejects_regex_metacharacters" {
  command = plan

  variables {
    keep_go_metrics = ["go_.*"]
  }

  expect_failures = [var.keep_go_metrics]
}

run "keep_list_rejects_other_prefixes" {
  command = plan

  variables {
    keep_go_metrics = ["prometheus_http_requests_total"]
  }

  expect_failures = [var.keep_go_metrics]
}
