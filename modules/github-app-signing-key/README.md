# `github-app-signing-key`

Creates the Cloud KMS key that holds a GitHub App's private key, for a service that authenticates as its own App: a key ring, an import-only `RSA_SIGN_PKCS1_2048_SHA256` key in it, and `roles/cloudkms.signer` on the key for the service's service account. The service signs its App JWTs with the key through a `gcpkms://` key URI, so the private key never leaves Cloud KMS. For example, `NewApp` in the [`githubreconciler`](https://github.com/driftlessaf/go-driftlessaf/tree/main/reconcilers/githubreconciler) package of `chainguard.dev/driftlessaf` takes such a URI.

Terraform creates the key with no versions. GitHub generates the App's private key, and you import it as a key version after the first apply, as described in [Import the App's key](#import-the-apps-key).

## Usage

```hcl
module "app_signing_key" {
  source = "chainguard-dev/common/infra//modules/github-app-signing-key"

  project_id             = var.project_id
  key_ring_name          = "${var.name}-mybot"
  key_name               = "github-app-signing-key"
  signer_service_account = google_service_account.mybot.email
}

locals {
  github_app_key = "gcpkms://${module.app_signing_key.key_id}/cryptoKeyVersions/${var.github_app_key_version}"
}
```

Give the service the key URI only once a version has been imported, for example by creating the service or job only when the App ID is set. Until then the URI names a version that does not exist, and the service cannot sign.

## Import the App's key

Generate a private key on the App's settings page in GitHub, then import it with the [`import-github-app-key`](./import-github-app-key) script in this module's directory, passing the module's `key_id` output:

```sh
./import-github-app-key <downloaded>.pem \
  projects/<project>/locations/global/keyRings/<key_ring_name>/cryptoKeys/<key_name>
```

The script needs `gcloud`, with the Python `cryptography` package installed for the Python that `gcloud` runs on, and `openssl`. Run it with no arguments for its usage. It checks that the key is import-only with the right algorithm, imports the PEM through a new import job, waits for the version to be enabled, and checks that the version's public key matches the PEM. It then deletes the PEM and prints the number of the key version it created, which goes at the end of the key URI, and the key's SHA256 fingerprint, which GitHub lists on the App's settings page.

Terraform downloads the whole repository for a registry module, so after `terraform init` the script is also at `.terraform/modules/<module name>/modules/github-app-signing-key/import-github-app-key` in the root module's directory.

Rotating the App's key is another run of the script with the new key, which creates a new version. Point the key URI at the new version, and once the service signs with it, delete the old key on GitHub and disable the old version.

## Keys and key rings outlive Terraform

Cloud KMS deletes neither key rings nor keys. Destroying the key in Terraform schedules every version of it for destruction, including the imported App key, after which the App needs a new key generated in GitHub and imported. The module sets `prevent_destroy` on the key so that a plan that would destroy it fails instead. A destroyed key ring stays in the project, so creating it again under the same name fails.

Changing `key_ring_name`, `key_name` or `destroy_scheduled_duration` replaces the key, and `prevent_destroy` makes that plan fail. To move an existing key ring and key into this module, call it with the existing key ring's name, key's name and `destroy_scheduled_duration`, add `moved` blocks from their old addresses to `module.<name>.google_kms_key_ring.this`, `module.<name>.google_kms_crypto_key.this` and `module.<name>.google_kms_crypto_key_iam_member.signer`, and check that the plan shows only moves.

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
| [google_kms_crypto_key.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/kms_crypto_key) | resource |
| [google_kms_crypto_key_iam_member.signer](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/kms_crypto_key_iam_member) | resource |
| [google_kms_key_ring.this](https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/kms_key_ring) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_destroy_scheduled_duration"></a> [destroy\_scheduled\_duration](#input\_destroy\_scheduled\_duration) | How long a destroyed key version stays recoverable before Cloud KMS destroys it, such as "86400s". Null takes the Cloud KMS default of 30 days. Changing it replaces the key, so set it only on a new key. | `string` | `null` | no |
| <a name="input_key_name"></a> [key\_name](#input\_key\_name) | The name of the import-only key that holds the App's private key. | `string` | n/a | yes |
| <a name="input_key_ring_name"></a> [key\_ring\_name](#input\_key\_ring\_name) | The name of the key ring, in the global location. Cloud KMS never deletes a key ring, so a new name leaves the old ring and its key behind. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The project the key ring is created in. | `string` | n/a | yes |
| <a name="input_signer_service_account"></a> [signer\_service\_account](#input\_signer\_service\_account) | The email of the service account that signs the App's JWTs with the key. | `string` | n/a | yes |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_key_id"></a> [key\_id](#output\_key\_id) | The resource name of the key, projects/<project>/locations/global/keyRings/<ring>/cryptoKeys/<key>, which the import-github-app-key script takes. A gcpkms:// key URI appends /cryptoKeyVersions/<N>. |
<!-- END_TF_DOCS -->
