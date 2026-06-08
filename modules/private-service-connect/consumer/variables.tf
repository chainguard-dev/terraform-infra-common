# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project" {
  description = "The project ID in which to create the consumer-side PSC endpoint."
  type        = string
}

variable "region" {
  description = "The region in which to create the PSC endpoint. Must match the producer's service attachment region."
  type        = string
}

variable "network" {
  description = "Self-link of the consumer VPC network hosting the PSC endpoint."
  type        = string
}

variable "subnetwork" {
  description = "Self-link / id of the consumer subnetwork in which the endpoint's internal IP is allocated."
  type        = string
}

variable "service_attachment" {
  description = "Self-link of the producer's PSC service attachment to target (the producer module's service_attachment_id output)."
  type        = string
}

variable "name" {
  description = "Resource name prefix for the consumer-side resources."
  type        = string
}

variable "address" {
  description = "Optional pre-reserved internal IP address (self-link / id) for the PSC endpoint. If empty, the module reserves an internal IP from the subnetwork."
  type        = string
  default     = ""
}

variable "allow_psc_global_access" {
  description = "Allow clients in any region to reach this PSC endpoint. Leave false when every caller runs in the endpoint's region; set true when callers run in other regions (e.g. a multi-region Cloud Run service dialing this single-region endpoint), otherwise their connections are silently dropped at the PSC layer."
  type        = bool
  default     = false
}
