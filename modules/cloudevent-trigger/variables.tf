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

variable "dead_letter_policy" {
  description = "The dead letter policy for the subscription."
  type = object({
    dead_letter_topic     = string
    max_delivery_attempts = number
  })
  default = {
    dead_letter_topic     = null
    max_delivery_attempts = 3
  }
}
