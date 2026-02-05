# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "name" {
  type        = string
  description = "The name of the App Runner service"
}

variable "vpc_connector_arn" {
  description = "Optional VPC connector ARN for private resource access"
  type        = string
  default     = null
}

variable "create_service_role" {
  type        = bool
  description = "Whether to create the IAM service role for App Runner. If false, you must provide service_role_arn."
  default     = true
}

variable "service_role_arn" {
  type        = string
  description = "The ARN of the IAM role that App Runner will use (for ECR access and CloudWatch logs). Required if create_service_role is false."
  default     = ""
}

variable "create_instance_role" {
  type        = bool
  description = "Whether to create the IAM instance role for the running containers. If false, you must provide instance_role_arn."
  default     = true
}

variable "instance_role_arn" {
  type        = string
  description = "The ARN of the IAM role that the running service will assume. Required if create_instance_role is false."
  default     = ""
}

variable "container" {
  description = "The container configuration for the service. App Runner supports one container per service."
  type = object({
    source = object({
      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:9cef3c6a78264cb7e25923bf1bf7f39476dccbcc993af9f4ffeb191b77a7951e")
      working_dir = string
      importpath  = string
      repo        = optional(string) # Override the default ko repository for this container
      env         = optional(list(string), [])
    })
    args = optional(list(string), [])
    port = optional(number, 8080)
    env = optional(list(object({
      name  = string
      value = optional(string)
    })), [])
    # App Runner secrets from Secrets Manager or SSM Parameter Store
    secrets = optional(list(object({
      name  = string
      value = string # ARN of the secret
    })), [])
    health_check = optional(object({
      protocol            = optional(string, "TCP") # TCP or HTTP
      path                = optional(string, "/")   # For HTTP health checks
      interval            = optional(number, 5)     # Seconds between health checks
      timeout             = optional(number, 2)     # Seconds to wait for response
      healthy_threshold   = optional(number, 1)     # Consecutive successes needed
      unhealthy_threshold = optional(number, 5)     # Consecutive failures needed
    }))
  })
}

variable "cpu" {
  description = "The CPU units for the service. Valid values: 256 (0.25 vCPU), 512 (0.5 vCPU), 1024 (1 vCPU), 2048 (2 vCPU), 4096 (4 vCPU)"
  type        = number
  default     = 1024

  validation {
    condition     = contains([256, 512, 1024, 2048, 4096], var.cpu)
    error_message = "CPU must be one of: 256, 512, 1024, 2048, 4096"
  }
}

variable "memory" {
  description = "The memory in MB for the service. Valid values: 512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288"
  type        = number
  default     = 2048

  validation {
    condition     = contains([512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288], var.memory)
    error_message = "Memory must be one of: 512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288"
  }
}

variable "autoscaling" {
  description = "Autoscaling configuration for the service"
  type = object({
    min_instances   = optional(number, 1)
    max_instances   = optional(number, 25)
    max_concurrency = optional(number, 100) # Concurrent requests per instance
  })
  default = {
    min_instances   = 1
    max_instances   = 25
    max_concurrency = 100
  }
}

variable "ingress" {
  description = "Network ingress configuration. PUBLIC for internet access, PRIVATE for VPC only"
  type        = string
  default     = "PUBLIC"

  validation {
    condition     = contains(["PUBLIC", "PRIVATE"], var.ingress)
    error_message = "Ingress must be either PUBLIC or PRIVATE"
  }
}

variable "egress" {
  description = "Network egress configuration. DEFAULT for internet, VPC for private resources"
  type        = string
  default     = "DEFAULT"

  validation {
    condition     = contains(["DEFAULT", "VPC"], var.egress)
    error_message = "Egress must be either DEFAULT or VPC"
  }
}

variable "auto_deployments_enabled" {
  description = "Enable automatic deployments when new image pushed to ECR"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}

variable "team" {
  description = "Team label to apply to resources"
  type        = string
}

variable "product" {
  description = "Product label to apply to resources"
  type        = string
}

variable "observability_enabled" {
  description = "Enable AWS X-Ray tracing"
  type        = bool
  default     = true
}

variable "image_repository_type" {
  description = "The type of image repository. ECR for private AWS ECR, ECR_PUBLIC for public ECR"
  type        = string
  default     = "ECR"

  validation {
    condition     = contains(["ECR", "ECR_PUBLIC"], var.image_repository_type)
    error_message = "image_repository_type must be either ECR or ECR_PUBLIC"
  }
}

variable "create_ecr_repository" {
  description = "Whether to create an ECR repository for the container images. Set to false if using an existing repository."
  type        = bool
  default     = true
}

variable "ecr_repository_name" {
  description = "Name of the ECR repository. If not provided, defaults to the service name."
  type        = string
  default     = null
}

variable "ecr_force_delete" {
  description = "If true, will delete the ECR repository even if it contains images. Use with caution in production."
  type        = bool
  default     = false
}
