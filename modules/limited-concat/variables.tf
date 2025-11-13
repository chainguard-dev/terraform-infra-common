variable "prefix" {
  description = "First part of the result, will be shortened if length(prefix)+length(suffix) > limit."
  type        = string
}

variable "suffix" {
  description = "Second part of the result, included in whole."
  type        = string
}

variable "limit" {
  description = "Maximum length of the resulting concatenation."
  type        = number

  validation {
    condition     = var.limit >= length(var.suffix)
    error_message = "limit cannot be less than the length of suffix."
  }
}
