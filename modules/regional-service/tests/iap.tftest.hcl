# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

mock_provider "google-beta" {
  mock_resource "google_project_service_identity" {
    override_during = plan
    defaults = {
      email = "service-123456789@gcp-sa-iap.iam.gserviceaccount.com"
    }
  }
}

mock_provider "google" {
  mock_data "google_project" {
    defaults = {
      number = "123456789"
    }
  }
}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
  }
  ingress               = "INGRESS_TRAFFIC_ALL"
  service_account       = "fixture@fixture-project.iam.gserviceaccount.com"
  notification_channels = []
  team                  = "fixture"
  containers = {
    "main" = {
      image = "cgr.dev/chainguard/static:latest"
      ports = [{ container_port = 8080 }]
    }
  }
}

run "empty_members_preserve_public_access" {
  command = plan

  assert {
    condition = (
      !google_cloud_run_v2_service.this["us-central1"].iap_enabled &&
      length(google_project_service.iap) == 0 &&
      length(google_project_service_identity.iap) == 0 &&
      length(google_cloud_run_v2_service_iam_member.iap-invoker) == 0 &&
      length(google_iap_web_cloud_run_service_iam_binding.access) == 0 &&
      google_cloud_run_v2_service_iam_member.public-services-are-unauthenticated["us-central1"].member == "allUsers"
    )
    error_message = "An empty member set must preserve public access and create no IAP infrastructure."
  }
}

run "empty_members_preserve_authenticated_invocations" {
  command = plan
  variables {
    require_authenticated_invocations = true
  }
  assert {
    condition = (
      !google_cloud_run_v2_service.this["us-central1"].iap_enabled &&
      length(google_cloud_run_v2_service_iam_member.public-services-are-unauthenticated) == 0
    )
    error_message = "Existing IAM-only services must remain private without enabling IAP."
  }
}

run "members_enable_iap_in_every_region" {
  command = plan
  variables {
    iap_members = ["user:alice@example.com", "group:developers@example.com"]
    regions = {
      "us-central1" = {
        network = "projects/fixture-project/global/networks/fixture"
        subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
      }
      "us-east1" = {
        network = "projects/fixture-project/global/networks/fixture"
        subnet  = "projects/fixture-project/regions/us-east1/subnetworks/fixture"
      }
    }
  }
  assert {
    condition = (
      alltrue([for service in google_cloud_run_v2_service.this : service.iap_enabled]) &&
      length(google_cloud_run_v2_service_iam_member.public-services-are-unauthenticated) == 0
    )
    error_message = "IAP must protect every region and suppress public invocation without another flag."
  }
  assert {
    condition = (
      length(google_project_service.iap) == 1 &&
      google_project_service.iap[0].service == "iap.googleapis.com" &&
      !google_project_service.iap[0].disable_on_destroy &&
      length(google_project_service_identity.iap) == 1
    )
    error_message = "IAP setup must be project-scoped and must not disable the shared API on removal."
  }
  assert {
    condition = (
      toset(keys(google_cloud_run_v2_service_iam_member.iap-invoker)) == toset(keys(var.regions)) &&
      alltrue([for region, grant in google_cloud_run_v2_service_iam_member.iap-invoker :
        grant.project == var.project_id && grant.location == region &&
        grant.name == var.name && grant.role == "roles/run.invoker" &&
        grant.member == "serviceAccount:service-123456789@gcp-sa-iap.iam.gserviceaccount.com"
      ])
    )
    error_message = "Only the IAP service agent should receive the module's invoker grant in each region."
  }
  assert {
    condition = (
      toset(keys(google_iap_web_cloud_run_service_iam_binding.access)) == toset(keys(var.regions)) &&
      alltrue([for region, grant in google_iap_web_cloud_run_service_iam_binding.access :
        grant.project == var.project_id && grant.location == region &&
        grant.cloud_run_service_name == var.name &&
        grant.role == "roles/iap.httpsResourceAccessor" && grant.members == var.iap_members
      ])
    )
    error_message = "Each service's IAP policy must grant exactly the configured users and groups."
  }
}

run "iap_also_works_with_authenticated_invocations" {
  command = plan
  variables {
    iap_members                       = ["group:developers@example.com"]
    require_authenticated_invocations = true
  }
  assert {
    condition = (
      google_cloud_run_v2_service.this["us-central1"].iap_enabled &&
      length(google_cloud_run_v2_service_iam_member.iap-invoker) == 1 &&
      length(google_iap_web_cloud_run_service_iam_binding.access) == 1 &&
      length(google_cloud_run_v2_service_iam_member.public-services-are-unauthenticated) == 0
    )
    error_message = "The existing authenticated invocation flag must not suppress IAP grants."
  }
}
