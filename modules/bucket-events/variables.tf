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

variable "location" {
  description = "The location of the bucket."
  type        = string
}

locals {
  region = lookup({
    "us" : "us-central1",
    "eu" : "europe-west1",
    "asia" : "asia-east1",
  }, lower(var.location), lower(var.location))
}

variable "broker" {
  description = "The name of the pubsub topic we are using as a broker."
  type        = string
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
