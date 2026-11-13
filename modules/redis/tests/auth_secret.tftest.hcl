# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

variables {
  project_id               = "fixture-project"
  name                     = "fixture"
  team                     = "fixture"
  region                   = "us-central1"
  zone                     = "us-central1-a"
  secret_accessor_sa_email = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels    = []
  secret_version_adder     = "serviceAccount:fixture@fixture-project.iam.gserviceaccount.com"
}

run "auth_disabled_stores_nothing" {
  command = plan

  variables {
    auth_enabled = false
  }

  assert {
    condition     = length(google_secret_manager_secret_version.auth) == 0
    error_message = "an instance without AUTH must not store a version"
  }
}

run "auth_version_is_write_only_and_tracks_the_instance" {
  command = apply

  assert {
    condition     = length(google_secret_manager_secret_version.auth) == 1
    error_message = "an AUTH-enabled instance must store its AUTH string"
  }

  # A secret_data value would put the AUTH string in state and make every
  # refresh read the payload.
  assert {
    condition     = google_secret_manager_secret_version.auth[0].secret_data == null
    error_message = "the AUTH version must be written through secret_data_wo, not secret_data"
  }

  assert {
    condition     = google_secret_manager_secret_version.auth[0].secret_data_wo_version == google_redis_instance.default.create_time
    error_message = "the AUTH version must be replaced when the instance is"
  }
}

run "connection_details_are_published_on_the_secret" {
  command = apply
  # A fresh state, so the instance is created with the overridden values.
  state_key = "connection_details"

  override_resource {
    target = google_redis_instance.default
    values = {
      host            = "10.0.0.3"
      port            = 6378
      server_ca_certs = [{ cert = "CA1" }, { cert = "CA2\n" }]
    }
  }

  assert {
    condition = local.connection_annotations == {
      "redis-host"          = "10.0.0.3"
      "redis-port"          = "6378"
      "redis-server-ca-pem" = "CA1\nCA2\n"
    }
    error_message = "the AUTH secret must carry the host, port and full CA bundle"
  }

  assert {
    condition     = module.redis_auth_secret[0].annotations == tomap(local.connection_annotations)
    error_message = "the connection annotations must be set on the AUTH secret"
  }
}

run "no_ca_annotation_without_tls" {
  command   = apply
  state_key = "no_ca"

  override_resource {
    target = google_redis_instance.default
    values = {
      host            = "10.0.0.3"
      port            = 6379
      server_ca_certs = []
    }
  }

  assert {
    condition     = !contains(keys(local.connection_annotations), "redis-server-ca-pem")
    error_message = "an instance without a server CA must not publish an empty CA annotation"
  }
}

# Plans no longer read the AUTH payload, so CI identities must not regain
# accessor through the deprecated ci_service_accounts input.
run "ci_service_accounts_get_no_accessor" {
  command = plan

  variables {
    ci_service_accounts = ["ci@fixture-project.iam.gserviceaccount.com"]
  }

  assert {
    condition     = !contains(module.redis_auth_secret[0].accessor_members, "serviceAccount:ci@fixture-project.iam.gserviceaccount.com")
    error_message = "ci_service_accounts must not be granted accessor on the AUTH secret"
  }

  assert {
    condition     = contains(module.redis_auth_secret[0].accessor_members, "serviceAccount:fixture@fixture-project.iam.gserviceaccount.com")
    error_message = "the workload's accessor must still be granted"
  }
}
