# `secret`

This module encapsulates the creation of a Google Secret Manager secret to hold
sensitive data in a manner that can be used as an environment variable or
volume with Cloud Run.  Unlike `configmap` this data is considered sensitive and
so it is NOT loaded directly by this logic, but by an authorized party.

The built-in alert policy fires on any version read or change except reads by
`service-accounts` and version management (add, enable, disable, destroy) by
`authorized-adder(s)` and `version_managers`. Audit logs name the identity that
acted, never a group it acted through, so a group's members are exempt only
when `group_members` lists them; otherwise every version a member adds raises
the alert. Version managers can't read payloads, and the alert fires when
anyone outside `service-accounts` reads one. Version changes always reach the
Admin Activity audit log, but reads appear only when the project enables Data
Access audit logs (`DATA_READ`) for Secret Manager, so without them the alert
can't see reads.

Set `access_log_metric_name` to also count matching entries in a project-local
log-based metric. A monitoring project that includes this project in its metrics
scope can alert on that counter using its existing notification channels. The
counter uses the same identity exemptions as the local alert and adds no identity
labels. The caller owns the scope link and metric alert policy; this option alone
does not route notifications. It defaults to null and requires the same audit-log
configuration as the local alert.

```hcl
module "my-secret" {
  source = "chainguard-dev/common/infra//modules/secret"

  project_id = var.project_id
  name       = "my-secret"

  # What the service accessing this configuration will run as.
  service-account = google_service_account.foo.email

  # What group of identities are authorized to add new secret versions.
  authorized-adder = "group:oncall@my-corp.dev"

  # Optionally: channels to notify if this secret is manipulated.
  notification-channels = [ ... ]
}
```

To let a group rotate the secret (add, list, get, enable, disable, and destroy
versions, and view the secret) without alerting on its members, name it in
`version_managers` and pass its members, resolved with the `group-members`
module, as `group_members`:

```hcl
module "oncall" {
  source = "chainguard-dev/common/infra//modules/group-members"

  groups = ["oncall@my-corp.dev"]
}

module "my-secret" {
  source = "chainguard-dev/common/infra//modules/secret"

  project_id       = var.project_id
  name             = "my-secret"
  service-account  = google_service_account.foo.email
  authorized-adder = "group:oncall@my-corp.dev"

  version_managers = ["group:oncall@my-corp.dev"]
  group_members    = module.oncall.emails

  notification-channels = [ ... ]
}
```

A Cloud Run service reads the secret like this:

