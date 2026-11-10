terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

// Create the actual service account.
resource "google_service_account" "this" {
  project    = var.project_id
  account_id = var.name
  disabled   = var.disabled
}

locals {
  # NOTE: this module matches federated identities exclusively through the
  # provider's attribute mappings (attribute-match below). An earlier
  # regex-over-subject matcher (refPart/workflowPart/principalSubject) was
  # never consumed by any resource and was removed: it misled readers into
  # believing refspec values were regex-matched against OIDC subjects.
  #
  # event_aware swaps the four attributes that cannot tell a
  # pull_request_target run from a push to its base branch for the provider's
  # event-aware forms. Those carry the bare ref only for push,
  # workflow_dispatch and schedule, so a branch binding matches only those
  # events, and the pull_request_target forms match only runs against main.
  exact-attribute                            = var.event_aware ? "exactevent" : "exact"
  exact-any-workflow-attribute               = var.event_aware ? "exactanyworkflowevent" : "exactanyworkflow"
  pull-request-target-attribute              = var.event_aware ? "prtevent" : "pullrequesttarget"
  pull-request-target-any-workflow-attribute = var.event_aware ? "prtanyworkflowevent" : "pullrequesttargetanyworkflow"

  exact = "attribute.${local.exact-attribute}/${join("|", [
    var.repository,
    var.refspec,
    "${var.repository}/${var.workflow_ref}",
  ])}"

  exact-any-ref = "attribute.exactanyref/${join("|", [
    var.repository,
    "${var.repository}/${var.workflow_ref}",
  ])}"

  exact-any-ref-any-workflow = "attribute.exactanyrefanyworkflow/${var.repository}"

  exact-any-workflow = "attribute.${local.exact-any-workflow-attribute}/${join("|", [
    var.repository,
    var.refspec,
  ])}"

  pull-request = "attribute.pullrequest/${join("|", [
    var.repository,
    "true",
    "${var.repository}/${var.workflow_ref}",
  ])}"

  pull-request-any-workflow = "attribute.pullrequestanyworkflow/${join("|", [
    var.repository,
    "true",
  ])}"

  pull-request-target = "attribute.${local.pull-request-target-attribute}/${join("|", [
    var.repository,
    "true",
    "${var.repository}/${var.workflow_ref}",
  ])}"

  pull-request-target-any-workflow = "attribute.${local.pull-request-target-any-workflow-attribute}/${join("|", [
    var.repository,
    "true",
  ])}"

  version-tags = "attribute.versiontags/${join("|", [
    var.repository,
    "true",
    "${var.repository}/${var.workflow_ref}",
  ])}"

  version-tags-any-workflow = "attribute.versiontagsanyworkflow/${join("|", [
    var.repository,
    "true",
  ])}"

  attribute-match = var.refspec == "*" ? (
    var.workflow_ref == "*" ? (
      local.exact-any-ref-any-workflow
      ) : (
      local.exact-any-ref
    )
    ) : (
    var.refspec == "pull_request" ? (
      var.workflow_ref == "*" ? (
        local.pull-request-any-workflow
        ) : (
        local.pull-request
      )
      ) : (
      var.refspec == "pull_request_target" ? (
        var.workflow_ref == "*" ? (
          local.pull-request-target-any-workflow
          ) : (
          local.pull-request-target
        )
        ) : (
        var.refspec == "version_tags" ? (
          var.workflow_ref == "*" ? (
            local.version-tags-any-workflow
            ) : (
            local.version-tags
          )
          ) : (
          var.workflow_ref == "*" ? (
            local.exact-any-workflow
            ) : (
            local.exact
          )
        )
      )
  ))

}

locals {
  additional_selector_attributes = {
    pull_request        = "attribute.pullrequest"
    pull_request_target = "attribute.${local.pull-request-target-attribute}"
    version_tags        = "attribute.versiontags"
  }

  # also_pull_request_target adds a pull_request_target member for workflow_ref,
  # for an identity that the same workflow uses both on main and on its pull
  # requests.
  federations = concat(var.additional_federations, var.also_pull_request_target ? [{
    refspec      = "pull_request_target"
    workflow_ref = var.workflow_ref
  }] : [])

  additional_members = [
    for federation in local.federations :
    "principalSet://iam.googleapis.com/${var.wif-pool}/${lookup(local.additional_selector_attributes, federation.refspec, "attribute.${local.exact-attribute}")}/${join("|", [
      var.repository,
      contains(keys(local.additional_selector_attributes), federation.refspec) ? "true" : federation.refspec,
      "${var.repository}/${federation.workflow_ref}",
    ])}"
  ]
}

