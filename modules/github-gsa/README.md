# `github-gsa`

This module creates a Google Service Account that can be assumed by particular
GitHub Actions workflows. It is intended to be used in conjunction with the
[`github-wif-provider` module](../github-wif-provider/README.md).

```hcl
module "github-wif" {
  source = "chainguard-dev/common/infra//modules/github-wif-provider"

  project_id = var.project_id
  name       = "my-wif-pool"
}

module "foo" {
  source = "chainguard-dev/common/infra//modules/github-gsa"

  project_id = var.project_id
  name       = "foo"
  wif-pool   = module.github-wif.pool_name

  repository   = "the-org/the-repo"
  refspec      = "refs/heads/main"
  workflow_ref = ".github/workflows/my-workflow.yaml"
}
```

The primary `refspec` and `workflow_ref` retain their existing selector and
wildcard behavior. `additional_federations` defaults to an empty list and adds
pinned pairs to the same authoritative IAM binding. Additional pairs reject `*`
in either field. Concrete refs use `attribute.exact`; the existing named
`pull_request`, `pull_request_target`, and `version_tags` selectors use their
corresponding bounded provider attributes. For example:

```hcl
additional_federations = [{
  refspec      = "refs/heads/main"
  workflow_ref = ".github/workflows/another-workflow.yaml"
}]
```

A `pull_request_target` run carries its base branch in the `ref` claim, so a
`refs/heads/main` binding also accepts pull request runs of the same workflow,
and so do events such as `issue_comment` and `workflow_run`. Setting
`event_aware = true` matches on the provider's event-aware attributes instead,
here and in `additional_federations`: a branch refspec then accepts only
`push`, `workflow_dispatch` and `schedule` runs, and `pull_request_target`
accepts only `pull_request_target` runs against main. Leave it unset for a
workflow that federates on any other event. An identity that one workflow uses
both on main and on its pull requests keeps `refspec = "refs/heads/main"` and
sets `also_pull_request_target = true`.

## Wildcards

`refspec = "*"` lets any ref federate, and `workflow_ref = "*"` lets any
workflow federate. Either one needs a `wildcard_justification` that says why
and names a tracking issue:

```hcl
refspec                = "*"
workflow_ref           = ".github/workflows/provision.yaml"
wildcard_justification = "#123: dispatch-only provisioner; pinned callers in additional_federations"
```

Today a missing justification is a plan-time warning (a failed `check`
block); it will become a hard validation error. Glob values such as
`refs/heads/*` or `.github/workflows/*.yaml`, and a `workflow_ref` with an
`@ref` suffix, never federate: the provider matches these attributes
literally and strips `@ref` from the token. They warn now and will be
rejected. Put the ref in `refspec` instead. The `@ref` rule covers each
`additional_federations` entry too; a glob there is already rejected by the
variable's validation. A justification on a binding with no wildcard also
warns, so it is removed once the wildcard is gone. The
`wildcard_policy_violations` output lists the rules a binding breaks.

`notification_channels` is deprecated and has no effect.

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
| ---- | ------- |
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.9 |
| <a name="requirement_google"></a> [google](#requirement\_google) | >= 8.7.0 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | >= 8.7.0 |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [google_service_account.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account) | resource |
| [google_service_account_iam_binding.allow-impersonation](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_account_iam_binding) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_additional_federations"></a> [additional\_federations](#input\_additional\_federations) | Additional pinned refspec and workflow pairs allowed to federate with this identity. Named refspec selectors retain their existing provider attribute semantics. | <pre>list(object({<br/>    refspec      = string<br/>    workflow_ref = string<br/>  }))</pre> | `[]` | no |
| <a name="input_also_pull_request_target"></a> [also\_pull\_request\_target](#input\_also\_pull\_request\_target) | Also accept pull\_request\_target runs, against main, of workflow\_ref. For an identity that a workflow uses both on main and on its pull requests. Needs refspec 'refs/heads/main' and a pinned workflow\_ref. | `bool` | `false` | no |
| <a name="input_audit_refspec"></a> [audit\_refspec](#input\_audit\_refspec) | Deprecated: has no effect. It only fed a subject-regex matcher that no resource ever consumed; federation is matched via the provider's attribute mappings, which have no audited-wildcard form. Retained so existing callers do not break; do not set it expecting tightened scoping. | `string` | `""` | no |
| <a name="input_audit_workflow_ref"></a> [audit\_workflow\_ref](#input\_audit\_workflow\_ref) | Deprecated: has no effect, for the same reason as audit\_refspec. | `string` | `""` | no |
| <a name="input_disabled"></a> [disabled](#input\_disabled) | Disable the service account, so that no workflow can mint a token for it. Retire an identity this way before deleting it, so an unknown caller fails reversibly. | `bool` | `false` | no |
| <a name="input_event_aware"></a> [event\_aware](#input\_event\_aware) | Match on the provider's event-aware attributes, so a branch refspec matches only push, workflow\_dispatch and schedule runs, and 'pull\_request\_target' matches only pull\_request\_target runs against main. Also applies to additional\_federations. The pool's provider must map exactevent, exactanyworkflowevent, prtevent and prtanyworkflowevent. | `bool` | `false` | no |
| <a name="input_name"></a> [name](#input\_name) | The name to give the service account. | `string` | n/a | yes |
| <a name="input_notification_channels"></a> [notification\_channels](#input\_notification\_channels) | Deprecated: has no effect. No resource in this module reads it, and it never alerted on misuse. Retained so existing callers do not break. | `list(string)` | `[]` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_refspec"></a> [refspec](#input\_refspec) | The refspec to allow to federate with this identity. | `string` | n/a | yes |
| <a name="input_repository"></a> [repository](#input\_repository) | The name of the repository to allow to assume this identity. | `string` | n/a | yes |
| <a name="input_wif-pool"></a> [wif-pool](#input\_wif-pool) | The name of the Workload Identity Federation pool. | `string` | n/a | yes |
| <a name="input_wildcard_justification"></a> [wildcard\_justification](#input\_wildcard\_justification) | Why this identity needs refspec = "*" or workflow\_ref = "*", with a tracking issue (e.g. "#123: dispatch-only provisioner; pinned callers in additional\_federations"). Required whenever either field is "*". Today a missing justification is a plan-time warning; it will become a hard validation error. | `string` | `""` | no |
| <a name="input_workflow_ref"></a> [workflow\_ref](#input\_workflow\_ref) | The workflow to allow to federate with this identity (e.g. .github/workflows/deploy.yaml). | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_email"></a> [email](#output\_email) | n/a |
| <a name="output_wildcard_policy_violations"></a> [wildcard\_policy\_violations](#output\_wildcard\_policy\_violations) | Wildcard-policy rules this binding breaks (unjustified-wildcard, glob-refspec, glob-or-ref-pinned-workflow\_ref, ref-pinned-additional-workflow\_ref, stale-justification). Empty when compliant. |
<!-- END_TF_DOCS -->
