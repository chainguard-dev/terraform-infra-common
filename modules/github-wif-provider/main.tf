resource "google_iam_workload_identity_pool" "this" {
  project                   = var.project_id
  provider                  = google-beta
  workload_identity_pool_id = var.name
  display_name              = "Pool for ${var.name}"
}

resource "google_iam_workload_identity_pool_provider" "this" {
  project                            = var.project_id
  provider                           = google-beta
  workload_identity_pool_id          = google_iam_workload_identity_pool.this.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider" # This gets 4-32 alphanumeric characters (and '-')
  display_name                       = "GitHub provider"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  # ref: https://cloud.google.com/iam/docs/workload-identity-federation#mapping
  # ref: https://github.com/google/cel-spec/blob/master/doc/langdef.md#list-of-standard-definitions
  attribute_mapping = {
    # Don't use the GitHub subject because it it less specific than ours, which also captures:
    #   - The pull request number via `refs/pull/N/merge`, and
    #   - The workflow file.
    # We don't include assertion.repository because it is redundant with the prefix on assertion.workflow_ref.
    # We use assertion.ref instead of the ref included in workflow_ref so that we get the PR number on pull_request_target PRs.
    "google.subject" = "assertion.workflow_ref.split('@')[0] + '|' + assertion.ref"

    # assertion.ref has one of the forms:
    #   - Branch: refs/heads/main
    #   - Pull Request: refs/pull/1/merge
    #   - Tag: refs/tags/v1.0.0
    # assertion.workflow_ref has one of the forms:
    #   - .github/workflows/secrets.yaml@refs/heads/main
    #   - .github/workflows/secrets.yaml@refs/tags/v1.0.0
    #   - .github/workflows/secrets.yaml@refs/pull/1/merge
    "attribute.exact"                        = "assertion.repository + '|' + assertion.ref + '|' + assertion.workflow_ref.split('@')[0]"
    "attribute.exactanyref"                  = "assertion.repository + '|' + assertion.workflow_ref.split('@')[0]"
    "attribute.exactanyrefanyworkflow"       = "assertion.repository"
    "attribute.exactanyworkflow"             = "assertion.repository + '|' + assertion.ref"
    "attribute.pullrequest"                  = "assertion.repository + '|' + (assertion.ref.matches('^refs/pull/[0-9]+/merge$') ? 'true' : 'false') + '|' + assertion.workflow_ref.split('@')[0]"
    "attribute.pullrequestanyworkflow"       = "assertion.repository + '|' + (assertion.ref.matches('^refs/pull/[0-9]+/merge$') ? 'true' : 'false')"
    "attribute.pullrequesttarget"            = "assertion.repository + '|' + (assertion.ref == 'refs/heads/main' ? 'true' : 'false') + '|' + assertion.workflow_ref.split('@')[0]"
    "attribute.pullrequesttargetanyworkflow" = "assertion.repository + '|' + (assertion.ref == 'refs/heads/main' ? 'true' : 'false')"
    "attribute.versiontags"                  = "assertion.repository + '|' + (assertion.ref.matches('^refs/tags/v[0-9]+([.][0-9]+([.][0-9]+)?)?$') ? 'true' : 'false') + '|' + assertion.workflow_ref.split('@')[0]"
    "attribute.versiontagsanyworkflow"       = "assertion.repository + '|' + (assertion.ref.matches('^refs/tags/v[0-9]+([.][0-9]+([.][0-9]+)?)?$') ? 'true' : 'false')"
  }

  attribute_condition = "assertion.repository_owner == '${var.github_org}'"
}
