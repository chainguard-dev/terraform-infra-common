# Pub/Sub to Slack Module

This module creates a generic bridge that receives JSON messages from Google Cloud Pub/Sub and forwards them to Slack with configurable formatting.

## Architecture

The module deploys:
- **Pub/Sub Topic**: Receives notifications from other GCP services
- **Cloud Run Service**: Processes messages and sends to Slack
- **Push Subscription**: Delivers messages from topic to Cloud Run service
- **Dead Letter Queue**: Handles failed message deliveries
- **Secret Manager**: Securely stores Slack webhook URL

## Usage

### Basic Example

```hcl
module "budget_slack_alerts" {
  source = "./modules/pubsub-to-slack"

  name       = "budget-alerts"
  project_id = var.project_id
  region     = "us-central1"
  network    = var.network
  subnet     = var.subnet

  slack_webhook_url = var.slack_webhook_url
  slack_channel     = "#budget-alerts"
  message_template  = "🚨 Budget Alert: $${budgetDisplayName} exceeded $${alertThresholdExceeded*100}% ($${costAmount} $${currencyCode})"

  image = "gcr.io/your-project/pubsub-slack-bridge:latest"

  squad   = "platform"
  product = "infrastructure"
}
```

### Budget Alerts Integration

```hcl
# Deploy the Slack bridge
module "budget_slack_notifications" {
  source = "./modules/pubsub-to-slack"

  name       = "budget-notifications"
  project_id = var.project_id
  region     = var.region
  network    = var.network
  subnet     = var.subnet

  slack_webhook_url = var.slack_webhook_url
  slack_channel     = "#budget-alerts"
  message_template  = "💸 Budget Alert: *$${budgetDisplayName}* exceeded $${alertThresholdExceeded}% threshold\n💰 Current spend: $${costAmount} $${currencyCode} / $${budgetAmount} $${currencyCode}\n📅 Period: $${costIntervalStart}"

  image = var.pubsub_slack_bridge_image
}

# Create a Pub/Sub notification channel for budget alerts
resource "google_monitoring_notification_channel" "budget_pubsub" {
  display_name = "Budget Pub/Sub Channel"
  type         = "pubsub"

  labels = {
    topic = module.budget_slack_notifications.topic_name
  }
}

# Use the notification channel in budget alerts
resource "google_billing_budget" "example" {
  billing_account = var.billing_account
  display_name    = "Example Budget"

  amount {
    specified_amount {
      currency_code = "USD"
      units         = "100"
    }
  }

  threshold_rules {
    threshold_percent = 0.5
  }

  threshold_rules {
    threshold_percent = 0.9
  }

  all_updates_rule {
    monitoring_notification_channels = [
      google_monitoring_notification_channel.budget_pubsub.name
    ]
  }
}
```

## Message Templating

The `message_template` variable supports simple variable substitution using `$${field_name}` syntax. Note the double `$` to escape Terraform's interpolation.

### Budget Alert Fields

For Google Cloud Budget alerts, available fields include:
- `budgetDisplayName`: The display name of the budget
- `alertThresholdExceeded`: The threshold percentage exceeded (e.g., 0.5 for 50%)
- `costAmount`: Current cost amount
- `budgetAmount`: The budget limit amount
- `costIntervalStart`: Start of the current billing period
- `currencyCode`: Currency code (e.g., "USD")
- `budgetAmountType`: Type of budget amount ("SPECIFIED_AMOUNT" or "LAST_PERIOD_AMOUNT")
- `forecastThresholdExceeded`: Forecasted threshold exceeded (if applicable)

### Template Examples

```hcl
# Simple alert
message_template = "Budget $${budgetDisplayName} exceeded $${alertThresholdExceeded*100}%"

# Detailed alert with formatting
message_template = <<-EOT
🚨 *Budget Alert*
Budget: $${budgetDisplayName}
Threshold: $${alertThresholdExceeded*100}% exceeded
Current: $${costAmount} $${currencyCode}
Budget: $${budgetAmount} $${currencyCode}
Period: $${costIntervalStart}
EOT

# Custom calculation in template
message_template = "Alert: $${budgetDisplayName} is at $${costAmount}/$${budgetAmount} $${currencyCode} ($${alertThresholdExceeded*100}%)"
```

## Building the Container Image

To build and deploy the container image:

```bash
# Build the image
cd modules/pubsub-to-slack/cmd/pubsub-slack-bridge
docker build -t gcr.io/your-project/pubsub-slack-bridge:latest .

# Push to registry
docker push gcr.io/your-project/pubsub-slack-bridge:latest
```

## Container Configuration

The Cloud Run service is configured via environment variables:

- `SLACK_WEBHOOK_SECRET`: Secret Manager secret ID containing the Slack webhook URL
- `SLACK_CHANNEL`: Target Slack channel (e.g., "#alerts" or "@user")
- `MESSAGE_TEMPLATE`: Template for formatting messages
- `PROJECT_ID`: GCP project ID for accessing Secret Manager
- `ENABLE_PROFILER`: Enable Cloud Profiler (set by `enable_profiler` variable)

## Monitoring

The module includes:
- Dead letter queue for failed message processing
- Configurable retry policies
- Health check endpoint at `/health`
- Structured logging via chainguard-dev/clog
- Prometheus metrics on port 2112 (via httpmetrics package)
- OpenTelemetry tracing integration
- Optional Cloud Profiler support

Failed messages are sent to the dead letter topic `{name}-dlq` after `max_delivery_attempts` retries.

## Slack Webhook Setup

1. Create a Slack app at https://api.slack.com/apps
2. Add "Incoming Webhooks" feature
3. Create a webhook for your target channel
4. Use the webhook URL as the `slack_webhook_url` variable

## Security

- Slack webhook URL is stored in Secret Manager
- Cloud Run service uses a dedicated service account
- Network access is restricted via VPC
- Follows principle of least privilege for IAM permissions

## Variables

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| `name` | Name for the Pub/Sub to Slack bridge resources | `string` | n/a | yes |
| `project_id` | The GCP project ID | `string` | n/a | yes |
| `slack_webhook_url` | The Slack webhook URL | `string` | n/a | yes |
| `image` | Container image for the bridge service | `string` | n/a | yes |
| `network` | VPC network for Cloud Run service | `string` | n/a | yes |
| `subnet` | VPC subnet for Cloud Run service | `string` | n/a | yes |
| `region` | GCP region to deploy the service | `string` | `"us-central1"` | no |
| `slack_channel` | Slack channel to send messages to | `string` | `"#alerts"` | no |
| `message_template` | Template for formatting Slack messages | `string` | `"Notification: $${message}"` | no |
| `squad` | Squad/team label | `string` | `""` | no |
| `product` | Product label | `string` | `""` | no |
| `labels` | Additional resource labels | `map(string)` | `{}` | no |
| `enable_profiler` | Enable Cloud Profiler for the service | `bool` | `false` | no |

See `variables.tf` for additional Cloud Run and Pub/Sub configuration options.

## Outputs

| Name | Description |
|------|-------------|
| `topic_name` | Name of the Pub/Sub topic for publishing messages |
| `topic_id` | Full resource ID of the Pub/Sub topic |
| `service_url` | URL of the Cloud Run service |
| `service_account_email` | Email of the service account |

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| google | >= 4.0 |
| google-beta | >= 4.0 |

## License

Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
