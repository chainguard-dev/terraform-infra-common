/*
Copyright 2022 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

locals {
  uptime_check_name = local.use_gclb ? google_monitoring_uptime_check_config.global_uptime_check[0].display_name : google_monitoring_uptime_check_config.regional_uptime_check[0].display_name
  uptime_check_id   = local.use_gclb ? google_monitoring_uptime_check_config.global_uptime_check[0].uptime_check_id : google_monitoring_uptime_check_config.regional_uptime_check[0].uptime_check_id
}

output "uptime_check" {
  value = local.uptime_check_id
}

output "uptime_check_name" {
  value = local.uptime_check_name
}

output "alert_id" {
  value = var.enable_alert ? google_monitoring_alert_policy.uptime_alert[0].id : ""
}
