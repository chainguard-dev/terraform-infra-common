# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0
#
# Run in CI by .github/workflows/tf-module-tests.yaml.

mock_provider "google" {}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  dns_zone   = "fixture-zone"
  team       = "fixture"
}

run "no_cors_policy_leaves_the_url_map_and_schemes_alone" {
  command = plan
  variables {
    public-services = {
      "api.example.dev" = { name = "api" }
    }
  }
  assert {
    condition     = length(google_compute_url_map.public-service.path_matcher[0].default_route_action) == 0
    error_message = "a service without cors_policy must not emit a default_route_action (classic load balancers reject one)"
  }
  assert {
    condition     = google_compute_backend_service.public-services["api.example.dev"].load_balancing_scheme == "EXTERNAL"
    error_message = "the backend service must default to the classic EXTERNAL scheme"
  }
  assert {
    condition     = google_compute_global_forwarding_rule.this.load_balancing_scheme == "EXTERNAL"
    error_message = "the forwarding rule must default to the classic EXTERNAL scheme"
  }
}

run "cors_policy_is_carried_on_the_default_route_action" {
  command = plan
  variables {
    public-services = {
      "api.example.dev" = {
        name                  = "api"
        load_balancing_scheme = "EXTERNAL_MANAGED"
        cors_policy = {
          allow_origins = ["https://console.example.dev", "http://localhost:3000"]
        }
      }
    }
    forwarding_rule_load_balancing = {
      load_balancing_scheme = "EXTERNAL_MANAGED"
    }
  }
  assert {
    condition     = google_compute_url_map.public-service.path_matcher[0].default_route_action[0].cors_policy[0].allow_origins == tolist(["https://console.example.dev", "http://localhost:3000"])
    error_message = "cors_policy.allow_origins must reach the URL map"
  }
  assert {
    condition     = google_compute_url_map.public-service.path_matcher[0].default_route_action[0].cors_policy[0].allow_methods == tolist(["GET", "HEAD", "POST", "OPTIONS"])
    error_message = "allow_methods must default to the JSON-API set"
  }
  assert {
    condition     = google_compute_url_map.public-service.path_matcher[0].default_route_action[0].cors_policy[0].allow_headers == tolist(["Authorization", "Content-Type"])
    error_message = "allow_headers must default to Authorization and Content-Type"
  }
  assert {
    condition     = google_compute_url_map.public-service.path_matcher[0].default_route_action[0].cors_policy[0].allow_credentials == false
    error_message = "allow_credentials must default to false"
  }
  assert {
    condition     = length(google_compute_url_map.public-service.path_matcher[0].default_route_action[0].weighted_backend_services) == 0
    error_message = "default_route_action must name no weighted backends, or GCP rejects it alongside default_service"
  }
  assert {
    condition     = google_compute_backend_service.public-services["api.example.dev"].load_balancing_scheme == "EXTERNAL_MANAGED"
    error_message = "the per-service load_balancing_scheme must reach the backend service"
  }
}

run "cors_policy_on_a_classic_load_balancer_fails_the_plan" {
  command = plan
  variables {
    public-services = {
      "api.example.dev" = {
        name        = "api"
        cors_policy = { allow_origins = ["https://console.example.dev"] }
      }
    }
  }
  expect_failures = [google_compute_url_map.public-service]
}

run "cors_policy_without_origins_is_rejected" {
  command = plan
  variables {
    public-services = {
      "api.example.dev" = {
        name                  = "api"
        load_balancing_scheme = "EXTERNAL_MANAGED"
        cors_policy           = {}
      }
    }
    forwarding_rule_load_balancing = {
      load_balancing_scheme = "EXTERNAL_MANAGED"
    }
  }
  expect_failures = [var.public-services]
}

run "backend_scheme_must_match_the_forwarding_rule" {
  command = plan
  variables {
    public-services = {
      "api.example.dev" = {
        name                  = "api"
        load_balancing_scheme = "EXTERNAL_MANAGED"
      }
    }
  }
  expect_failures = [google_compute_backend_service.public-services]
}

run "migration_state_permits_a_temporary_scheme_mismatch" {
  command = plan
  variables {
    public-services = {
      "api.example.dev" = {
        name                             = "api"
        load_balancing_scheme            = "EXTERNAL_MANAGED"
        external_managed_migration_state = "PREPARE"
      }
    }
  }
  assert {
    condition     = google_compute_backend_service.public-services["api.example.dev"].external_managed_migration_state == "PREPARE"
    error_message = "external_managed_migration_state must reach the backend service"
  }
}
