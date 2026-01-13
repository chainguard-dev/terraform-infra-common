# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    ko     = { source = "ko-build/ko" }
    cosign = { source = "chainguard-dev/cosign" }
    aws    = { source = "hashicorp/aws" }
  }
}

// Get current AWS account and region for ECR repository URL
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

// Create ECR repository if requested
resource "aws_ecr_repository" "this" {
  count = var.create_ecr_repository ? 1 : 0

  name                 = coalesce(var.ecr_repository_name, var.name)
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  force_delete = var.ecr_force_delete

  tags = merge(var.tags, {
    Name             = coalesce(var.ecr_repository_name, var.name)
    terraform-module = "aws-apprunner-regional-go-service"
    team             = var.team
    product          = var.product
  })
}

locals {
  // Determine the ECR repository URL to use
  ecr_repository_url = var.create_ecr_repository ? aws_ecr_repository.this[0].repository_url : "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com/${coalesce(var.ecr_repository_name, var.name)}"

  // Determine which role ARNs to use (created or provided)
  service_role_arn  = var.create_service_role ? aws_iam_role.apprunner_service[0].arn : var.service_role_arn
  instance_role_arn = var.create_instance_role ? aws_iam_role.apprunner_instance[0].arn : var.instance_role_arn

  default_tags = {
    Name             = var.name
    terraform-module = "aws-apprunner-regional-go-service"
    team             = var.team
    product          = var.product
  }
}

// Create IAM Service Role - Used by App Runner to access ECR and write logs
resource "aws_iam_role" "apprunner_service" {
  count = var.create_service_role ? 1 : 0

  name = "${var.name}-apprunner-service-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "build.apprunner.amazonaws.com"
      }
    }]
  })

  tags = merge(var.tags, local.default_tags)
}

// Attach the managed policy for ECR access to the service role
resource "aws_iam_role_policy_attachment" "apprunner_service_ecr" {
  count = var.create_service_role ? 1 : 0

  role       = aws_iam_role.apprunner_service[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSAppRunnerServicePolicyForECRAccess"
}

// Create IAM Instance Role - Used by the running containers
resource "aws_iam_role" "apprunner_instance" {
  count = var.create_instance_role ? 1 : 0

  name = "${var.name}-apprunner-instance-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "tasks.apprunner.amazonaws.com"
      }
    }]
  })

  tags = merge(var.tags, local.default_tags)
}

// Attach X-Ray policy to instance role when observability is enabled
resource "aws_iam_role_policy_attachment" "apprunner_instance_xray" {
  count = var.create_instance_role && var.observability_enabled ? 1 : 0

  role       = aws_iam_role.apprunner_instance[0].name
  policy_arn = "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess"
}

// Build the application image from source using ko
resource "ko_build" "this" {
  repo        = coalesce(var.container.source.repo, local.ecr_repository_url)
  base_image  = var.container.source.base_image
  working_dir = var.container.source.working_dir
  importpath  = var.container.source.importpath
  env         = var.container.source.env
}

// Sign the built image with cosign
resource "cosign_sign" "this" {
  image    = ko_build.this.image_ref
  conflict = "REPLACE"
}

// Create App Runner service
resource "aws_apprunner_service" "this" {
  service_name = var.name

  source_configuration {
    image_repository {
      image_identifier      = cosign_sign.this.signed_ref
      image_repository_type = var.image_repository_type

      image_configuration {
        port = tostring(var.container.port)

        # Start command (args)
        start_command = length(var.container.args) > 0 ? join(" ", var.container.args) : null

        # Environment variables
        runtime_environment_variables = {
          for env in var.container.env : env.name => env.value if env.value != null
        }

        # Secrets from Secrets Manager or SSM Parameter Store
        runtime_environment_secrets = {
          for secret in var.container.secrets : secret.name => secret.value
        }
      }
    }

    auto_deployments_enabled = var.auto_deployments_enabled

    authentication_configuration {
      access_role_arn = local.service_role_arn
    }
  }

  instance_configuration {
    cpu               = tostring(var.cpu)
    memory            = tostring(var.memory)
    instance_role_arn = local.instance_role_arn
  }

  dynamic "health_check_configuration" {
    for_each = var.container.health_check != null ? [var.container.health_check] : []
    content {
      protocol            = health_check_configuration.value.protocol
      path                = health_check_configuration.value.protocol == "HTTP" ? health_check_configuration.value.path : null
      interval            = health_check_configuration.value.interval
      timeout             = health_check_configuration.value.timeout
      healthy_threshold   = health_check_configuration.value.healthy_threshold
      unhealthy_threshold = health_check_configuration.value.unhealthy_threshold
    }
  }

  dynamic "network_configuration" {
    for_each = var.egress == "VPC" || var.ingress == "PRIVATE" ? [1] : []
    content {
      ingress_configuration {
        is_publicly_accessible = var.ingress == "PUBLIC"
      }

      egress_configuration {
        egress_type       = var.egress == "VPC" ? "VPC" : "DEFAULT"
        vpc_connector_arn = var.egress == "VPC" ? var.vpc_connector_arn : null
      }
    }
  }

  auto_scaling_configuration_arn = aws_apprunner_auto_scaling_configuration_version.this.arn

  observability_configuration {
    observability_enabled           = var.observability_enabled
    observability_configuration_arn = var.observability_enabled ? aws_apprunner_observability_configuration.this[0].arn : null
  }

  tags = merge(var.tags, local.default_tags)
}

// Create auto-scaling configuration
resource "aws_apprunner_auto_scaling_configuration_version" "this" {
  auto_scaling_configuration_name = "${var.name}-autoscaling"

  min_size        = var.autoscaling.min_instances
  max_size        = var.autoscaling.max_instances
  max_concurrency = var.autoscaling.max_concurrency

  tags = merge(var.tags, local.default_tags)
}

// Create observability configuration for X-Ray tracing
resource "aws_apprunner_observability_configuration" "this" {
  count = var.observability_enabled ? 1 : 0

  observability_configuration_name = "${var.name}-observability"

  trace_configuration {
    vendor = "AWSXRAY"
  }

  tags = merge(var.tags, local.default_tags)
}
