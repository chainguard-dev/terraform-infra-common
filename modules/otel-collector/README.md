# `otel-collector`

This module encapsulates producing a sidecar image for publishing otel collected
metrics, and granting the service account as which the sidecar runs permission
to write those metrics (so it's impossible to forget):

```
module "otel-collector" {
  source = "chainguard-dev/common/infra//modules/otel-collector"

  project_id      = var.project_id
  service_account = google_service_account.this.email
}

resource "google_cloud_run_v2_service" "this" {
  template {
    service_account = google_service_account.this.email
    containers {
      image = "..."

      // Specifying port is necessary when there are multiple containers.
      ports { container_port = 8080 }
    }
    // Install the sidecar!
    containers { image = module.otel-collector.image }
  }
}
```

This module is automatically invoked by the
[`regional-go-service`](../regional-go-service/README.md) module.

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_cosign"></a> [cosign](#provider\_cosign) | n/a |
| <a name="provider_google"></a> [google](#provider\_google) | n/a |
| <a name="provider_ko"></a> [ko](#provider\_ko) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [cosign_sign.otel-image](https://registry.terraform.io/providers/chainguard-dev/cosign/latest/docs/resources/sign) | resource |
| [google_project_iam_member.metrics-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [google_project_iam_member.trace-writer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/project_iam_member) | resource |
| [ko_build.otel-image](https://registry.terraform.io/providers/ko-build/ko/latest/docs/resources/build) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_otel_collector_image"></a> [otel\_collector\_image](#input\_otel\_collector\_image) | The otel collector image to use as a base. | `string` | `"chainguard/opentelemetry-collector-contrib:latest"` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | n/a | `string` | n/a | yes |
| <a name="input_service_account"></a> [service\_account](#input\_service\_account) | The service account as which the collector will run. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_image"></a> [image](#output\_image) | n/a |
<!-- END_TF_DOCS -->
