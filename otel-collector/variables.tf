variable "project_id" {
  type = string
}

variable "service_account" {
  type        = string
  description = "The service account as which the collector will run."
}

variable "otel_collector_image" {
  type        = string
  default     = "cgr.dev/chainguard/opentelemetry-collector-contrib:latest"
  description = "The otel collector image to use as a base."
}

variable "otel_collector_policy" {
  type        = string
  default     = <<EOF
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: base-policy
spec:
  images:
    - glob: "**"
  authorities:
    - keyless:
        url: https://fulcio.sigstore.dev
        identities:
          - issuer: https://token.actions.githubusercontent.com
            subject: https://github.com/chainguard-images/images/.github/workflows/release.yaml@refs/heads/main
      ctlog:
        url: https://rekor.sigstore.dev
EOF
  description = "The otel collector image to use as a base."
}
