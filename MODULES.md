# Module Catalog

Reusable Terraform modules for Cloud Run services, event-driven architectures, and GCP infrastructure.

## Core Services

### [`regional-service`](./modules/regional-service/)

Deploy a regionalized Cloud Run service from a pre-built container image, with multi-region support, region-specific environment variables, VPC integration, and optional OpenTelemetry telemetry.

Use this when deploying a pre-built container as a Cloud Run service across one or more regions.

### [`regional-go-service`](./modules/regional-go-service/)

Deploy a regionalized Cloud Run service by building and signing Go source code using ko and cosign, with regional environment configuration and telemetry sidecar injection.

Use this when deploying a Go service from source — it handles the build, sign, and deploy pipeline.

### [`cron`](./modules/cron/)

Run a scheduled Cloud Run job on a cron schedule with optional OpenTelemetry monitoring and failure alerting.

Use this when you need a periodic background job (e.g., cleanup, sync, report generation).

### [`networking`](./modules/networking/)

Set up a GCP VPC network with regional subnets, DNS policies, and private Google API access for Cloud Run services using direct VPC egress.

Use this when Cloud Run services need VPC networking for private connectivity or egress control.

## Event-Driven Architecture

### [`cloudevent-broker`](./modules/cloudevent-broker/)

Create a regionalized Pub/Sub-based event broker with an ingress Cloud Run service that publishes CloudEvents, similar to Knative Broker.

Use this when you need a central event bus for publishing and routing CloudEvents.

### [`cloudevent-trigger`](./modules/cloudevent-trigger/)

Configure a push-based Pub/Sub subscription that delivers filtered CloudEvents to a private Cloud Run service via OIDC-authenticated requests.

Use this when a service should receive CloudEvents pushed to it in real time from a broker.

### [`cloudevent-pull-trigger`](./modules/cloudevent-pull-trigger/)

Create a pull-based Pub/Sub subscription with filtering for consumers to pull CloudEvents on-demand, ideal for bursty traffic with controlled consumption.

Use this when consumers need to pull events at their own pace rather than receiving pushes.

### [`cloudevent-recorder`](./modules/cloudevent-recorder/)

Record filtered CloudEvents to Google Cloud Storage and automatically transfer them into BigQuery tables with configurable retention and schema.

Use this when you need to archive events for analytics, audit, or long-term storage in BigQuery.

### [`bucket-events`](./modules/bucket-events/)

Forward Google Cloud Storage bucket events to a CloudEvent broker via Pub/Sub, converting GCS notifications into CloudEvents.

Use this when GCS object changes (create, delete, archive) should trigger downstream event processing.

## GitHub Integration

### [`github-events`](./modules/github-events/)

Receive GitHub webhook events, publish them to a broker as CloudEvents, and record them to BigQuery.

Use this when you need GitHub webhook events available as CloudEvents for downstream consumers.

### [`github-bots`](./modules/github-bots/)

Scaffold event-driven GitHub bots that receive webhook events through a broker and run as regional Cloud Run services with dashboards and alerting.

Use this when building a GitHub bot that reacts to webhook events (PR comments, issue labels, etc.).

### [`github-wif-provider`](./modules/github-wif-provider/)

Set up Google Workload Identity Federation to accept GitHub Actions OIDC tokens for passwordless authentication from CI workflows.

Use this when you need to create a WIF provider pool for GitHub Actions in a GCP project.

### [`github-gsa`](./modules/github-gsa/)

Create a Google Service Account that GitHub Actions can assume via Workload Identity Federation with fine-grained branch, ref, and workflow controls.

Use this when a specific GitHub Actions workflow needs to authenticate as a GCP service account.

## External Integrations

### [`linear-events`](./modules/linear-events/)

Receive Linear webhook events and publish them to a regional event broker for downstream processing of issues, comments, and project changes.

Use this when you need Linear webhook events available as CloudEvents for automation.

### [`azure-github-wif`](./modules/azure-github-wif/)

Set up GitHub Actions OIDC federation with Azure AD for passwordless authentication from GitHub Actions to Azure resources.

Use this when GitHub Actions workflows need to authenticate with Azure without static credentials.

## Databases and Storage

### [`cloudsql-postgres`](./modules/cloudsql-postgres/)

Provision a private Cloud SQL PostgreSQL instance with configurable HA, cross-region read replicas, automated backups, and IAM authentication.

Use this when a service needs a managed PostgreSQL database with private networking.

### [`redis`](./modules/redis/)

