variable "project_id" {
  type = string
}

variable "name" {
  description = "The name to give the service account."
  type        = string
}

variable "wif-pool" {
  description = "The name of the Workload Identity Federation pool."
  type        = string
}

variable "repository" {
  description = "The name of the repository to allow to assume this identity."
  type        = string
}

variable "refspec" {
  description = "The refspec to allow to federate with this identity."
  type        = string
  validation {
    condition     = var.refspec == "*" || var.refspec == "pull_request" || var.refspec == "pull_request_target" || var.refspec == "version_tags" || startswith(var.refspec, "refs/")
    error_message = "Expected '*', 'pull_request', 'pull_request_target', 'version_tags' or a refspec of the form 'refs/heads/main', but got '${var.refspec}'."
  }
}
variable "audit_refspec" {
  description = "Deprecated: has no effect. It only fed a subject-regex matcher that no resource ever consumed; federation is matched via the provider's attribute mappings, which have no audited-wildcard form. Retained so existing callers do not break; do not set it expecting tightened scoping."
  type        = string
  default     = ""
}

variable "workflow_ref" {
  description = "The workflow to allow to federate with this identity (e.g. .github/workflows/deploy.yaml)."
  type        = string
  validation {
    condition     = var.workflow_ref == "*" || startswith(var.workflow_ref, ".github/workflows/")
    error_message = "Expected '*' or a path of the form '.github/workflows/foo.yaml', but got '${var.workflow_ref}'."
  }
}

variable "audit_workflow_ref" {
  description = "Deprecated: has no effect, for the same reason as audit_refspec."
  type        = string
  default     = ""
}

variable "notification_channels" {
  description = "The list of notification channels to alert when the service account is misused."
  type        = list(string)
}
