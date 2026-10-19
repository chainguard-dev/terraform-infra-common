// Copyright 2026 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 8.7.0"
    }
  }
}

// Resolve each group's email to its Cloud Identity name (groups/<id>).
data "google_cloud_identity_group_lookup" "this" {
  for_each = var.groups

  group_key {
    id = each.key
  }
}

// IAM grants a group's roles to the members of its nested groups too, so list
// those members as well.
data "google_cloud_identity_group_transitive_memberships" "this" {
  for_each = var.groups

  group = data.google_cloud_identity_group_lookup.this[each.key].name
}

locals {
  # A nested group appears under its own address, which never acts, next to
  # its members. Any other member keyed by an email can act under it; devices
  # and shared drives have no email.
  emails = {
    for group, listing in data.google_cloud_identity_group_transitive_memberships.this : group => sort(distinct([
      for m in listing.memberships : lower(m.preferred_member_key[0].id)
      if !startswith(m.member, "groups/") && strcontains(try(m.preferred_member_key[0].id, ""), "@")
    ]))
  }
}
