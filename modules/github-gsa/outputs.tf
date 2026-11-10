output "email" {
  value = google_service_account.this.email
}

output "wildcard_policy_violations" {
  description = "Wildcard-policy rules this binding breaks (unjustified-wildcard, glob-refspec, glob-or-ref-pinned-workflow_ref, ref-pinned-additional-workflow_ref, stale-justification). Empty when compliant."
  value       = local.wildcard_policy_violations
}