```hcl

module "foo-service" {
  source     = "chainguard-dev/common/infra//modules/regional-go-service"
  project_id = var.project_id
  name       = "foo"
  regions    = module.networking.regional-networks

  service_account = google_service_account.foo.email
  containers = {
    "foo" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/foo"
      }
      ports = [{ container_port = 8080 }]
      volume_mounts = [{
        name       = "foo"
        mount_path = "/var/run/foo/"
      }]
    }
  }
  volumes = [{
    name = "foo"
    secret = {
      secret = module.my-secret.secret_id
      items = [{
        version = "latest"
        path    = "my-filename"
      }]
    }
  }]
}
```

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
| [google_logging_metric.anomalous_secret_access](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_metric) | resource |
| [google_monitoring_alert_policy.anomalous-secret-access](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy) | resource |
| [google_secret_manager_secret.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret) | resource |
| [google_secret_manager_secret_iam_binding.authorize-service-access](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_iam_binding) | resource |
| [google_secret_manager_secret_iam_binding.authorize-version-adder](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_iam_binding) | resource |
| [google_secret_manager_secret_iam_binding.authorize-version-manager](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_iam_binding) | resource |
| [google_secret_manager_secret_version.placeholder](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/secret_manager_secret_version) | resource |
| [google_tags_tag_binding.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/tags_tag_binding) | resource |
| [google_project.project](https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/project) | data source |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_access_log_metric_name"></a> [access\_log\_metric\_name](#input\_access\_log\_metric\_name) | Optional log counter name for monitoring this secret through a shared metrics scope. Null creates no metric. Uses the same unauthorized-access filter as the local alert. | `string` | `null` | no |
| <a name="input_alert_documentation"></a> [alert\_documentation](#input\_alert\_documentation) | Markdown documentation attached to the abnormal-access alert (e.g. what to check and a Logs Explorer link). Empty (default) attaches none. | `string` | `""` | no |
| <a name="input_authorized-adder"></a> [authorized-adder](#input\_authorized-adder) | A member-style representation of the identity authorized to add new secret values (e.g. group:oncall@my-corp.dev). | `string` | n/a | yes |
| <a name="input_authorized-adders"></a> [authorized-adders](#input\_authorized-adders) | List of additional identities authorized to add new secret values (e.g. ["serviceAccount:sa@project.iam.gserviceaccount.com"]). Use this when multiple identities need version adder permissions. | `list(string)` | `[]` | no |
| <a name="input_create_placeholder_version"></a> [create\_placeholder\_version](#input\_create\_placeholder\_version) | Whether to create a placeholder secret version to avoid bad reference on first deploy. | `bool` | `false` | no |
| <a name="input_group_members"></a> [group\_members](#input\_group\_members) | Member emails of each group in authorized-adder, authorized-adders, and version\_managers, keyed by the group's email. Audit logs name the member who acted, never the group, so the abnormal-access alert recognizes a group's members only through this map. Resolve it from the groups (e.g. with the group-members module) so it follows membership changes. Grants no IAM. | `map(list(string))` | `{}` | no |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels to apply to the secret. | `map(string)` | `{}` | no |
| <a name="input_name"></a> [name](#input\_name) | The name to give the secret. | `string` | n/a | yes |
| <a name="input_notification-channels"></a> [notification-channels](#input\_notification-channels) | The channels to notify if the configuration data is improperly accessed. | `list(string)` | n/a | yes |
| <a name="input_product"></a> [product](#input\_product) | Unused: cost attribution comes from Resource Manager tags, not labels. Kept so existing callers still plan. | `string` | `null` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_replication_locations"></a> [replication\_locations](#input\_replication\_locations) | List of GCP regions for user\_managed replication. When null (default), uses automatic replication. | `list(string)` | `null` | no |
| <a name="input_resource_manager_tags"></a> [resource\_manager\_tags](#input\_resource\_manager\_tags) | Resource Manager tags to bind to the secret, as tagKeys/<id> => tagValues/<id>. | `map(string)` | `{}` | no |
| <a name="input_service-account"></a> [service-account](#input\_service-account) | (Deprecated: Use service-accounts instead) The email of the service account that will access the secret. | `string` | `""` | no |
| <a name="input_service-accounts"></a> [service-accounts](#input\_service-accounts) | The emails of the service accounts that will access the secret. | `list(string)` | `[]` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources (replaces deprecated 'squad'). | `string` | n/a | yes |
| <a name="input_version_managers"></a> [version\_managers](#input\_version\_managers) | Member-style identities (e.g. group:oncall@my-corp.dev) that manage this secret's versions: add, list, get, enable, disable, and destroy them, and view the secret. They can't read version payloads, and the abnormal-access alert ignores their version management. Only user:, group:, and serviceAccount: members are accepted, because the alert recognizes identities by email. The module owns the secret's roles/secretmanager.secretVersionManager and roles/secretmanager.viewer bindings once this is set, so don't grant those roles on the secret elsewhere. | `list(string)` | `[]` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_access_log_metric_name"></a> [access\_log\_metric\_name](#output\_access\_log\_metric\_name) | Unauthorized-access log counter name, or null when disabled. Alert policies belong in the caller's monitoring project. |
| <a name="output_authorized_secret_id"></a> [authorized\_secret\_id](#output\_authorized\_secret\_id) | The ID of the secret after its accessor binding is applied. |
| <a name="output_secret_id"></a> [secret\_id](#output\_secret\_id) | The ID of the secret. |
<!-- END_TF_DOCS -->
