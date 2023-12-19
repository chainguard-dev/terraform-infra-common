variable "project_id" { type = string }
variable "subscription_prefix" { type = string }

variable "alert_policies" {
  type = map(object({
    id = string
  }))
  default = {}
}
