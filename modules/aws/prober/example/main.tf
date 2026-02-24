# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.0"
    }
    ko = {
      source  = "ko-build/ko"
      version = "0.0.19"
    }
    cosign = {
      source  = "chainguard-dev/cosign"
      version = "0.0.20"
    }
  }
}

# -----------------------------------------------------------------------------
# Variables
# -----------------------------------------------------------------------------

variable "region" {
  description = "AWS region to deploy the prober"
  type        = string
  default     = "us-east-1"
}

variable "target_url" {
  description = "URL for the prober to check"
  type        = string
  default     = "https://httpbin.org/status/200"
}

variable "enable_alerts" {
  description = "Enable CloudWatch alarms for the prober"
  type        = bool
  default     = true
}

variable "alert_email" {
  description = "Email address to receive alerts (optional)"
  type        = string
  default     = "cpanato@chainguard.dev"
}

# -----------------------------------------------------------------------------
# Providers and data sources
# -----------------------------------------------------------------------------

provider "aws" {
  region = var.region
}

provider "ko" {}

data "aws_caller_identity" "current" {}

# -----------------------------------------------------------------------------
# SNS Topic for Alerts (Optional)
# -----------------------------------------------------------------------------

resource "aws_sns_topic" "prober_alerts" {
  count = var.enable_alerts && var.alert_email != "" ? 1 : 0

  name = "prober-alerts"

  tags = {
    team    = "platform"
    product = "monitoring"
  }
}

resource "aws_sns_topic_subscription" "prober_alerts_email" {
  count = var.enable_alerts && var.alert_email != "" ? 1 : 0

  topic_arn = aws_sns_topic.prober_alerts[0].arn
  protocol  = "email"
  endpoint  = var.alert_email
}

# -----------------------------------------------------------------------------
# Deploy the Prober using the module
# -----------------------------------------------------------------------------

module "example_prober" {
  source = "../" # Points to the aws/prober module

  name    = "example-prober"
  team    = "platform"
  product = "monitoring"

  # Go application configuration
  importpath  = "github.com/chainguard-dev/mono/public/terraform-infra-common/modules/aws/prober/example/app"
  working_dir = "${path.module}/app"

  # Environment variables for the prober
  env = {
    TARGET_URL = var.target_url
    LOG_LEVEL  = "info"
  }

  # Optional: Add secrets from AWS Secrets Manager or SSM Parameter Store
  # secret_env = {
  #   API_KEY = aws_secretsmanager_secret.api_key.arn
  # }

  # Resource allocation
  cpu    = 1024 # 1 vCPU
  memory = 2048 # 2 GB

  # Scaling configuration
  scaling = {
    min_instances                    = 1
    max_instances                    = 5
    max_instance_request_concurrency = 100
  }

  # CloudWatch Synthetics configuration
  cloudwatch_synthetics_enabled = true
  canary_schedule               = "rate(5 minutes)"
  canary_runtime_version        = "syn-nodejs-puppeteer-13.0"

  # Alerting configuration
  enable_alert          = var.enable_alerts
  alert_description     = "The example prober is failing health checks. This may indicate that ${var.target_url} is unreachable."
  uptime_alert_duration = "300s" # Alert after 5 minutes of failures
  notification_channels = var.enable_alerts && var.alert_email != "" ? [aws_sns_topic.prober_alerts[0].arn] : []

  # Enable X-Ray tracing
  enable_profiler = true

  # Network configuration (use defaults)
  ingress = "PUBLIC"
  egress  = "DEFAULT"

  # Additional tags
  tags = {
    Environment = "example"
    ManagedBy   = "terraform"
    Example     = "true"
  }
}

# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------

output "prober_url" {
  description = "App Runner service URL for the prober"
  value       = module.example_prober.service_url
}

output "prober_name" {
  description = "App Runner service name"
  value       = module.example_prober.service_name
}

output "canary_name" {
  description = "CloudWatch Synthetics canary name"
  value       = module.example_prober.canary_name
}

output "alarm_arn" {
  description = "CloudWatch alarm ARN (if alerts are enabled)"
  value       = module.example_prober.alarm_arn
}

output "region" {
  description = "AWS region where the prober is deployed"
  value       = var.region
}

output "target_url" {
  description = "Target URL being monitored"
  value       = var.target_url
}

output "authorization_secret" {
  description = "The shared authorization secret (for testing only - keep secure!)"
  value       = module.example_prober.authorization_secret
  sensitive   = true
}
