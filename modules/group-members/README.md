# `group-members`

This module resolves Google groups to the emails of their members, including
members of nested groups, through the Cloud Identity API. Audit logs name the
member who acted, never the group it acted through, so log filters and alerts
that should recognize a group's members need this list. The `secret` module's
`group_members` input takes the `emails` output as is.

```hcl
module "oncall" {
  source = "chainguard-dev/common/infra//modules/group-members"

  groups = ["oncall@my-corp.dev"]
}

module "my-secret" {
  source = "chainguard-dev/common/infra//modules/secret"

  # ...
  version_managers = ["group:oncall@my-corp.dev"]
  group_members    = module.oncall.emails
}
```

The module reads at plan time, so membership changes take effect on the next
apply.

## Prerequisites

The identity that runs Terraform needs:

- The Workspace "Groups Reader" admin role, when it is a service account, and
  permission to view every nested group's members. Workspace admins assign the
  role; IAM can't grant it. Without it, the read fails with `Error 403 ...
  Permission denied for resource groups/<id> (or it may not exist)`, the same
  error a misspelled group address gets.
- A quota project with `cloudidentity.googleapis.com` enabled. A service
  account uses its own project. User credentials need the provider's
  `user_project_override = true`, which bills `billing_project` (or `project`),
  and `serviceusage.services.use` there; without them the read fails with
  `Error 403 ... requires a quota project`.

The Cloud Identity Groups API rejects federated principals, so a workflow must
impersonate a service account rather than call it with a Workload Identity
Federation token. Listing nested members needs a Google Workspace Enterprise or
Cloud Identity Premium edition.

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
| [google_cloud_identity_group_lookup.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/cloud_identity_group_lookup) | data source |
| [google_cloud_identity_group_transitive_memberships.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/cloud_identity_group_transitive_memberships) | data source |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_groups"></a> [groups](#input\_groups) | Lowercase emails of the Google groups to resolve, without a group: prefix (e.g. oncall@my-corp.dev). | `set(string)` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_emails"></a> [emails](#output\_emails) | Sorted, lowercase emails of each group's members, including members of nested groups, keyed by group email. Suits the secret module's group\_members input. |
<!-- END_TF_DOCS -->
