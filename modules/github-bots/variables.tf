variable "extra_filter" {
  type        = map(string)
  default     = {}
  description = "Optional additional filters to include."
}

variable "extra_filter_prefix" {
  type        = map(string)
  default     = {}
  description = "Optional additional prefixes for filtering events."
}

variable "extra_filter_has_attributes" {
  type        = list(string)
  default     = []
  description = "Optional additional attributes to check for presence."
}

variable "extra_filter_not_has_attributes" {
  type        = list(string)
  default     = []
  description = "Optional additional prefixes to check for presence."
}

variable "enable_profiler" {
  type        = bool
  default     = false
  description = "Enable cloud profiler."
}
