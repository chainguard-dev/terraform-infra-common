# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Plan-only checks of the impersonation binding's members. The baseline_* runs
# leave event_aware and also_pull_request_target unset and pin today's member
# for every selector, so the new inputs cannot change an existing binding.
# The member strings must match the attribute values that the
# github-wif-provider module maps:
#   - exactevent and exactanyworkflowevent carry the bare ref only for push,
#     workflow_dispatch and schedule, so a branch binding matches only those.
#   - prtevent and prtanyworkflowevent are "true" only for a
#     pull_request_target run against main.

mock_provider "google" {}

variables {
  project_id            = "fixture-project"
  name                  = "fixture-ci"
  wif-pool              = "projects/123/locations/global/workloadIdentityPools/fixture"
  repository            = "chainguard-dev/fixture"
  refspec               = "refs/heads/main"
  workflow_ref          = ".github/workflows/provision.yaml"
  notification_channels = []
}

run "baseline_branch" {
  command = plan

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exact/chainguard-dev/fixture|refs/heads/main|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "a branch binding must stay on attribute.exact"
  }
}

run "baseline_branch_any_workflow" {
  command = plan

  variables {
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyworkflow/chainguard-dev/fixture|refs/heads/main",
    ])
    error_message = "a wildcard branch binding must stay on attribute.exactanyworkflow"
  }
}

run "baseline_any_ref" {
  command = plan

  variables {
    refspec                = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyref/chainguard-dev/fixture|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "an any-ref binding must stay on attribute.exactanyref"
  }
}

run "baseline_any_ref_any_workflow" {
  command = plan

  variables {
    refspec                = "*"
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyrefanyworkflow/chainguard-dev/fixture",
    ])
    error_message = "an any-ref, any-workflow binding must stay on attribute.exactanyrefanyworkflow"
  }
}

run "baseline_pull_request" {
  command = plan

  variables {
    refspec = "pull_request"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequest/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "a pull_request binding must stay on attribute.pullrequest"
  }
}

run "baseline_pull_request_any_workflow" {
  command = plan

  variables {
    refspec                = "pull_request"
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequestanyworkflow/chainguard-dev/fixture|true",
    ])
    error_message = "a wildcard pull_request binding must stay on attribute.pullrequestanyworkflow"
  }
}

run "baseline_pull_request_target" {
  command = plan

  variables {
    refspec = "pull_request_target"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequesttarget/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "a pull_request_target binding must stay on attribute.pullrequesttarget"
  }
}

run "baseline_pull_request_target_any_workflow" {
  command = plan

  variables {
    refspec                = "pull_request_target"
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequesttargetanyworkflow/chainguard-dev/fixture|true",
    ])
    error_message = "a wildcard pull_request_target binding must stay on attribute.pullrequesttargetanyworkflow"
  }
}

run "baseline_version_tags" {
  command = plan

  variables {
    refspec = "version_tags"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.versiontags/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "a version_tags binding must stay on attribute.versiontags"
  }
}

run "baseline_version_tags_any_workflow" {
  command = plan

  variables {
    refspec                = "version_tags"
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.versiontagsanyworkflow/chainguard-dev/fixture|true",
    ])
    error_message = "a wildcard version_tags binding must stay on attribute.versiontagsanyworkflow"
  }
}

run "baseline_additional_federations" {
  command = plan

  variables {
    workflow_ref = "*"
    additional_federations = [
      { refspec = "refs/heads/main", workflow_ref = ".github/workflows/provision-stages.yaml" },
      { refspec = "pull_request", workflow_ref = ".github/workflows/presubmit.yaml" },
      { refspec = "pull_request_target", workflow_ref = ".github/workflows/tf-verify.yaml" },
      { refspec = "version_tags", workflow_ref = ".github/workflows/release.yaml" },
    ]
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyworkflow/chainguard-dev/fixture|refs/heads/main",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exact/chainguard-dev/fixture|refs/heads/main|chainguard-dev/fixture/.github/workflows/provision-stages.yaml",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequest/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/presubmit.yaml",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequesttarget/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/tf-verify.yaml",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.versiontags/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/release.yaml",
    ])
    error_message = "additional_federations must keep their current attributes"
  }
}

