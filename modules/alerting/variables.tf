variable "project_id" {
  type = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}

variable "oom_filter" {
  description = "additional filter to apply to oom alert policy"
  type        = string
}
