variable "project_id" {
  type = string
}

variable "name" {
  type = string
}

variable "regions" {
  description = "A map from region names to a network and subnetwork.  A pub/sub topic and ingress service (publishing to the respective topic) will be created in each region, with the ingress service configured to egress all traffic via the specified subnetwork."
  type = map(object({
    network = string
    subnet  = string
  }))
}

variable "ingress" {
  type        = string
  description = "The ingress mode for the service.  Must be one of INGRESS_TRAFFIC_ALL, INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER, or INGRESS_TRAFFIC_INTERNAL_ONLY."
  default     = "INGRESS_TRAFFIC_INTERNAL_ONLY"
}

variable "egress" {
  type        = string
  description = "The egress mode for the service.  Must be one of ALL_TRAFFIC, or PRIVATE_RANGES_ONLY. Egress traffic is routed through the regional VPC network from var.regions."
  default     = "ALL_TRAFFIC"
}

variable "service_account" {
  type        = string
  description = "The service account as which to run the service."
}

variable "containers" {
  description = "The containers to run in the service.  Each container will be run in each region."
  type = map(object({
    source = object({
      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc")
      base_policy = optional(string, <<EOF
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
)
      working_dir = string
      importpath  = string
    })
    args = optional(list(string), [])
    ports = optional(list(object({
      name           = optional(string, "http1")
      container_port = number
    })), [])
    resources = optional(object({
      limits = object({
        cpu    = string
        memory = string
      })
    }), null)
    env = optional(list(object({
      name  = string
      value = optional(string)
      value_source = optional(object({
        secret_key_ref = object({
          secret  = string
          version = string
        })
      }), null)
    })), [])
    regional-env = optional(list(object({
      name  = string
      value = map(string)
    })), [])
    volume_mounts = optional(list(object({
      name       = string
      mount_path = string
    })), [])
  }))
}

variable "volumes" {
  description = "The volumes to make available to the containers in the service for mounting."
  type = list(object({
    name = string
    empty_dir = optional(object({
      medium = optional(string, "MEMORY")
    }))
    secret = optional(object({
      secret = string
      items = list(object({
        version = string
        path    = string
      }))
    }))
  }))
  default = []
}
