# Architecture Decisions

## Slack Integration Approach

**Decision**: Use direct HTTP calls to Slack webhook URLs instead of the `slack-go/slack` library.

**Context**:
- Initially considered using the `slack-go/slack` library for Slack integration
- The library is designed for Slack's Web API using bot tokens, not incoming webhooks
- Two approaches were evaluated:
  1. **Bot Token + Web API**: Use `slack-go/slack` with bot tokens and `client.PostMessage()`
  2. **Webhook URLs**: Direct HTTP POST to webhook URLs with JSON payload

**Decision Rationale**:
- **Simplicity**: Webhook approach requires minimal setup - users just need to create an incoming webhook
- **Generic Use Case**: This module is designed as a simple bridge for any JSON-based Pub/Sub notifications
- **Reduced Dependencies**: No need for additional Slack API libraries when simple HTTP calls suffice
- **Lower Barrier to Entry**: Creating a webhook is easier than setting up a full Slack app with bot permissions

**Trade-offs**:
- **Pros**:
  - Simple setup and configuration
  - Fewer dependencies to manage
  - Direct control over HTTP requests
  - Works well for basic text messaging needs
- **Cons**:
  - Limited to webhook capabilities (no interactive features, file uploads, etc.)
  - No built-in retry logic from Slack library
  - Manual JSON payload construction

**Implementation Details**:
- Accept `slack_webhook_url` as input variable
- Store webhook URL securely in Secret Manager
- Make direct HTTP POST requests with JSON payload containing `channel` and `text`
- Handle HTTP response codes for basic error detection

**Future Considerations**:
If more advanced Slack features are needed (interactive messages, file uploads, advanced formatting), we could:
1. Add a new module variant that uses the full Slack Web API
2. Add a configuration option to choose between webhook and API approaches
3. Migrate existing webhook users to API tokens with proper migration path

**Date**: January 2025
**Contributors**: Assistant (via user request)
