variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A pub/sub topic and ingress service (publishing to the respective topic) will be created in each region, with the ingress service configured to egress all traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "limits" {
  description = "Resource limits for the regional go service."
  type = object({
    cpu    = string
    memory = string
  })
  default = null
}
