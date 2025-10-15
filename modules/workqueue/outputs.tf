output "receiver" {
  depends_on = [module.receiver-service]
  value = {
    name = "${var.name}-rcv"
  }
}

output "dispatcher" {
  depends_on = [module.dispatcher-service]
  value = {
    name = "${var.name}-dsp"
  }
}
