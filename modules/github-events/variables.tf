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

variable "service-ingress" {
  type        = string
  description = <<EOD
Which type of ingress traffic to accept for the service (see regional-go-service). Valid values are:

- INGRESS_TRAFFIC_ALL accepts all traffic, enabling the public .run.app URL for the service
- INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER accepts traffic only from a load balancer
EOD
}

variable "webhook-secret" {
  type        = string
  description = "The secret to use as the webhook secret. If empty, the value must be populated manually."
  default     = ""
}
