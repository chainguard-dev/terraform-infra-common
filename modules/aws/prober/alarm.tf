/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// CloudWatch alarm for uptime check failures
resource "aws_cloudwatch_metric_alarm" "uptime_alert" {
  count = var.enable_alert && var.cloudwatch_synthetics_enabled ? 1 : 0

  alarm_name          = "${var.name}-uptime-alert"
  comparison_operator = var.alarm_comparison_operator
  evaluation_periods  = var.alarm_evaluation_periods
  datapoints_to_alarm = var.alarm_datapoints_to_alarm
  metric_name         = "SuccessPercent"
  namespace           = "CloudWatchSynthetics"
  period              = tonumber(replace(var.uptime_alert_duration, "s", ""))
  statistic           = var.alarm_statistic
  threshold           = var.alarm_threshold
  alarm_description   = var.alert_description
  treat_missing_data  = var.alarm_treat_missing_data

  dimensions = {
    CanaryName = aws_synthetics_canary.uptime_check[0].name
  }

  alarm_actions = var.notification_channels
  ok_actions    = var.notification_channels

  tags = merge(var.tags, {
    Name    = "${var.name}-uptime-alert"
    team    = var.team
    product = var.product
  })
}
