<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_ko"></a> [ko](#provider\_ko) | n/a |
| <a name="provider_kubernetes"></a> [kubernetes](#provider\_kubernetes) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [ko_build.inmem](https://registry.terraform.io/providers/ko-build/ko/latest/docs/resources/build) | resource |
| [kubernetes_manifest.inmem-ksvc](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/manifest) | resource |
| [kubernetes_manifest.svc-acct](https://registry.terraform.io/providers/hashicorp/kubernetes/latest/docs/resources/manifest) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_batch-size"></a> [batch-size](#input\_batch-size) | Optional cap on how much work to launch per dispatcher pass. Defaults to the concurrent work value when unset. | `number` | `null` | no |
| <a name="input_concurrent-work"></a> [concurrent-work](#input\_concurrent-work) | The amount of concurrent work to dispatch at a given time. | `number` | n/a | yes |
| <a name="input_name"></a> [name](#input\_name) | n/a | `string` | n/a | yes |
| <a name="input_namespace"></a> [namespace](#input\_namespace) | n/a | `string` | n/a | yes |
| <a name="input_reconciler-service"></a> [reconciler-service](#input\_reconciler-service) | The address of the k8s service to push keys to. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_receiver"></a> [receiver](#output\_receiver) | n/a |
<!-- END_TF_DOCS -->
