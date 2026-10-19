// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

// Test fixture: calls the module behind a module-level depends_on on a
// resource that is being created, as callers do.

terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = ">= 8.7.0"
    }
    random = { source = "hashicorp/random" }
  }
}

provider "google" {}

provider "google-beta" {}

variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "broker" {
  type = string
}

variable "private-service" {
  type = object({
    name   = string
    region = string
  })
}

variable "notification_channels" {
  type = list(string)
}

variable "team" {
  type = string
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
  input = var.name
}

module "trigger" {
  source = "../.."

  depends_on = [terraform_data.dependency]

  project_id            = var.project_id
  name                  = var.name
  broker                = var.broker
  private-service       = var.private-service
  notification_channels = var.notification_channels
  team                  = var.team

  project_number        = var.project_number
  resource_manager_tags = var.resource_manager_tags
}

output "tag_binding_parents" {
  value = module.trigger.tag_binding_parents
}
