variable "project_id" { type = string }
variable "topic_prefix" { type = string }

variable "alert_policies" {
  type = map(object({
    id = string
  }))
  default = {}
}
