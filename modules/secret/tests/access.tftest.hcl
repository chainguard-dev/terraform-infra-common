# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Who may manage or read the secret, and which of their actions the
# abnormal-access alert ignores. A mocked plan accepts any filter text and any
# IAM member, so these runs pin the rendered filter and reject at plan the
# inputs that Cloud Monitoring or IAM would reject only at apply.

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

variables {
  project_id            = "fixture-project"
  name                  = "fixture"
  authorized-adder      = "serviceAccount:adder@fixture-project.iam.gserviceaccount.com"
  service-account       = "reader@fixture-project.iam.gserviceaccount.com"
  notification-channels = []
  team                  = "fixture"
}

run "defaults_grant_no_version_managers" {
  command = plan

  assert {
    condition     = length(google_secret_manager_secret_iam_binding.authorize-version-manager) == 0
    error_message = "no version_managers must create no version-manager bindings"
  }
  assert {
    condition     = local.payload_reader_emails == tolist(["reader@fixture-project.iam.gserviceaccount.com"])
    error_message = "the service account must be the only exempt payload reader"
  }
  assert {
    condition     = local.version_management_emails == tolist(["adder@fixture-project.iam.gserviceaccount.com"])
    error_message = "the authorized adder must be the only exempt version manager"
  }
}

run "exemptions_match_whole_emails" {
  command = plan

  # The previous filter used principalEmail=~"a|b", an unanchored regex, so an
  # exemption for adder@... also covered evil-adder@... and dots matched any
  # character. Every exemption is now an exact comparison.
  assert {
    condition     = !strcontains(google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter, "principalEmail=~")
    error_message = "principal exemptions must not use regex matching"
  }
  assert {
    condition = can(regex(
      "principalEmail=\\(\"reader@fixture-project\\.iam\\.gserviceaccount\\.com\"\\)\\s+protoPayload\\.methodName=\"google\\.cloud\\.secretmanager\\.v1\\.SecretManagerService\\.AccessSecretVersion\"",
      google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter,
    ))
    error_message = "the payload-read exemption must compare the reader's exact email"
  }
  assert {
    condition = can(regex(
      "principalEmail=\\(\"adder@fixture-project\\.iam\\.gserviceaccount\\.com\"\\)\\s+protoPayload\\.methodName=\\(",
      google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter,
    ))
    error_message = "the version-management exemption must compare the adder's exact email"
  }
}

run "version_management_exemption_covers_every_version_method" {
  command = plan

  assert {
    condition = alltrue([
      for method in ["AddSecretVersion", "EnableSecretVersion", "DisableSecretVersion", "DestroySecretVersion"] :
      strcontains(
        regex("principalEmail=\\([^)]*\\)\\s+protoPayload\\.methodName=\\(([^)]*)\\)", google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter)[0],
        "\"google.cloud.secretmanager.v1.SecretManagerService.${method}\"",
      )
    ])
    error_message = "version managers must be exempt for add, enable, disable, and destroy"
  }
  assert {
    condition = !strcontains(
      regex("principalEmail=\\([^)]*\\)\\s+protoPayload\\.methodName=\\(([^)]*)\\)", google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter)[0],
      "AccessSecretVersion",
    )
    error_message = "version managers must never be exempt for payload reads"
  }
}

run "version_managers_get_manager_and_viewer_roles" {
  command = plan

  variables {
    version_managers = [
      "group:managers@fixture.dev",
      "user:alice@fixture.dev",
      "serviceAccount:rotator@fixture-project.iam.gserviceaccount.com",
    ]
  }

  assert {
    condition     = toset(keys(google_secret_manager_secret_iam_binding.authorize-version-manager)) == toset(["roles/secretmanager.secretVersionManager", "roles/secretmanager.viewer"])
    error_message = "version_managers must get exactly the version-manager and viewer roles"
  }
  assert {
    condition = alltrue([
      for b in google_secret_manager_secret_iam_binding.authorize-version-manager :
      b.members == toset(["group:managers@fixture.dev", "user:alice@fixture.dev", "serviceAccount:rotator@fixture-project.iam.gserviceaccount.com"])
    ])
    error_message = "each version-manager binding must hold exactly the version_managers"
  }
  assert {
    condition     = google_secret_manager_secret_iam_binding.authorize-version-adder.members == toset(["serviceAccount:adder@fixture-project.iam.gserviceaccount.com"])
    error_message = "version_managers must not change the version-adder binding"
  }
  assert {
    condition     = google_secret_manager_secret_iam_binding.authorize-service-access.members == toset(["serviceAccount:reader@fixture-project.iam.gserviceaccount.com"])
    error_message = "version_managers must never gain payload access"
  }
  assert {
    condition = local.version_management_emails == tolist([
      "adder@fixture-project.iam.gserviceaccount.com",
      "alice@fixture.dev",
      "rotator@fixture-project.iam.gserviceaccount.com",
    ])
    error_message = "user and service-account managers must be exempt by email; a group's own address never acts"
  }
  assert {
    condition     = !strcontains(google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter, "managers@fixture.dev")
    error_message = "a group address can never match an audit log principal, so it must not take filter space"
  }
}

