# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

provider "google" {
  project = "workload-project"
}

variable "enable_ipv6" {
  type    = bool
  default = false
}

module "endpoint" {
  source = "../.."

  project_id  = "workload-project"
  name        = "legacy"
  dns_zone    = "existing-zone"
  team        = "fixture"
  enable_ipv6 = var.enable_ipv6
  public-services = {
    "legacy.example.dev" = {
      name = "receiver"
    }
  }
  buckets = {
    "downloads.example.dev" = {
      name        = "downloads"
      bucket_name = "fixture-download-objects"
    }
  }
}

output "backend_services" {
  value = module.endpoint.backend_services
}

output "frontend" {
  value = module.endpoint.frontend
}

output "backend_buckets" {
  value = module.endpoint.backend_buckets
}
