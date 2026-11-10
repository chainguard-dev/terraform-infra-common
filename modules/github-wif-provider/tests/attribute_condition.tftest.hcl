# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google" {}

variables {
  project_id            = "fixture-project"
  name                  = "fixture-pool"
  github_org            = "example-org"
  notification_channels = []
}

run "defaults_keep_org_only_condition" {
  command = plan
  assert {
    condition     = google_iam_workload_identity_pool_provider.this.attribute_condition == "assertion.repository_owner == 'example-org'"
    error_message = "with default inputs the condition must equal the original org-only condition"
  }
}

run "org_id_only" {
  command = plan
  variables {
    github_org_id = "123456789"
  }
  assert {
    condition     = google_iam_workload_identity_pool_provider.this.attribute_condition == "assertion.repository_owner == 'example-org' && assertion.repository_owner_id == '123456789'"
    error_message = "github_org_id must add a repository_owner_id clause"
  }
}

run "allowlist_sorted_deduped" {
  command = plan
  variables {
    github_org_id        = "123456789"
    allowed_repositories = ["example-org/repo-b", "example-org/repo-a", "example-org/repo-b"]
  }
  assert {
    condition     = google_iam_workload_identity_pool_provider.this.attribute_condition == "assertion.repository_owner == 'example-org' && assertion.repository_owner_id == '123456789' && assertion.repository in [\"example-org/repo-a\",\"example-org/repo-b\"]"
    error_message = "allowed_repositories must add a sorted, de-duplicated assertion.repository in [...] clause"
  }
}

run "foreign_org_repo_rejected" {
  command = plan
  variables {
    allowed_repositories = ["other-org/repo-a"]
  }
  expect_failures = [var.allowed_repositories]
}

run "bare_repo_rejected" {
  command = plan
  variables {
    allowed_repositories = ["repo-a"]
  }
  expect_failures = [var.allowed_repositories]
}

run "non_numeric_org_id_rejected" {
  command = plan
  variables {
    github_org_id = "example-org"
  }
  expect_failures = [var.github_org_id]
}
