# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Plan-only tests that pin the rendered Cloud Run vpc_access block.
#
# Cloud Run accepts a Serverless VPC Access connector OR direct VPC egress
# (network_interfaces), never both, so regional-connector has to suppress the
# network_interfaces block for the regions it covers. Getting this wrong fails
# silently in the worst way: on a Shared VPC, a region left on direct VPC egress
# with ALL_TRAFFIC is not NATed at all, so its public egress blackholes rather
# than erroring. These tests fail when that wiring regresses.
#
# Mock providers keep this fully offline: no credentials, no state.

mock_provider "google-beta" {}

mock_provider "google" {}

variables {
  project_id = "fixture-project"
  name       = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
    }
    "us-west1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-west1/subnetworks/fixture"
    }
  }
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

# With regional-connector unset, every region keeps direct VPC egress exactly as
# before and no connector is attached.
run "no_connector_renders_direct_vpc_egress" {
  command = plan

  assert {
    condition     = length(google_cloud_run_v2_service.this["us-central1"].template[0].vpc_access[0].network_interfaces) == 1
    error_message = "region without a connector must keep its network_interfaces block (direct VPC egress)"
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].template[0].vpc_access[0].connector == null
    error_message = "region without a connector must not render a connector"
  }
}

# A region present in regional-connector attaches the connector AND drops
# network_interfaces, which Cloud Run would otherwise reject as conflicting.
run "connector_replaces_direct_vpc_egress" {
  command = plan

  variables {
    regional-connector = {
      "us-central1" = "projects/host-project/locations/us-central1/connectors/cr-egress-us-central1"
    }
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].template[0].vpc_access[0].connector == "projects/host-project/locations/us-central1/connectors/cr-egress-us-central1"
    error_message = "region with a connector must render that connector"
  }

  assert {
    condition     = length(google_cloud_run_v2_service.this["us-central1"].template[0].vpc_access[0].network_interfaces) == 0
    error_message = "region with a connector must NOT also render network_interfaces; Cloud Run takes one or the other"
  }
}

# The map is per-region: covering one region must not disturb the others.
run "connector_is_scoped_to_its_region" {
  command = plan

  variables {
    regional-connector = {
      "us-central1" = "projects/host-project/locations/us-central1/connectors/cr-egress-us-central1"
    }
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-west1"].template[0].vpc_access[0].connector == null
    error_message = "a region absent from regional-connector must not inherit another region's connector"
  }

  assert {
    condition     = length(google_cloud_run_v2_service.this["us-west1"].template[0].vpc_access[0].network_interfaces) == 1
    error_message = "a region absent from regional-connector must keep direct VPC egress"
  }
}

# A bare connector name resolves in the service's own project, which silently is
# not where a Shared VPC host-project connector lives. Reject it at plan time.
run "bare_connector_name_is_rejected" {
  command = plan

  variables {
    regional-connector = {
      "us-central1" = "cr-egress-us-central1"
    }
  }

  expect_failures = [
    var.regional-connector,
  ]
}

# Unset network_tags must render the tags argument as null, the same as omitting
# it. Any other value would plan a new revision for every existing caller.
run "network_tags_default_is_null" {
  command = plan

  assert {
    condition     = google_cloud_run_v2_service.this["us-central1"].template[0].vpc_access[0].network_interfaces[0].tags == null
    error_message = "network_tags left unset must leave the network_interfaces tags null"
  }

  assert {
    condition     = alltrue([for tags in values(output.network_tags) : tags == null])
    error_message = "the network_tags output must report null for untagged regions"
  }
}

run "network_tags_render_on_direct_vpc_egress" {
  command = plan

  variables {
    network_tags = ["fixture-egress"]
  }

  assert {
    condition = alltrue([
      for region in ["us-central1", "us-west1"] :
      google_cloud_run_v2_service.this[region].template[0].vpc_access[0].network_interfaces[0].tags == tolist(["fixture-egress"])
    ])
    error_message = "every direct VPC egress region must carry network_tags"
  }

  assert {
    condition = alltrue([
      for region in ["us-central1", "us-west1"] : output.network_tags[region] == tolist(["fixture-egress"])
    ])
    error_message = "the network_tags output must report each region's rendered tags"
  }
}

# A connector region has no network_interfaces to carry tags. The other regions
# must still be tagged.
run "network_tags_with_connector_tag_only_direct_regions" {
  command = plan

  variables {
    network_tags = ["fixture-egress"]
    regional-connector = {
      "us-central1" = "projects/host-project/locations/us-central1/connectors/cr-egress-us-central1"
    }
  }

  assert {
    condition     = length(google_cloud_run_v2_service.this["us-central1"].template[0].vpc_access[0].network_interfaces) == 0
    error_message = "a connector region must not render network_interfaces, tagged or not"
  }

  assert {
    condition     = google_cloud_run_v2_service.this["us-west1"].template[0].vpc_access[0].network_interfaces[0].tags == tolist(["fixture-egress"])
    error_message = "a direct VPC egress region must keep network_tags when another region uses a connector"
  }

  assert {
    condition     = output.network_tags["us-central1"] == null && output.network_tags["us-west1"] == tolist(["fixture-egress"])
    error_message = "the network_tags output must report null for a connector region and the tags for the others"
  }
}

run "invalid_network_tag_is_rejected" {
  command = plan

  variables {
    network_tags = ["Fixture_Egress"]
  }

  expect_failures = [
    var.network_tags,
  ]
}
