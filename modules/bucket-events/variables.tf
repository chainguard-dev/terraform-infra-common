variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "bucket" {
  description = "The name of the bucket to watch for events."
  type        = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A pub/sub topic and ingress service (publishing to the respective topic) will be created in each region, with the ingress service configured to egress all traffic via the specified subnetwork."
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

variable "filter" {
  description = "A Knative Trigger-style filter over the cloud event attributes."
  type        = map(string)
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
