# Upgrade fixture

The assembly and load-balancer fixtures link to the production Terraform files.
The relative module paths and resource addresses match the production modules,
so the upgrade tests exercise the real input forwarding and state moves.

The legacy fixture and the load-balancer fixture share `configuration.tf.inc`.
This test-only output exposes complete resource values without adding outputs
to the public modules. Each upgrade compares these values with the legacy state.
Only service-backend `enable_cdn` changes from null to false in the expected
values.

The assembly also exports the DNS component's record configuration. Each
upgrade compares the complete ordered record list with the legacy fixture,
including address values and records for disabled hostnames.
