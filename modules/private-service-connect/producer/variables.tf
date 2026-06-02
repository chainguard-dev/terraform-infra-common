# Copyright 2026 Chainguard, Inc.
# SPDX-License-Identifier: Apache-2.0

variable "project" {
  description = "The project ID in which to create the producer-side PSC resources."
  type        = string
}

variable "region" {
  description = "The region in which the Cloud Run service, internal ALB, and service attachment live."
  type        = string
}

variable "network" {
  description = "Self-link of the VPC network hosting the internal ALB frontend."
  type        = string
}

variable "subnetwork" {
  description = "Self-link of the subnetwork in which the internal ALB VIP is allocated."
  type        = string
}

variable "proxy_only_subnet" {
  description = "Self-link of the caller-created REGIONAL_MANAGED_PROXY subnet for this region. The module does not create this subnet; it is referenced only to order the ALB forwarding rule after the proxy-only subnet exists."
  type        = string
}

variable "psc_nat_subnets" {
  description = "List of self-links of caller-created PRIVATE_SERVICE_CONNECT NAT subnets used by the service attachment. The module does not create these subnets."
  type        = list(string)
}

variable "cloud_run_service_name" {
  description = "Name of the existing regional internal Cloud Run service to front with the internal ALB."
  type        = string
}

variable "consumer_accept_projects" {
  description = "List of consumer project IDs or numbers explicitly accepted by the service attachment (ACCEPT_MANUAL)."
  type        = list(string)
}

variable "connection_limit" {
  description = "Per-consumer connection limit applied to each entry in consumer_accept_projects."
  type        = number
  default     = 10
}

variable "name" {
  description = "Resource name prefix for the producer-side resources."
  type        = string
}

variable "labels" {
  description = "Labels to apply to resources that support them."
  type        = map(string)
  default     = {}
}
