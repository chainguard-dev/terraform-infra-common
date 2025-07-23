variable "project_id" {
  type = string
}

variable "region" {
  description = "The region in which this Cloud Run service is based."
  type        = string
}

variable "name" {
  description = "The name of the Cloud Run service in this region."
  type        = string
}

variable "service-account" {
  description = "The email of the service account being authorized to invoke the private Cloud Run service."
  type        = string
}

