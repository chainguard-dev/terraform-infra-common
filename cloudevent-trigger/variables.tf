variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "broker" {
  description = "The name of the pubsub topic we are using as a broker."
  type        = string
}

variable "filter" {
  description = "A Knative Trigger-style filter over the cloud event attributes."
  type        = map(string)
}

variable "private-service" {
  description = "The private cloud run service that is subscribing to these events."
  type = object({
    name   = string
    region = string
  })
}
