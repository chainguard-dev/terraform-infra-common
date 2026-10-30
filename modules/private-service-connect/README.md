# `private-service-connect`

Reusable [Private Service Connect (PSC)][psc] building blocks for exposing a
regional internal Cloud Run service across VPCs (and across projects) without
public ingress.

The module is split into two submodules that are applied independently, usually
from two different Terraform states:

- [`producer/`](./producer/) — fronts an existing regional internal Cloud Run
  service with a regional internal Application Load Balancer (INTERNAL_MANAGED)
  and publishes it via a PSC **service attachment**.
- [`consumer/`](./consumer/) — creates a PSC **endpoint** (a forwarding rule
  targeting a service attachment) with an internal IP in the consumer VPC.

## Flow

```
                  producer project / VPC                      consumer project / VPC
  ┌──────────────────────────────────────────────┐      ┌──────────────────────────────┐
  │ Cloud Run service                              │      │                              │
  │   -> serverless NEG                            │      │   PSC endpoint               │
  │   -> regional INTERNAL_MANAGED backend service │      │   (forwarding rule, internal │
  │   -> regional URL map                          │      │    IP, lb scheme = "")       │
  │   -> regional target HTTP proxy                │      │            │                 │
  │   -> internal ALB forwarding rule (VIP)        │      │            │ targets          │
  │   -> PSC service attachment ──────────────────────────────────────┘                 │
  │      (ACCEPT_MANUAL + accept list)             │      │                              │
  └──────────────────────────────────────────────┘      └──────────────────────────────┘
```

Traffic from the consumer reaches the endpoint's internal IP, traverses the PSC
service attachment into the producer VPC, hits the internal ALB, and is routed
to the Cloud Run service via the serverless NEG.

## Caller-owned networking

These subnets are **created by the caller** (the environment stack), not by this
module. Each submodule accepts their self-links as inputs:

- A `REGIONAL_MANAGED_PROXY` proxy-only subnet in the producer region — required
  by the regional INTERNAL_MANAGED ALB. The producer module references it via
  `proxy_only_subnet` purely to order the ALB forwarding rule after the subnet
  exists.
- One or more `PRIVATE_SERVICE_CONNECT` NAT subnets in the producer region — used
  by the service attachment (`psc_nat_subnets`).
- The consumer subnetwork in which the endpoint's internal IP is allocated.

## Two-phase apply

The producer's service-attachment self-link is an input to the consumer, so the
two sides cannot be applied in a single pass:

1. **Apply the producer** first. It emits `service_attachment_id` (and the
   internal LB VIP).
2. **Hand the service-attachment self-link to the consumer.** Because the two
   sides typically live in separate states, this is passed as a tfvar in the
   consumer stack (the cross-state hand-off is the caller's responsibility — it
   is out of scope for these modules).
3. **Apply the consumer.** It creates the PSC endpoint targeting the service
   attachment and emits `endpoint_ip` and `psc_connection_id`.

Because the producer uses `connection_preference = "ACCEPT_MANUAL"`, the consumer
project must appear in the producer's `consumer_accept_projects` list before the
endpoint connection is accepted.

## Keeping the service PSC-only

The attachment is not the only way in. Two other paths reach the Cloud Run
service, and the producer leaves both open unless the caller closes them:

- **The internal ALB's address.** Envoy-based load balancers admit every
  client in the network, peered networks, and hybrid links when no Cloud NGFW
  rule targets them. Set `lb_firewall` to a regional network firewall policy
  the caller owns and has associated with `network` in `region`; the producer
  adds an allow for its PSC NAT subnets and a logged deny for every other
  source, both aimed at its own forwarding rule.
- **The service's `run.app` URL.** Cloud Run counts every VM, pod, and Google
  service in the project as internal on that URL. Set
  `default_uri_disabled = true` on the `regional-service` or
  `regional-go-service` the producer fronts.

To accept one consumer network rather than every network in a consumer
project, set `consumer_accept_networks` instead of `consumer_accept_projects`.

## Notes

- The internal ALB speaks plain HTTP on :80 unless `tls_certificates` is set,
  which switches the frontend to HTTPS on :443 (see the producer README). TLS
  to the `run.app` backend is terminated by the serverless NEG, and inbound
  authorization is enforced via Cloud Run invoker IAM (configured separately).
- DNS for the consumer endpoint is intentionally out of scope; wire it up in the
  consumer stack using the `endpoint_ip` output.

[psc]: https://cloud.google.com/vpc/docs/private-service-connect
