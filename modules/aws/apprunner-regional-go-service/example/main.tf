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
  description = "AWS region to deploy the service"
  type        = string
  default     = "us-east-1"
}

variable "observability_enabled" {
  description = "Enable AWS X-Ray tracing for observability"
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# Providers and data sources
# -----------------------------------------------------------------------------


# Configure AWS provider
provider "aws" {
  region = var.region
}

# Configure ko provider for building Go containers
provider "ko" {}

# Get current AWS account ID for resource ARNs
data "aws_caller_identity" "current" {}

# -----------------------------------------------------------------------------
# Deploy the Go service using the module
# -----------------------------------------------------------------------------

module "my_go_service" {
  source = "../" # Points to the aws/apprunner-regional-go-service module

  name    = "my-go-service"
  team    = "platform"
  product = "my-service"

  # IAM roles are created automatically by the module
  # To use custom roles, set create_service_role = false and create_instance_role = false
  # and provide service_role_arn and instance_role_arn

  # VPC connector (optional)
  # vpc_connector_arn = aws_apprunner_vpc_connector.this.arn

  # ECR Repository (module will create it automatically)
  create_ecr_repository = true
  ecr_force_delete      = false # Set to true to allow deletion even with images

  # Container configuration
  container = {
    source = {
      working_dir = "${path.module}/app"
      importpath  = "github.com/chainguard-dev/mono/terraform/public-modules/modules/aws/apprunner-regional-go-service/example/app"
      # repo is optional - module will use the created ECR repository automatically
    }

    port = 8080

    env = [
      {
        name  = "PORT"
        value = "8080"
      },
      {
        name  = "ENVIRONMENT"
        value = "production"
      },
      {
        name  = "REGION"
        value = var.region
      }
    ]

    # Uncomment to add secrets from Secrets Manager or SSM Parameter Store
    # secrets = [
    #   {
    #     name  = "DATABASE_URL"
    #     value = aws_secretsmanager_secret.database_url.arn
    #   },
    #   {
    #     name  = "API_KEY"
    #     value = aws_ssm_parameter.api_key.arn
    #   }
    # ]

    health_check = {
      protocol            = "HTTP"
      path                = "/health"
      interval            = 10
      timeout             = 5
      healthy_threshold   = 1
      unhealthy_threshold = 3
    }
  }

  cpu    = 1024
  memory = 2048

  autoscaling = {
    min_instances   = 2
    max_instances   = 10
    max_concurrency = 100
  }

  ingress                  = "PUBLIC"
  egress                   = "DEFAULT"
  image_repository_type    = "ECR"
  auto_deployments_enabled = true
  observability_enabled    = var.observability_enabled

  tags = {
    Environment = "production"
    ManagedBy   = "terraform"
  }
}

# -----------------------------------------------------------------------------
# Optional: Add application-specific IAM permissions
# -----------------------------------------------------------------------------
# The module creates the base IAM roles automatically.
# Use these resources to add custom permissions for your application's AWS access.

# Example: Grant secrets access to the instance role
# Uncomment if you uncomment the secrets resources below
# resource "aws_iam_role_policy" "app_secrets" {
#   name = "secrets-access"
#   role = module.my_go_service.instance_role_name
#
#   policy = jsonencode({
#     Version = "2012-10-17"
#     Statement = [
#       {
#         Effect = "Allow"
#         Action = [
#           "secretsmanager:GetSecretValue",
#           "ssm:GetParameter",
#           "ssm:GetParameters"
#         ]
#         Resource = [
#           "arn:aws:secretsmanager:*:${data.aws_caller_identity.current.account_id}:secret:my-service-*",
#           "arn:aws:ssm:*:${data.aws_caller_identity.current.account_id}:parameter/my-service/*"
#         ]
#       }
#     ]
#   })
# }

# Example: Grant your app access to AWS resources (S3, DynamoDB, etc.)
# resource "aws_iam_role_policy" "app_permissions" {
#   name = "app-permissions"
#   role = module.my_go_service.instance_role_name
#
#   policy = jsonencode({
#     Version = "2012-10-17"
#     Statement = [
#       {
#         Effect = "Allow"
#         Action = [
#           "s3:GetObject",
#           "s3:PutObject"
#         ]
#         Resource = [
#           "arn:aws:s3:::my-app-bucket/*"
#         ]
#       },
#       {
#         Effect = "Allow"
#         Action = [
#           "dynamodb:GetItem",
#           "dynamodb:PutItem",
#           "dynamodb:Query"
#         ]
#         Resource = [
#           "arn:aws:dynamodb:*:${data.aws_caller_identity.current.account_id}:table/my-app-table"
#         ]
#       }
#     ]
#   })
# }

# -----------------------------------------------------------------------------
# Optional: Create secrets for the application
# -----------------------------------------------------------------------------
# Uncomment these resources to add secrets to your application
# resource "aws_secretsmanager_secret" "database_url" {
#   name        = "my-service-database-url"
#   description = "Database connection URL for my-service"
#
#   tags = {
#     team    = "platform"
#     product = "my-service"
#   }
# }
#
# resource "aws_secretsmanager_secret_version" "database_url" {
#   secret_id     = aws_secretsmanager_secret.database_url.id
#   secret_string = "postgresql://user:pass@db.example.com:5432/mydb"
# }
#
# resource "aws_ssm_parameter" "api_key" {
#   name        = "/my-service/api-key"
#   description = "API key for external service"
#   type        = "SecureString"
#   value       = "your-api-key-here"
#
#   tags = {
#     team    = "platform"
#     product = "my-service"
#   }
# }

# -----------------------------------------------------------------------------
# Optional: VPC Connector (only if you need private resource access)
# -----------------------------------------------------------------------------

# Uncomment if you need VPC access for private resources like RDS
# data "aws_vpc" "main" {
#   filter {
#     name   = "tag:Name"
#     values = ["main-vpc"]
#   }
# }

# data "aws_subnets" "private" {
#   filter {
#     name   = "vpc-id"
#     values = [data.aws_vpc.main.id]
#   }
#   filter {
#     name   = "tag:Type"
#     values = ["private"]
#   }
# }

# resource "aws_security_group" "apprunner_vpc_connector" {
#   name        = "my-service-apprunner-vpc-connector"
#   description = "Security group for App Runner VPC connector"
#   vpc_id      = data.aws_vpc.main.id

#   egress {
#     from_port   = 0
#     to_port     = 0
#     protocol    = "-1"
#     cidr_blocks = ["0.0.0.0/0"]
#   }

#   tags = {
#     Name    = "my-service-apprunner-vpc-connector"
#     team    = "platform"
#     product = "my-service"
#   }
# }

# resource "aws_apprunner_vpc_connector" "this" {
#   vpc_connector_name = "my-service-vpc-connector"
#   subnets            = data.aws_subnets.private.ids
#   security_groups    = [aws_security_group.apprunner_vpc_connector.id]

#   tags = {
#     team    = "platform"
#     product = "my-service"
#   }
# }

# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------

output "service_url" {
  description = "App Runner service URL"
  value       = module.my_go_service.service_url
}

output "service_arn" {
  description = "App Runner service ARN"
  value       = module.my_go_service.service_arn
}

output "service_status" {
  description = "App Runner service status"
  value       = module.my_go_service.service_status
}

output "built_image" {
  description = "Built and signed container image"
  value       = module.my_go_service.built_image
}

output "region" {
  description = "AWS region where the service is deployed"
  value       = var.region
}

output "ecr_repository_url" {
  description = "ECR repository URL"
  value       = module.my_go_service.ecr_repository_url
}
