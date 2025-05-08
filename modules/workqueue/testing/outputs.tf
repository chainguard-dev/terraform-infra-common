output "receiver" {
  depends_on = [kubernetes_manifest.inmem-ksvc]
  value      = "http://${var.name}.${var.namespace}.svc"
}
