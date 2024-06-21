variable "project_id" {
  type = string
}

variable "name" {
  description = "The name to give the secret."
  type        = string
}

variable "service_account" {
  description = "The email of the service account that will access the secret."
  type        = string
}

variable "region" {
  default     = "us-east4"
  description = "The region to run the job."
}

variable "invokers" {
  description = "List of user emails to grant invoker perimssions to invoke the job."
  type        = list(string)
  default     = []
}

variable "github_org" {
  description = "The GitHub organization for which the octo-sts token will be requested."
  type        = string
}

variable "github_repo" {
  description = "The GitHub repository for which the octo-sts token will be requested."
  type        = string
}

variable "octosts_policy" {
  description = "The name of the octo-sts policy for which to request a token."
  type        = string
}

variable "notification_channels" {
  description = "List of notification channels to alert."
  type        = list(string)
}
