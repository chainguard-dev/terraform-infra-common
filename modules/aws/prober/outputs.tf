/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

output "service_name" {
  description = "App Runner service name"
  value       = module.this.service_name
}

output "service_url" {
  description = "App Runner service URL"
  value       = module.this.service_url
}

output "service_arn" {
  description = "App Runner service ARN"
  value       = module.this.service_arn
}

output "canary_name" {
  description = "CloudWatch Synthetics canary name (if enabled)"
  value       = var.cloudwatch_synthetics_enabled ? aws_synthetics_canary.uptime_check[0].name : null
}

output "canary_arn" {
  description = "CloudWatch Synthetics canary ARN (if enabled)"
  value       = var.cloudwatch_synthetics_enabled ? aws_synthetics_canary.uptime_check[0].arn : null
}

output "alarm_arn" {
  description = "CloudWatch alarm ARN (if enabled)"
  value       = var.enable_alert ? aws_cloudwatch_metric_alarm.uptime_alert[0].arn : null
}

output "authorization_secret" {
  description = "The shared secret used for authorization (sensitive)"
  value       = random_password.secret.result
  sensitive   = true
}

output "instance_role_arn" {
  description = "IAM instance role ARN used by the running containers"
  value       = module.this.instance_role_arn
}

output "instance_role_name" {
  description = "IAM instance role name (if created by module)"
  value       = module.this.instance_role_name
}