Create a managed Redis instance with configurable HA, authentication, persistence, and automatic credential management in Secret Manager.

Use this when a service needs a managed Redis instance for caching or session storage.

### [`bigquery-logsink`](./modules/bigquery-logsink/)

Export logs from specified sources into BigQuery via Cloud Logging sinks, with optional retention policies and ingestion health alerts.

Use this when you need to route logs to BigQuery for querying or long-term retention.

### [`secret`](./modules/secret/)

Create a Google Secret Manager secret with automatic access controls, version management, and unauthorized access alerting.

Use this when you need to store a sensitive value (API key, token, certificate) in Secret Manager.

### [`configmap`](./modules/configmap/)

Store non-sensitive configuration data in Secret Manager for mounting as environment variables or volumes in Cloud Run services.

Use this when you need to inject non-sensitive configuration into Cloud Run services without rebuilding.

## Kubernetes

### [`gke`](./modules/gke/)

Provision a GKE cluster with Dataplane V2, workload identity, managed Prometheus, configurable node pools, and cluster autoscaling.

Use this when you need a standard GKE cluster with full control over node pool configuration.

### [`gke-ap`](./modules/gke-ap/)

Deploy a GKE Autopilot cluster with automatic node provisioning and the same networking, monitoring, and identity capabilities as standard GKE.

Use this when you want a GKE cluster without managing node pools — Autopilot handles scaling and provisioning.

## Load Balancing and Networking

### [`serverless-gclb`](./modules/serverless-gclb/)

Front multiple regional Cloud Run services with a Google Cloud Load Balancer, managed SSL certificates, DNS records, and host-based routing.

Use this when Cloud Run services need a global load balancer with custom domains and TLS.

### [`serverless-gclb-cbd`](./modules/serverless-gclb-cbd/)

Same as serverless-gclb but with create-before-destroy lifecycle on the URL map to avoid resource conflicts during redeployment.

Use this instead of `serverless-gclb` when URL map updates cause downtime or conflicts during `terraform apply`.

### [`authorize-private-service`](./modules/authorize-private-service/)

Grant a service account permission to invoke a private Cloud Run service and return the service URI.

Use this when one service needs to call another private Cloud Run service.

### [`bastion`](./modules/bastion/)

Provision a hardened IAP-only jump host VM with OS Login, auditd, automatic patching, optional Cloud SQL Auth Proxy, and optional Cloud NAT.

Use this when you need SSH access to private resources through IAP, or a Cloud SQL Auth Proxy for database access.

## Observability

### [`alerting`](./modules/alerting/)

Create Cloud Monitoring alert policies and custom logging metrics to detect application failures, OOM errors, panics, high error rates, and dead letter queue backups.

Use this when deploying a Cloud Run service that needs standard alerting for failures and resource exhaustion.

### [`dashboard`](./modules/dashboard/)

Create normalized Google Cloud Monitoring dashboards from JSON definitions with cleaned-up encoding.

Use this when you have a dashboard JSON definition and need to provision it as a Cloud Monitoring dashboard.

### [`prober`](./modules/prober/)

Deploy a regionalized Go-based prober service for custom health checks, with optional GCLB integration for global uptime monitoring.

Use this when you need custom health probes beyond what GCP uptime checks provide.

### [`slo`](./modules/slo/)

Define Service Level Objectives for Cloud Run services with request-based success metrics, multi-region rolling windows, and burn-rate alerting.

Use this when a service needs formal SLOs with automated burn-rate alerts.

### [`ocistatus`](./modules/ocistatus/)

Create an Artifact Registry repository for OCI status attestations with automatic cleanup policies and write access grants.

Use this when a service publishes OCI status attestations and needs a dedicated repository.

## AWS

### [`aws/apprunner-regional-go-service`](./modules/aws/apprunner-regional-go-service/)

Deploys a containerized Go application to AWS App Runner with ECR, IAM roles, auto-scaling, health checks, and optional X-Ray tracing.

Use this when deploying a Go service on AWS App Runner.

### [`aws/prober`](./modules/aws/prober/)

Wraps `apprunner-regional-go-service` to deploy uptime probers with a shared authorization secret and optional CloudWatch Synthetics canaries.

Use this when deploying a prober on AWS infrastructure.

## Utilities

### [`limited-concat`](./modules/limited-concat/)

Concatenate two strings while respecting a maximum length limit by truncating only the prefix as needed.

Use this when constructing resource names that must stay within a character limit (e.g., GCP's 63-char label constraint).
