variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "service_name" {
  description = "Name of service to setup SLO for."
  type        = string
}

variable "service_type" {
  description = "Type of service to setup SLO for."
  type        = string
  default     = "CLOUD_RUN"
}

variable "regions" {
  description = "A list of regions that the cloudrun service is deployed in."
  type        = list(string)
}

variable "slo" {
  description = "Configuration for setting up SLO"
  type = object({
    enable          = optional(bool, false)
    enable_alerting = optional(bool, false)
    success = optional(object(
      {
        multi_region_goal = optional(number, 0.999)
        per_region_goal   = optional(number, 0.999)
      }
    ), {})
    monitor_gclb = optional(bool, false)
    alerting = optional(object(
      {
        threshold = optional(number, 10)
        duration  = optional(string, "0s")
        severity  = optional(string, null)
        # Single-region services get identical multi-region and per-region
        # alerts; burn rate over the 60m lookback is also independent of the
        # rolling period, so the 07d and 30d policies fire together. These
        # knobs let such services alert once per incident instead of 4x.
        #
        # ["07", "30"] here, in the validation below, and in
        # local.rolling_periods (main.tf) must stay in sync — validation
        # blocks cannot reference locals, so the duplication is forced.
        per_region_alerts = optional(bool, true)
        rolling_periods   = optional(list(string), ["07", "30"])
      }
    ), {})
  })
  default = {}

  validation {
    # Non-empty: an empty list (easy to produce with compact() or a filtered
    # [for ...]) would silently disable every burn-rate alert policy while
    # enable_alerting = true still reads as "alerting on".
    condition = length(var.slo.alerting.rolling_periods) > 0 && alltrue([
      for p in var.slo.alerting.rolling_periods : contains(["07", "30"], p)
    ])
    error_message = "slo.alerting.rolling_periods must be a non-empty subset of [\"07\", \"30\"]."
  }
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}
