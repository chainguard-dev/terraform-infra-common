# `limited-concat`

This module takes two strings, `prefix` and `suffix`, and returns the concatenation
of the two, up to a given `limit` length. If the concatenation is longer than `limit`,
the `prefix` is shortened so that
`length(output.result) == var.limit`

```hcl
resource "random_string" "suffix" {
  length  = 4
  upper   = false
  special = false
}

module "limited-name" {
  source = "chainguard-dev/common/infra//modules/limited-concat"

  prefix = "foo-bar-baz"
  suffix = "-${random_string.suffix.result}"
  limit  = 12
}

output "limited-name" {
  value = module.limited-name.result
}
# Output: foo-bar-wxyz
```

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

No modules.

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_limit"></a> [limit](#input\_limit) | Maximum length of the resulting concatenation. | `number` | n/a | yes |
| <a name="input_prefix"></a> [prefix](#input\_prefix) | First part of the result, will be shortened if length(prefix)+length(suffix) > limit. | `string` | n/a | yes |
| <a name="input_suffix"></a> [suffix](#input\_suffix) | Second part of the result, included in whole. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_result"></a> [result](#output\_result) | The concatenation of prefix and suffix, with limit applied. |
<!-- END_TF_DOCS -->
