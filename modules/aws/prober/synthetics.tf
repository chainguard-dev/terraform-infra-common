/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

// IAM role for CloudWatch Synthetics canary
resource "aws_iam_role" "canary" {
  count = var.cloudwatch_synthetics_enabled ? 1 : 0

  name = "${var.name}-canary-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })

  tags = merge(var.tags, {
    Name    = "${var.name}-canary-role"
    team    = var.team
    product = var.product
  })
}

// Attach X-Ray write policy for tracing
resource "aws_iam_role_policy_attachment" "canary_xray" {
  count = var.cloudwatch_synthetics_enabled && var.enable_profiler ? 1 : 0

  role       = aws_iam_role.canary[0].name
  policy_arn = "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess"
}

// Inline policy for CloudWatch metrics and S3 access
resource "aws_iam_role_policy" "canary_permissions" {
  count = var.cloudwatch_synthetics_enabled ? 1 : 0

  name = "${var.name}-canary-permissions"
  role = aws_iam_role.canary[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "CloudWatchMetrics"
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "cloudwatch:namespace" = "CloudWatchSynthetics"
          }
        }
      },
      {
        Sid    = "S3ArtifactAccess"
        Effect = "Allow"
        Action = [
          "s3:PutObject"
        ]
        Resource = "${aws_s3_bucket.canary_artifacts[0].arn}/*"
      },
      {
        Sid    = "S3BucketLocation"
        Effect = "Allow"
        Action = [
          "s3:GetBucketLocation"
        ]
        Resource = aws_s3_bucket.canary_artifacts[0].arn
      },
      {
        Sid    = "CloudWatchLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:log-group:/aws/lambda/cwsyn-${var.name}-*:*"
      },
      {
        Sid    = "CloudWatchLogsCreateGroup"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup"
        ]
        Resource = "arn:aws:logs:*:*:log-group:/aws/lambda/cwsyn-${var.name}-*"
      }
    ]
  })
}

// S3 bucket for canary artifacts
resource "aws_s3_bucket" "canary_artifacts" {
  count = var.cloudwatch_synthetics_enabled ? 1 : 0

  bucket = "${var.name}-canary-artifacts-${data.aws_caller_identity.current.account_id}"

  tags = merge(var.tags, {
    Name    = "${var.name}-canary-artifacts"
    team    = var.team
    product = var.product
  })
}

// Get current AWS account
data "aws_caller_identity" "current" {}

// Create the canary script for HTTP check with Authorization header
data "archive_file" "canary_script" {
  count = var.cloudwatch_synthetics_enabled ? 1 : 0

  type        = "zip"
  output_path = "${path.module}/canary-${substr(md5(file("${path.module}/templates/canary-script.js.tpl")), 0, 8)}.zip"

  source {
    content = templatefile("${path.module}/templates/canary-script.js.tpl", {
      service_url   = module.this.service_url
      authorization = random_password.secret.result
    })
    filename = "nodejs/node_modules/apiCanaryBlueprint.js"
  }
}

// CloudWatch Synthetics canary for uptime monitoring
resource "aws_synthetics_canary" "uptime_check" {
  count = var.cloudwatch_synthetics_enabled ? 1 : 0

  name                 = "${var.name}-canary"
  artifact_s3_location = "s3://${aws_s3_bucket.canary_artifacts[0].bucket}/"
  execution_role_arn   = aws_iam_role.canary[0].arn
  handler              = "apiCanaryBlueprint.handler"
  zip_file             = data.archive_file.canary_script[0].output_path
  runtime_version      = var.canary_runtime_version
  start_canary         = var.start_canary

  schedule {
    expression = var.canary_schedule
  }

  run_config {
    timeout_in_seconds = tonumber(replace(var.timeout, "s", ""))
    memory_in_mb       = 960
    active_tracing     = var.enable_profiler

    environment_variables = {
      AUTHORIZATION = random_password.secret.result
    }
  }

  artifact_config {
    s3_encryption {
      encryption_mode = "SSE_S3"
    }
  }

  // The canary script checks the prober endpoint
  // This is a basic HTTP check blueprint
  success_retention_period = 2
  failure_retention_period = 14

  tags = merge(var.tags, {
    Name    = "${var.name}-canary"
    team    = var.team
    product = var.product
  })

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [
    aws_iam_role_policy.canary_permissions,
    module.this,
    data.archive_file.canary_script
  ]
}
