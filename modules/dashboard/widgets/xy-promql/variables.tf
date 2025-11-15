/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

variable "title" {
  type        = string
  description = "Title of the XY chart widget"
}

variable "promql_query" {
  type        = string
  description = "PromQL query for the time series data"
}

variable "plot_type" {
  type        = string
  default     = "LINE"
  description = "Plot type for the chart (LINE, AREA, STACKED_AREA, STACKED_BAR)"
}

variable "thresholds" {
  type        = list(number)
  default     = []
  description = "List of threshold values to display on the chart"
}

variable "timeshift_duration" {
  type        = string
  default     = "0s"
  description = "Duration to timeshift the data"
}
