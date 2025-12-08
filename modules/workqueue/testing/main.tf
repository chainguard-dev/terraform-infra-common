terraform {
  required_providers {
    ko = {
      source = "ko-build/ko"
    }
  }
}

locals {
  dispatcher_batch_size = coalesce(var.batch-size, var.concurrent-work)
}

resource "kubernetes_manifest" "svc-acct" {
  manifest = {
    "apiVersion" : "v1",
    "kind" : "ServiceAccount",
    "metadata" : {
      "name" : var.name,
      "namespace" : var.namespace
    }
  }
}

resource "ko_build" "inmem" {
  base_image  = "chainguard/static:latest"
  importpath  = "./cmd/inmem"
  working_dir = "${path.module}/../"
}

resource "kubernetes_manifest" "inmem-ksvc" {
  manifest = {
    "apiVersion" = "serving.knative.dev/v1"
    "kind"       = "Service"
    "metadata" = {
      "namespace" = var.namespace
      "name"      = var.name
      "annotations" = {
        "serving.knative.dev/rollout-duration" : "0s"
      }
      "labels" = {
        "networking.knative.dev/visibility" : "cluster-local"
      }
    }
    "spec" = {
      "template" = {
        "metadata" = {
          "annotations" = {
            "autoscaling.knative.dev/maxScale" = "1"
            "autoscaling.knative.dev/minScale" = "1"
          }
        }
        "spec" = {
          "serviceAccountName"   = var.name
          "containerConcurrency" = 0
          "timeoutSeconds"       = 600
          "containers" = [
            {
              "image" : ko_build.inmem.image_ref
              "env" = [
                {
                  "name"  = "WORKQUEUE_CONCURRENCY"
                  "value" = var.concurrent-work
                },
                {
                  "name"  = "WORKQUEUE_BATCH_SIZE"
                  "value" = local.dispatcher_batch_size
                },
                {
                  "name"  = "WORKQUEUE_TARGET"
                  "value" = var.reconciler-service
                },
              ]
              "ports" = [{
                "name"          = "h2c"
                "containerPort" = 8080
              }]
            }
          ]
        }
      }
    }
  }
}
