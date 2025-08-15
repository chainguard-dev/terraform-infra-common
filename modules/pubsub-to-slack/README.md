# Pub/Sub to Slack Module

Forwards JSON messages from Google Cloud Pub/Sub to Slack via webhooks.

## Usage

**Prerequisites:** Create a Secret Manager secret containing your Slack webhook URL.

Use the module:

```hcl
module "budget_alerts" {
  source = "./modules/pubsub-to-slack"

  name                     = "budget-alerts"
  project_id               = var.project_id
  slack_webhook_secret_id  = "budget-alerts-slack-webhook"
  slack_channel            = "#alerts"
  message_template         = "🚨 Budget {{.budgetDisplayName}} exceeded threshold"

  region  = "us-central1"
  network = var.network
  subnet  = var.subnet
}
```

## Security Best Practices

- Never commit Slack webhook URLs to Git repositories
- Create secrets manually via `gcloud` CLI or Google Cloud Console
- Use separate Secret Manager secrets for different environments (dev/staging/prod)
- Ensure proper IAM permissions are configured for secret access

## Template Syntax
Uses Go's text/template with JSON fields from Pub/Sub messages:
- `{{.fieldName}}` - Insert field value
- `{{if .condition}}text{{end}}` - Conditionals

## Outputs

- `topic_name` - Pub/Sub topic for publishing messages
- `service_url` - Cloud Run service URL

## Building

The Go service is automatically built and deployed by the `regional-go-service` module.
No manual container building required.

## Environment Variables

The Cloud Run service uses these environment variables:

- `SLACK_WEBHOOK_SECRET` - Secret Manager secret ID for webhook URL
- `SLACK_CHANNEL` - Target Slack channel
- `MESSAGE_TEMPLATE` - Template for formatting messages
- `PROJECT_ID` - GCP project ID

## Variables

- `slack_webhook_secret_id` - (Required) Secret Manager secret ID containing the Slack webhook URL
- `name` - (Required) Name for the Pub/Sub to Slack bridge resources
- `project_id` - (Required) The GCP project ID
- `region` - (Required) The region where the service will be deployed
- `network` - (Required) The network for the service to egress traffic via
- `subnet` - (Required) The subnetwork for the service to egress traffic via
- `slack_channel` - (Optional) Slack channel, defaults to "#alerts"
- `message_template` - (Optional) Go template for message formatting

## License

Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
