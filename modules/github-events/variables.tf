variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork. The bucket must be in one of these regions."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}

variable "ingress" {
  description = "An object holding the name of the ingress service, which can be used to authorize callers to publish cloud events."
  type = object({
    name = string
  })
}

variable "max_delivery_attempts" {
  description = "The maximum number of delivery attempts for any event."
  type        = number
  default     = 5
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "secret_version_adder" {
  type        = string
  description = "The user allowed to populate new webhook secret versions."
}

variable "additional_webhook_secrets" {
  type = map(object({
    secret  = string
    version = string
  }))
  description = <<EOD
Additional secrets to be used by the service.

- key: Local identifier for the secret. This will be prefixed with WEBHOOK_SECRET_ in the service's environment vars.
- secret: The name of the secret in Cloud Secret Manager. Format: {secretName} if the secret is in the same project. projects/{project}/secrets/{secretName} if the secret is in a different project.
- version: The version of the secret to use. Can be a number or 'latest'.

See https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service#nested_env for related documentation.
EOD
  default     = {}
}

variable "service-ingress" {
  type        = string
  description = <<EOD
Which type of ingress traffic to accept for the service (see regional-go-service). Valid values are:

- INGRESS_TRAFFIC_ALL accepts all traffic, enabling the public .run.app URL for the service
- INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER accepts traffic only from a load balancer
EOD
  default     = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}



variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
}

variable "github_organizations" {
  description = "csv string of GitHub organizations to allow."
  type        = string
  default     = ""
}

variable "requested_only_webhook_id" {
  description = "If set, the csv IDs of the webhooks that should only receive check requested events."
  type        = string
  default     = ""
}

variable "webhook_id" {
  description = "If set, the csv IDs of the webhooks that the trampoline should listen to."
  type        = string
  default     = ""
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
