# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Wildcard policy. These rules currently ship as check blocks,
# which only warn, so the rejected shapes use expect_failures on the check.

mock_provider "google" {}

# notification_channels is deliberately omitted: it is deprecated and defaults
# to [], so every run here also proves callers may leave it out.
variables {
  project_id   = "fixture-project"
  name         = "fixture-sa"
  wif-pool     = "projects/123/locations/global/workloadIdentityPools/fixture"
  repository   = "chainguard-dev/fixture"
  refspec      = "refs/heads/main"
  workflow_ref = ".github/workflows/x.yaml"
}

run "pinned_branch_and_workflow" {
  command = plan

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "pull_request_and_workflow" {
  command = plan

  variables {
    refspec = "pull_request"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "pull_request_target_and_workflow" {
  command = plan

  variables {
    refspec = "pull_request_target"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "version_tags_and_workflow" {
  command = plan

  variables {
    refspec = "version_tags"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "pinned_tag_and_workflow" {
  command = plan

  variables {
    refspec = "refs/tags/v1"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "justified_any_ref" {
  command = plan

  variables {
    refspec                = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "justified_any_workflow" {
  command = plan

  variables {
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "justified_any_ref_any_workflow" {
  command = plan

  variables {
    refspec                = "*"
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "justified_pull_request_any_workflow" {
  command = plan

  variables {
    refspec                = "pull_request"
    workflow_ref           = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "expected no wildcard policy violations"
  }
}

run "notification_channels_still_accepted" {
  command = plan

  variables {
    notification_channels = ["projects/fixture-project/notificationChannels/1"]
  }

  assert {
    condition     = length(output.wildcard_policy_violations) == 0
    error_message = "passing the deprecated notification_channels must keep working"
  }
}

run "unjustified_any_ref" {
  command = plan

  variables {
    refspec = "*"
  }

  expect_failures = [check.wildcard_justified]

  assert {
    condition     = contains(output.wildcard_policy_violations, "unjustified-wildcard")
    error_message = "expected unjustified-wildcard in wildcard_policy_violations"
  }
}

run "unjustified_any_workflow" {
  command = plan

  variables {
    workflow_ref = "*"
  }

  expect_failures = [check.wildcard_justified]

  assert {
    condition     = contains(output.wildcard_policy_violations, "unjustified-wildcard")
    error_message = "expected unjustified-wildcard in wildcard_policy_violations"
  }
}

run "whitespace_justification_is_unjustified" {
  command = plan

  variables {
    refspec                = "*"
    wildcard_justification = " \t "
  }

  expect_failures = [check.wildcard_justified]

  assert {
    condition     = contains(output.wildcard_policy_violations, "unjustified-wildcard")
    error_message = "expected unjustified-wildcard in wildcard_policy_violations"
  }
}

run "glob_refspec" {
  command = plan

  variables {
    refspec = "refs/heads/*"
  }

  expect_failures = [check.refspec_not_glob]

  assert {
    condition     = contains(output.wildcard_policy_violations, "glob-refspec")
    error_message = "expected glob-refspec in wildcard_policy_violations"
  }
}

run "ref_pinned_workflow_ref" {
  command = plan

  variables {
    workflow_ref = ".github/workflows/x.yaml@refs/heads/main"
  }

  expect_failures = [check.workflow_ref_not_glob_or_ref_pinned]

  assert {
    condition     = contains(output.wildcard_policy_violations, "glob-or-ref-pinned-workflow_ref")
    error_message = "expected glob-or-ref-pinned-workflow_ref in wildcard_policy_violations"
  }
}

run "ref_pinned_additional_workflow_ref" {
  command = plan

  variables {
    additional_federations = [{
      refspec      = "refs/heads/main"
      workflow_ref = ".github/workflows/x.yaml@refs/heads/main"
    }]
  }

  expect_failures = [check.additional_workflow_refs_not_ref_pinned]

  assert {
    condition     = contains(output.wildcard_policy_violations, "ref-pinned-additional-workflow_ref")
    error_message = "expected ref-pinned-additional-workflow_ref in wildcard_policy_violations"
  }
}

run "plain_additional_workflow_ref_is_compliant" {
  command = plan

  variables {
    additional_federations = [{
      refspec      = "refs/heads/main"
      workflow_ref = ".github/workflows/x.yaml"
    }]
  }

  assert {
    condition     = !contains(output.wildcard_policy_violations, "ref-pinned-additional-workflow_ref")
    error_message = "a plain additional_federations workflow_ref must not be flagged"
  }
}

run "glob_workflow_ref" {
  command = plan

  variables {
    workflow_ref = ".github/workflows/*.yaml"
  }

  expect_failures = [check.workflow_ref_not_glob_or_ref_pinned]

  assert {
    condition     = contains(output.wildcard_policy_violations, "glob-or-ref-pinned-workflow_ref")
    error_message = "expected glob-or-ref-pinned-workflow_ref in wildcard_policy_violations"
  }
}

run "stale_justification" {
  command = plan

  variables {
    wildcard_justification = "#0: fixture"
  }

  expect_failures = [check.wildcard_justification_not_stale]

  assert {
    condition     = contains(output.wildcard_policy_violations, "stale-justification")
    error_message = "expected stale-justification in wildcard_policy_violations"
  }
}

# The checks must not change what the binding grants: a justified or
# unjustified wildcard renders the same member string it did before the
# wildcard checks were added.
run "members_unchanged_for_any_ref" {
  command = plan

  variables {
    refspec = "*"
  }

  expect_failures = [check.wildcard_justified]

  assert {
    condition     = google_service_account_iam_binding.allow-impersonation.members == toset(["principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyref/chainguard-dev/fixture|chainguard-dev/fixture/.github/workflows/x.yaml"])
    error_message = "the primary member string changed"
  }
}

run "members_unchanged_for_justified_any_ref" {
  command = plan

  variables {
    refspec                = "*"
    wildcard_justification = "#0: fixture"
  }

  assert {
    condition     = google_service_account_iam_binding.allow-impersonation.members == toset(["principalSet://iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/fixture/attribute.exactanyref/chainguard-dev/fixture|chainguard-dev/fixture/.github/workflows/x.yaml"])
    error_message = "setting wildcard_justification changed the member string"
  }
}
