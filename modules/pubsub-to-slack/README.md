# Pub/Sub to Slack Module

Forwards JSON messages from Google Cloud Pub/Sub to Slack via webhooks.

## Usage

```hcl
module "budget_alerts" {
  source = "./modules/pubsub-to-slack"

  name              = "budget-alerts"
  project_id        = var.project_id
  slack_webhook_url = var.slack_webhook_url
  slack_channel     = "#alerts"
  message_template  = "🚨 Budget {{.budgetDisplayName}} exceeded threshold"
  image             = var.container_image
  network           = var.network
  subnet            = var.subnet
}
```

## Template Syntax

Uses Go's text/template with JSON fields from Pub/Sub messages:
- `{{.fieldName}}` - Insert field value
- `{{if .condition}}text{{end}}` - Conditionals

## Outputs

- `topic_name` - Pub/Sub topic for publishing messages
- `service_url` - Cloud Run service URL

## Building

```bash
cd modules/pubsub-to-slack/cmd/pubsub-slack-bridge
docker build -t gcr.io/your-project/pubsub-slack-bridge:latest .
```

## Environment Variables

- `SLACK_WEBHOOK_SECRET` - Secret Manager secret ID for webhook URL
- `SLACK_CHANNEL` - Target Slack channel
- `MESSAGE_TEMPLATE` - Template for formatting messages
- `PROJECT_ID` - GCP project ID

## License

Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