run "group_members_are_exempt_for_version_management_only" {
  command = plan

  variables {
    version_managers = ["group:managers@fixture.dev"]
    group_members = {
      "managers@fixture.dev" = ["bob@fixture.dev", "carol@fixture.dev"]
    }
  }

  assert {
    condition = local.version_management_emails == tolist([
      "adder@fixture-project.iam.gserviceaccount.com",
      "bob@fixture.dev",
      "carol@fixture.dev",
    ])
    error_message = "each group member must be exempt for version management"
  }
  assert {
    condition     = local.payload_reader_emails == tolist(["reader@fixture-project.iam.gserviceaccount.com"])
    error_message = "group members must never be exempt for payload reads"
  }
  assert {
    condition = strcontains(
      regex("principalEmail=\\(([^)]*)\\)\\s+protoPayload\\.methodName=\\(", google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter)[0],
      "\"bob@fixture.dev\" OR \"carol@fixture.dev\"",
    )
    error_message = "the rendered version-management exemption must list each member as an exact, quoted email"
  }
  assert {
    condition = !strcontains(
      regex("principalEmail=\\(([^)]*)\\)\\s+protoPayload\\.methodName=\"google\\.cloud\\.secretmanager\\.v1\\.SecretManagerService\\.AccessSecretVersion\"", google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter)[0],
      "bob@fixture.dev",
    )
    error_message = "the rendered payload-read exemption must not list group members"
  }
}

run "group_members_work_for_a_group_adder" {
  command = plan

  # Secrets created before version_managers name their group as the adder.
  variables {
    authorized-adder = "group:adders@fixture.dev"
    group_members = {
      "adders@fixture.dev" = ["dave@fixture.dev"]
    }
  }

  assert {
    condition     = local.version_management_emails == tolist(["dave@fixture.dev"])
    error_message = "a group adder's members must be exempt, and the group's own address must not be"
  }
  assert {
    condition     = length(google_secret_manager_secret_iam_binding.authorize-version-manager) == 0
    error_message = "group_members must grant no IAM"
  }
}

run "shared_members_render_once" {
  command = plan

  variables {
    authorized-adders = ["group:adders@fixture.dev"]
    version_managers  = ["group:managers@fixture.dev", "user:erin@fixture.dev"]
    group_members = {
      "adders@fixture.dev"   = ["erin@fixture.dev", "frank@fixture.dev"]
      "managers@fixture.dev" = ["frank@fixture.dev"]
    }
  }

  assert {
    condition = local.version_management_emails == tolist([
      "adder@fixture-project.iam.gserviceaccount.com",
      "erin@fixture.dev",
      "frank@fixture.dev",
    ])
    error_message = "an identity in several groups, or also named directly, must be exempt once"
  }
}

run "an_empty_group_adds_no_exemption" {
  command = plan

  variables {
    version_managers = ["group:managers@fixture.dev"]
    group_members    = { "managers@fixture.dev" = [] }
  }

  assert {
    condition     = local.version_management_emails == tolist(["adder@fixture-project.iam.gserviceaccount.com"])
    error_message = "an empty group must add no exemption"
  }
}

run "no_resolvable_manager_omits_the_clause" {
  command = plan

  # A group-only adder with no listed members leaves nothing to compare, and
  # principalEmail=() is a syntax error Cloud Monitoring reports only at apply.
  variables {
    authorized-adder = "group:adders@fixture.dev"
  }

  assert {
    condition     = length(local.version_management_emails) == 0
    error_message = "a group-only adder without members must exempt no one"
  }
  assert {
    condition     = !strcontains(google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter, "principalEmail=()")
    error_message = "the filter must never compare against an empty list"
  }
  assert {
    condition     = !strcontains(google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter, "DisableSecretVersion")
    error_message = "with no one to exempt, the version-management clause must be absent"
  }
}

run "applies_with_version_managers_and_group_members" {
  command = apply

  variables {
    version_managers = ["group:managers@fixture.dev"]
    group_members    = { "managers@fixture.dev" = ["bob@fixture.dev"] }
  }

  assert {
    condition     = google_secret_manager_secret_iam_binding.authorize-version-manager["roles/secretmanager.secretVersionManager"].role == "roles/secretmanager.secretVersionManager"
    error_message = "the version-manager binding must survive a mocked apply"
  }
  assert {
    condition     = output.authorized_secret_id == google_secret_manager_secret.this.id
    error_message = "the authorized secret ID must identify the secret, not the IAM binding"
  }
  assert {
    condition     = google_secret_manager_secret_iam_binding.authorize-service-access.members == toset(["serviceAccount:reader@fixture-project.iam.gserviceaccount.com"])
    error_message = "the authorized secret output must preserve the configured payload readers"
  }
}