run "baseline_also_pull_request_target_matches_today" {
  command = plan

  variables {
    also_pull_request_target = true
  }

  # Today's pullrequesttarget value is "true" exactly when the ref is
  # refs/heads/main, so this member accepts the same tokens as the exact one.
  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exact/chainguard-dev/fixture|refs/heads/main|chainguard-dev/fixture/.github/workflows/provision.yaml",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequesttarget/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "also_pull_request_target must add a pullrequesttarget member for workflow_ref"
  }
}

run "event_aware_branch" {
  command = plan

  variables {
    event_aware = true
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactevent/chainguard-dev/fixture|refs/heads/main|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "event_aware must move a branch binding onto attribute.exactevent"
  }
}

run "event_aware_branch_any_workflow" {
  command = plan

  variables {
    event_aware            = true
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyworkflowevent/chainguard-dev/fixture|refs/heads/main",
    ])
    error_message = "event_aware must move a wildcard branch binding onto attribute.exactanyworkflowevent"
  }
}

run "event_aware_pull_request_target" {
  command = plan

  variables {
    event_aware  = true
    refspec      = "pull_request_target"
    workflow_ref = ".github/workflows/tf-verify.yaml"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.prtevent/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/tf-verify.yaml",
    ])
    error_message = "event_aware must move a pull_request_target binding onto attribute.prtevent"
  }
}

run "event_aware_pull_request_target_any_workflow" {
  command = plan

  variables {
    event_aware            = true
    refspec                = "pull_request_target"
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.prtanyworkflowevent/chainguard-dev/fixture|true",
    ])
    error_message = "event_aware must move a wildcard pull_request_target binding onto attribute.prtanyworkflowevent"
  }
}

run "event_aware_leaves_other_selectors_alone" {
  command = plan

  variables {
    event_aware  = true
    refspec      = "*"
    workflow_ref = "*"
    additional_federations = [
      { refspec = "pull_request", workflow_ref = ".github/workflows/presubmit.yaml" },
      { refspec = "version_tags", workflow_ref = ".github/workflows/release.yaml" },
    ]
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyrefanyworkflow/chainguard-dev/fixture",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.pullrequest/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/presubmit.yaml",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.versiontags/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/release.yaml",
    ])
    error_message = "event_aware must not change the any-ref, pull_request or version_tags attributes"
  }
}

run "event_aware_additional_federations" {
  command = plan

  variables {
    event_aware  = true
    workflow_ref = "*"
    additional_federations = [
      { refspec = "refs/heads/main", workflow_ref = ".github/workflows/provision-stages.yaml" },
      { refspec = "pull_request_target", workflow_ref = ".github/workflows/tf-verify.yaml" },
    ]
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyworkflowevent/chainguard-dev/fixture|refs/heads/main",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactevent/chainguard-dev/fixture|refs/heads/main|chainguard-dev/fixture/.github/workflows/provision-stages.yaml",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.prtevent/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/tf-verify.yaml",
    ])
    error_message = "event_aware must move additional branch and pull_request_target pairs onto the event-aware attributes"
  }
}

run "event_aware_also_pull_request_target" {
  command = plan

  variables {
    event_aware              = true
    also_pull_request_target = true
  }

  assert {
    condition = google_service_account_iam_binding.allow-impersonation.members == toset([
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactevent/chainguard-dev/fixture|refs/heads/main|chainguard-dev/fixture/.github/workflows/provision.yaml",
      "principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.prtevent/chainguard-dev/fixture|true|chainguard-dev/fixture/.github/workflows/provision.yaml",
    ])
    error_message = "also_pull_request_target must add a prtevent member for workflow_ref when event_aware is set"
  }
}

run "also_pull_request_target_requires_main" {
  command = plan

  variables {
    also_pull_request_target = true
    refspec                  = "refs/heads/release"
  }

  expect_failures = [google_service_account_iam_binding.allow-impersonation]
}

run "also_pull_request_target_rejects_named_selector" {
  command = plan

  variables {
    also_pull_request_target = true
    refspec                  = "pull_request_target"
  }

  expect_failures = [google_service_account_iam_binding.allow-impersonation]
}

run "also_pull_request_target_requires_pinned_workflow" {
  command = plan

  variables {
    also_pull_request_target = true
    workflow_ref             = "*"
    wildcard_justification   = "#0: fixture"
  }

  expect_failures = [google_service_account_iam_binding.allow-impersonation]
}
