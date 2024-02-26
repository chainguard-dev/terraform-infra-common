/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

locals {
  # GCLB is expensive, so we only provision one when we have to put multiple
  # Cloud Run locations behind one.
  use_gclb = length(var.regions) > 1
}

module "gclb" {
  source = "../serverless-gclb"

  count = local.use_gclb ? 1 : 0

  name            = var.name
  project_id      = var.project_id
  regions         = keys(var.regions)
  serving_regions = keys(var.regions)
  dns_zone        = var.dns_zone

  public-services = {
    "${var.name}-prober.${var.domain}" : {
      name = local.service_name
    }
  }
}
