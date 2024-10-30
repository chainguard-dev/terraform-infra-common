variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A service will be created in each region configured to egress the specified traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "concurrent-work" {
  description = "The amount of concurrent work to dispatch at a given time."
  type        = number
}

variable "reconciler-service" {
  description = "The name of the reconciler service that the workqueue will dispatch work to."
  type = object({
    name = string
  })
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

// TODO(mattmoor): Allow passing labels through.
