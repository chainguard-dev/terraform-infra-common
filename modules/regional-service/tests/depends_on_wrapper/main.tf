# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

# Calls regional-service behind a module-level depends_on on a resource that is
# being created, which defers any data source read inside the module to apply.

terraform {
  required_version = ">= 1.3"

  required_providers {
    google      = { source = "hashicorp/google" }
    google-beta = { source = "hashicorp/google-beta" }
  }
}

variable "project_number" {
  type    = string
  default = null
}

variable "resource_manager_tags" {
  type    = map(string)
  default = {}
}

resource "terraform_data" "dependency" {
  input = "fixture"
}

module "service" {
  source = "../../"

  project_id     = "fixture-project"
  project_number = var.project_number
  name           = "fixture"
  regions = {
    "us-central1" = {
      network = "projects/fixture-project/global/networks/fixture"
      subnet  = "projects/fixture-project/regions/us-central1/subnetworks/fixture"
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
  resource_manager_tags = var.resource_manager_tags

  depends_on = [terraform_data.dependency]
}

output "binding_parents" {
  value = module.service.tag_binding_parents
}
