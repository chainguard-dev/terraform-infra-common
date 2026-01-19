variable "name" {
  description = "Name of the workqueue"
  type        = string
}

variable "max_retry" {
  description = "The maximum number of retry attempts before a task is moved to the dead letter queue"
  type        = number
  default     = 100
}

variable "concurrent_work" {
  description = "The amount of concurrent work to dispatch at a given time"
  type        = number
}

variable "shards" {
  description = "Number of workqueue shards. When > 1, dashboard shows per-shard metrics."
  type        = number
  default     = 1
}

variable "labels" {
  description = "Additional labels to apply to the dashboard"
  type        = map(string)
  default     = {}
}

variable "alerts" {
  description = "A mapping from alerting policy names to the alert ids to add to the dashboard"
  type        = map(string)
  default     = {}
}
