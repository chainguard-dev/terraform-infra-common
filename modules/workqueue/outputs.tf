output "receiver" {
  depends_on = [ module.receiver-service ]
  value = {
    name = "${var.name}-rcv"
  }
}
