# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

output "service_name" {
  description = "App Runner service name"
  value       = aws_apprunner_service.this.service_name
}

output "service_id" {
  description = "App Runner service ID"
  value       = aws_apprunner_service.this.service_id
}

output "service_arn" {
  description = "App Runner service ARN"
  value       = aws_apprunner_service.this.arn
}

output "service_url" {
  description = "App Runner service URL"
  value       = aws_apprunner_service.this.service_url
}

output "service_status" {
  description = "App Runner service status"
  value       = aws_apprunner_service.this.status
}

output "built_image" {
  description = "Built and signed container image reference"
  value       = cosign_sign.this.signed_ref
}

output "autoscaling_config_arn" {
  description = "Auto-scaling configuration ARN"
  value       = aws_apprunner_auto_scaling_configuration_version.this.arn
}

output "ecr_repository_url" {
  description = "ECR repository URL (if created by module)"
  value       = var.create_ecr_repository ? aws_ecr_repository.this[0].repository_url : null
}

output "ecr_repository_arn" {
  description = "ECR repository ARN (if created by module)"
  value       = var.create_ecr_repository ? aws_ecr_repository.this[0].arn : null
}

output "service_role_arn" {
  description = "IAM service role ARN used by App Runner for ECR access and logs"
  value       = local.service_role_arn
}

output "service_role_name" {
  description = "IAM service role name (if created by module)"
  value       = var.create_service_role ? aws_iam_role.apprunner_service[0].name : null
}

output "instance_role_arn" {
  description = "IAM instance role ARN used by the running containers"
  value       = local.instance_role_arn
}

output "instance_role_name" {
  description = "IAM instance role name (if created by module)"
  value       = var.create_instance_role ? aws_iam_role.apprunner_instance[0].name : null
}