// Create the IAM binding allowing workflows to impersonate the service account.
resource "google_service_account_iam_binding" "allow-impersonation" {
  service_account_id = google_service_account.this.name
  role               = "roles/iam.workloadIdentityUser"

  members = concat([
    "principalSet://iam.googleapis.com/${var.wif-pool}/${local.attribute-match}",
  ], local.additional_members)

  lifecycle {
    precondition {
      condition     = !var.also_pull_request_target || (var.refspec == "refs/heads/main" && var.workflow_ref != "*")
      error_message = "also_pull_request_target needs refspec = \"refs/heads/main\" and a pinned workflow_ref, because a pull_request_target member matches only runs against main."
    }
    precondition {
      condition     = var.audit_workflow_ref == "" || var.workflow_ref == "*"
      error_message = "audit_workflow_ref may only be specified when workflow_ref is '*'"
    }
    precondition {
      condition     = var.audit_refspec == "" || var.refspec == "*"
      error_message = "audit_refspec may only be specified when refspec is '*'"
    }
  }
}

// Wildcard policy. These check blocks only warn: a failed check is
// reported on plan and apply but never fails the run, so existing callers keep
// planning with zero diff. They will be promoted to variable validations once
// every remaining wildcard carries a justification.
locals {
  wildcard_in_use = var.refspec == "*" || var.workflow_ref == "*"

  wildcard_policy_violations = compact([
    local.wildcard_in_use && trimspace(var.wildcard_justification) == "" ? "unjustified-wildcard" : "",
    var.refspec != "*" && strcontains(var.refspec, "*") ? "glob-refspec" : "",
    var.workflow_ref != "*" && (strcontains(var.workflow_ref, "*") || strcontains(var.workflow_ref, "@")) ? "glob-or-ref-pinned-workflow_ref" : "",
    anytrue([for f in var.additional_federations : strcontains(f.workflow_ref, "@")]) ? "ref-pinned-additional-workflow_ref" : "",
    !local.wildcard_in_use && trimspace(var.wildcard_justification) != "" ? "stale-justification" : "",
  ])
}

check "wildcard_justified" {
  assert {
    condition     = !local.wildcard_in_use || trimspace(var.wildcard_justification) != ""
    error_message = "Service account '${var.name}' federates ${var.repository} with a wildcard (refspec = '${var.refspec}', workflow_ref = '${var.workflow_ref}') but has no wildcard_justification. This will become a hard error: set wildcard_justification to the reason and a tracking issue, or pin the binding to a concrete refspec and workflow_ref."
  }
}

check "refspec_not_glob" {
  assert {
    condition     = var.refspec == "*" || !strcontains(var.refspec, "*")
    error_message = "Service account '${var.name}' has refspec = '${var.refspec}'. The WIF provider matches attribute.exact literally, so a glob refspec never federates; it will be rejected. Use '*' with a wildcard_justification, or a concrete ref."
  }
}

check "workflow_ref_not_glob_or_ref_pinned" {
  assert {
    condition     = var.workflow_ref == "*" || (!strcontains(var.workflow_ref, "*") && !strcontains(var.workflow_ref, "@"))
    error_message = "Service account '${var.name}' has workflow_ref = '${var.workflow_ref}'. The WIF provider strips '@ref' from the token's workflow_ref (github-wif-provider attribute mappings), so an '@ref' suffix or a glob never matches; it will be rejected. Use a plain .github/workflows/ path and put the ref in refspec."
  }
}

# additional_federations already rejects '*' in variable validation, so only
# the '@ref' suffix needs a check here.
check "additional_workflow_refs_not_ref_pinned" {
  assert {
    condition     = !anytrue([for f in var.additional_federations : strcontains(f.workflow_ref, "@")])
    error_message = "Service account '${var.name}' has additional_federations workflow_ref(s) ${jsonencode([for f in var.additional_federations : f.workflow_ref if strcontains(f.workflow_ref, "@")])} with an '@ref' suffix. The WIF provider strips '@ref' from the token's workflow_ref, so these never match; it will be rejected. Use a plain .github/workflows/ path and put the ref in refspec."
  }
}

check "wildcard_justification_not_stale" {
  assert {
    condition     = local.wildcard_in_use || trimspace(var.wildcard_justification) == ""
    error_message = "Service account '${var.name}' sets wildcard_justification but uses neither refspec = '*' nor workflow_ref = '*'. Remove the justification now that the wildcard is gone."
  }
}
