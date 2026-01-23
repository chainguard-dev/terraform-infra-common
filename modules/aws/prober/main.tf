/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// Create a shared secret to have the uptime check pass to the
// App Runner app as an "Authorization" header to keep ~anyone
// from being able to use our prober endpoints to indirectly
// DoS our SaaS.
resource "random_password" "secret" {
  length           = 64
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

locals {
  service_name = "prb-${substr(var.name, 0, 45)}" // use a common prefix so that they group together.
}

module "this" {
  source = "../apprunner-regional-go-service"

  name    = local.service_name
  team    = var.team
  product = var.product

  // Different probers have different egress requirements.
  // For VPC access, users should provide vpc_connector_arn
  ingress           = var.ingress
  egress            = var.egress
  vpc_connector_arn = var.vpc_connector_arn

  // Allow external instance role to be provided
  create_instance_role = var.create_instance_role
  instance_role_arn    = var.instance_role_arn

  container = {
    source = {
      working_dir = var.working_dir
      importpath  = var.importpath
      base_image  = var.base_image
    }
    port = 8080
    env = concat([
      {
        // This is a shared secret with the uptime check, which must be
        // passed in an Authorization header for the probe to do work.
        name  = "AUTHORIZATION"
        value = random_password.secret.result
      }
      ],
      [for k, v in var.env : { name = k, value = v }],
    )
    secrets = [
      for k, v in var.secret_env : {
        name  = k
        value = v
      }
    ]
  }

  cpu    = var.cpu
  memory = var.memory

  autoscaling = {
    min_instances   = var.scaling.min_instances
    max_instances   = var.scaling.max_instances
    max_concurrency = var.scaling.max_instance_request_concurrency
  }

  observability_enabled = var.enable_profiler

  tags = var.tags
}
