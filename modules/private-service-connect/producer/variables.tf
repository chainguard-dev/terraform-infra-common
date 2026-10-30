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
  description = "Consumer project IDs or numbers accepted by the service attachment (ACCEPT_MANUAL). A project entry accepts endpoints from every VPC network in that project; use consumer_accept_networks to accept only specific networks. Set this or consumer_accept_networks, not both."
  type        = list(string)
  default     = []
}

variable "consumer_accept_networks" {
  description = "Consumer VPC networks accepted by the service attachment (ACCEPT_MANUAL), as self-links or projects/<project>/global/networks/<name> paths. Endpoints in other networks of the same project stay PENDING. Set this or consumer_accept_projects, not both: Google rejects an accept list that mixes consumer types."
  type        = list(string)
  default     = []

  validation {
    condition     = length(var.consumer_accept_networks) == 0 || length(var.consumer_accept_projects) == 0
    error_message = "Set consumer_accept_projects or consumer_accept_networks, not both: Google rejects an accept list that mixes consumer types."
  }

  validation {
    condition     = alltrue([for n in var.consumer_accept_networks : can(regex("^(https://www\\.googleapis\\.com/compute/v1/)?projects/[^/]+/global/networks/[^/]+$", n))])
    error_message = "Each consumer_accept_networks entry must be a network self-link or a projects/<project>/global/networks/<name> path."
  }
}

variable "connection_limit" {
  description = "Per-consumer connection limit applied to each accept-list entry."
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

variable "allow_global_access" {
  description = "Allow clients in any region to reach the internal ALB forwarding rule. Required before a PSC consumer endpoint targeting this service attachment can set allow_psc_global_access = true; without it the consumer apply fails with \"the producer service does not support consumer global access\". Leave false when all consumers are in the producer's region."
  type        = bool
  default     = false
}

variable "enable_logging" {
  description = "Enable access logging on the internal ALB backend service (L7 request logs -> Cloud Logging at full sampling; reaches the SIEM via the org all-load-balancer sink). Opt-in; default off so existing producers are unaffected."
  type        = bool
  default     = false
}

variable "tls_certificates" {
  description = <<-EOT
    Certificate Manager regional certificate ids
    (projects/<project>/locations/<region>/certificates/<name>) for the
    frontend to serve. Empty (the default) keeps the plain-HTTP frontend on
    :80. Non-empty switches the frontend to a target HTTPS proxy on :443,
    which replaces the forwarding rule and so recreates the service
    attachment (closing every connected consumer endpoint for good: each
    consumer must then recreate its endpoint, see the README; the attachment
    name is unchanged). Certificates must already be ACTIVE:
    consumers dial the certificate's hostname and resolve it to their own
    PSC endpoint IP (a private DNS zone in the consumer VPC), so a regional
    Google-managed certificate with DNS authorization for a real domain
    verifies against public roots with no trust distribution. Attach it on
    the provision after the one that issued it.
  EOT
  type        = list(string)
  default     = []
}

variable "lb_firewall" {
  description = <<-EOT
    Close the internal ALB's forwarding rule to everything but PSC consumers.
    Envoy-based load balancers admit every client in the network, peered
    networks, and hybrid links when no Cloud NGFW rule targets them, so
    without this the VIP is reachable from the whole producer VPC alongside
    the PSC path. When set, the module adds two rules to an existing regional
    network firewall policy that the caller owns and has associated with
    `network` in `region`: an allow from the PSC NAT subnets (plus
    allowed_source_ranges) on the frontend port at `priority`, and a logged
    deny for every other source at `priority + 1`. Both target only this
    forwarding rule, so producers sharing a network and policy coexist as
    long as their priorities differ. `project` is the policy's project,
    defaulting to `project`; set it to the host project when `network` is a
    Shared VPC. Null (the default) adds no rules.
  EOT
  type = object({
    policy                = string
    priority              = number
    project               = optional(string)
    allowed_source_ranges = optional(list(string), [])
  })
  default = null

  validation {
    condition     = var.lb_firewall == null || (var.lb_firewall.priority >= 0 && var.lb_firewall.priority < 2147483547 && floor(var.lb_firewall.priority) == var.lb_firewall.priority)
    error_message = "lb_firewall.priority must be an integer from 0 to 2147483546, leaving priority + 1 for the deny rule below Google's reserved range."
  }
}
