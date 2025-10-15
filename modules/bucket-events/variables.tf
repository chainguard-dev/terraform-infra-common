variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "bucket" {
  description = "The name of the bucket to watch for events. The region where the bucket is located will be the region where the Pub/Sub topic and trampoline service will be created. The bucket must be in a region that is in the set of regions passed to the regions variable."
  type        = string
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

variable "gcs_event_types" {
  description = "The types of GCS events to watch for (https://cloud.google.com/storage/docs/pubsub-notifications#payload)."
  type        = list(string)
  default     = ["OBJECT_FINALIZE", "OBJECT_METADATA_UPDATE", "OBJECT_DELETE", "OBJECT_ARCHIVE"]
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}

variable "deletion_protection" {
  type        = bool
  description = "Whether to enable delete protection for the service."
  default     = true
}


variable "team" {
  description = "Team label to apply to resources (replaces deprecated 'squad')."
  type        = string
  default     = ""
}

variable "squad" {
  description = "DEPRECATED: Use 'team' instead. Squad label to apply to resources."
  type        = string
  default     = ""
}

variable "labels" {
  description = "Labels to apply to the bucket event resources."
  type        = map(string)
  default     = {}
}

variable "product" {
  description = "Product label to apply to the service."
  type        = string
  default     = "unknown"
}
