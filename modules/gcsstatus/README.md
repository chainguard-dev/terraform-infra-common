# `gcsstatus`

This module provisions a GCS bucket for storing reconciliation status as JSON
objects, the GCS-backed analog of the [`ocistatus`](../ocistatus) module. It is
the infrastructure counterpart of the `gcsstatusmanager`
(`public/go-driftlessaf/reconcilers/gcsstatusmanager`): a reconciler writes each
key's status to `gs://<bucket>/<identity>/<key>` and reads it back to skip
already-completed work on a requeue.

It creates the bucket (uniform access, public access prevention enforced,
versioning off, with a globally-unique random name suffix) and grants the given
service accounts access: writers get `roles/storage.objectUser` and read-only
consumers (built with `gcsstatusmanager.NewReadOnly`) get
`roles/storage.objectViewer`. Because a status write is a plain object overwrite,
no `repoAdmin`/delete privilege is required (unlike `ocistatus`, whose
attestation replacement deletes referrer manifests). An optional
`lifecycle_age_days` adds a TTL so abandoned status objects are garbage-collected.

```hcl
module "reconciler_status" {
  source = "chainguard-dev/common/infra//modules/gcsstatus"

  project_id              = var.project_id
  name                    = "${var.name}-status"
  location                = var.primary_region
  writer_service_accounts = [google_service_account.reconciler.member]
  lifecycle_age_days      = 30
}

module "reconciler" {
  source = "chainguard-dev/common/infra//modules/regional-go-reconciler"

  # ... other config ...
  containers = {
    reconciler = {
      env = [{
        name  = "STATUS_BUCKET"
        value = module.reconciler_status.bucket_name
      }]
    }
  }
}
```

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_random"></a> [random](#provider\_random) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [google_storage_bucket.status](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket) | resource |
| [google_storage_bucket_iam_member.readers](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket_iam_member) | resource |
| [google_storage_bucket_iam_member.writers](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket_iam_member) | resource |
| [random_string.suffix](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/string) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_lifecycle_age_days"></a> [lifecycle\_age\_days](#input\_lifecycle\_age\_days) | When > 0, adds a bucket lifecycle rule that deletes status objects older than this many days. Status objects are cheap and self-heal, so a TTL bounds the cost of abandoned entries. 0 disables the rule. | `number` | `0` | no |
| <a name="input_location"></a> [location](#input\_location) | The location (region or multi-region) for the status bucket. | `string` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | Base name for the status bucket. A short random suffix is appended to keep the (globally unique) bucket name collision-free. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The GCP project ID where the status bucket will be created. | `string` | n/a | yes |
| <a name="input_reader_service_accounts"></a> [reader\_service\_accounts](#input\_reader\_service\_accounts) | Service account members granted read-only (roles/storage.objectViewer) access, for consumers built with gcsstatusmanager.NewReadOnly. | `list(string)` | `[]` | no |
| <a name="input_writer_service_accounts"></a> [writer\_service\_accounts](#input\_writer\_service\_accounts) | Service account members (e.g. serviceAccount:foo@project.iam.gserviceaccount.com) granted read+write on the status bucket. gcsstatusmanager overwrites objects, so roles/storage.objectUser (no repoAdmin/delete privilege needed for writes) is granted. | `list(string)` | `[]` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_bucket_name"></a> [bucket\_name](#output\_bucket\_name) | The name of the status bucket, for wiring into the reconciler's status bucket env var (e.g. STATUS\_BUCKET). Pair it with client.Bucket(name) and a gcsstatusmanager identity prefix. |
| <a name="output_bucket_url"></a> [bucket\_url](#output\_bucket\_url) | The gs:// URL of the status bucket. |
<!-- END_TF_DOCS -->
