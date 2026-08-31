# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only proof of the alerting knobs' for_each gating, which terraform
# validate cannot evaluate: the defaults must keep producing the full policy
# set for every existing consumer (behavior-preserving), and the opt-outs
# (per_region_alerts = false, rolling_periods = ["07"]) must produce exactly
# one policy. The google provider is mocked so the plan stays fully offline:
# no credentials, no state, no API calls.

mock_provider "google" {}

variables {
  project_id            = "fixture-project"
  service_name          = "fixture-svc"
  regions               = ["us-central1", "us-west1", "us-east1"]
  notification_channels = []
}

# Defaults: every rolling period alerts, multi-region and per-region — the
# pre-knob policy set (2 multi-region + 3 regions x 2 periods = 6 per-region).
run "defaults_preserve_full_policy_set" {
  command = plan

  variables {
    slo = {
      enable          = true
      enable_alerting = true
    }
  }

  assert {
    condition     = toset(output.alert_policy_keys.multi_region) == toset(["07", "30"])
    error_message = "defaults must create multi-region policies for both rolling periods"
  }
  assert {
    condition     = length(output.alert_policy_keys.per_region) == 6
    error_message = "defaults must create per-region policies for every region x period pair"
  }
  assert {
    condition     = length(output.alert_policy_keys.gclb) == 0
    error_message = "gclb policies must not exist without monitor_gclb"
  }
}

# The lens shape: single policy per incident.
run "opt_outs_reduce_to_one_policy" {
  command = plan

  variables {
    slo = {
      enable          = true
      enable_alerting = true
      alerting = {
        per_region_alerts = false
        rolling_periods   = ["07"]
      }
    }
  }

  assert {
    condition     = toset(output.alert_policy_keys.multi_region) == toset(["07"])
    error_message = "rolling_periods = [\"07\"] must gate the multi-region policy to the 07d period"
  }
  assert {
    condition     = length(output.alert_policy_keys.per_region) == 0
    error_message = "per_region_alerts = false must create no per-region policies"
  }
}

# SLOs themselves are never gated by the alerting knobs — only alert policies.
run "slos_survive_alerting_opt_outs" {
  command = plan

  variables {
    slo = {
      enable          = true
      enable_alerting = true
      alerting = {
        per_region_alerts = false
        rolling_periods   = ["07"]
      }
    }
  }

  assert {
    condition     = length(google_monitoring_slo.success_cr) == 2 && length(google_monitoring_slo.success_cr_per_region) == 6
    error_message = "alerting opt-outs must not remove any SLO"
  }
}

# An empty rolling_periods would silently disable all alerting while
# enable_alerting = true; the variable validation must reject it at plan time.
run "empty_rolling_periods_rejected" {
  command = plan

  variables {
    slo = {
      enable          = true
      enable_alerting = true
      alerting = {
        rolling_periods = []
      }
    }
  }

  expect_failures = [var.slo]
}