run "version_manager_domain_is_rejected" {
  command = plan

  variables {
    version_managers = ["domain:fixture.dev"]
  }

  expect_failures = [var.version_managers]
}

run "version_manager_principal_set_is_rejected" {
  command = plan

  variables {
    version_managers = ["principalSet://iam.googleapis.com/projects/1/locations/global/workloadIdentityPools/p/*"]
  }

  expect_failures = [var.version_managers]
}

run "version_manager_without_prefix_is_rejected" {
  command = plan

  variables {
    version_managers = ["alice@fixture.dev"]
  }

  expect_failures = [var.version_managers]
}

run "version_manager_workforce_principal_is_rejected" {
  command = plan

  # It ends in an email, but it is not one the audit log would name.
  variables {
    version_managers = ["principal://iam.googleapis.com/locations/global/workforcePools/pool/subject/alice@fixture.dev"]
  }

  expect_failures = [var.version_managers]
}

run "mixed_case_identities_are_exempt_once" {
  command = plan

  # IAM and the filter's = both ignore case, so case is accepted and only
  # folded to keep duplicates out of the filter.
  variables {
    version_managers = ["group:Managers@fixture.dev", "user:Alice@fixture.dev"]
    group_members = {
      "managers@fixture.dev" = ["Bob@fixture.dev", "bob@fixture.dev", "alice@fixture.dev"]
    }
  }

  assert {
    condition = local.version_management_emails == tolist([
      "adder@fixture-project.iam.gserviceaccount.com",
      "alice@fixture.dev",
      "bob@fixture.dev",
    ])
    error_message = "identities that differ only in case must be exempt once"
  }
}

run "group_members_for_an_unknown_group_are_rejected" {
  command = plan

  variables {
    version_managers = ["group:managers@fixture.dev"]
    group_members    = { "managres@fixture.dev" = ["bob@fixture.dev"] }
  }

  expect_failures = [var.group_members]
}

run "group_members_for_a_user_manager_are_rejected" {
  command = plan

  variables {
    version_managers = ["user:alice@fixture.dev"]
    group_members    = { "alice@fixture.dev" = ["bob@fixture.dev"] }
  }

  expect_failures = [var.group_members]
}

run "group_member_with_prefix_is_rejected" {
  command = plan

  variables {
    version_managers = ["group:managers@fixture.dev"]
    group_members    = { "managers@fixture.dev" = ["user:bob@fixture.dev"] }
  }

  expect_failures = [var.group_members]
}

run "group_member_that_would_break_the_filter_is_rejected" {
  command = plan

  variables {
    version_managers = ["group:managers@fixture.dev"]
    group_members    = { "managers@fixture.dev" = ["bob@fixture.dev\") OR (\"x"] }
  }

  expect_failures = [var.group_members]
}

run "filter_over_the_monitoring_limit_is_rejected_at_plan" {
  command = plan

  variables {
    version_managers = ["group:managers@fixture.dev"]
    group_members = {
      "managers@fixture.dev" = [for i in range(1000) : format("member-%04d@fixture.dev", i)]
    }
  }

  expect_failures = [google_monitoring_alert_policy.anomalous-secret-access]
}

run "shared_monitoring_defaults_off" {
  command = plan
  assert {
    condition     = length(google_logging_metric.anomalous_secret_access) == 0 && output.access_log_metric_name == null
    error_message = "Existing secret callers must not acquire a new log metric."
  }
}

run "shared_monitoring_preserves_access_exemptions" {
  command = plan
  variables {
    access_log_metric_name = "fixture-secret-access"
    service-account        = "runtime@fixture-project.iam.gserviceaccount.com"
    authorized-adder       = "serviceAccount:rotation@fixture-project.iam.gserviceaccount.com"
  }
  assert {
    condition = (
      google_logging_metric.anomalous_secret_access[0].filter == google_monitoring_alert_policy.anomalous-secret-access.conditions[0].condition_matched_log[0].filter &&
      strcontains(google_logging_metric.anomalous_secret_access[0].filter, "runtime@fixture-project.iam.gserviceaccount.com") &&
      strcontains(google_logging_metric.anomalous_secret_access[0].filter, "rotation@fixture-project.iam.gserviceaccount.com")
    )
    error_message = "Shared monitoring must preserve the local alert's runtime and rotation identity exemptions."
  }
  assert {
    condition = (
      google_logging_metric.anomalous_secret_access[0].project == "fixture-project" &&
      output.access_log_metric_name == "fixture-secret-access" &&
      google_logging_metric.anomalous_secret_access[0].metric_descriptor[0].metric_kind == "DELTA" &&
      google_logging_metric.anomalous_secret_access[0].metric_descriptor[0].value_type == "INT64"
    )
    error_message = "The source project must own the exported event counter."
  }
}
