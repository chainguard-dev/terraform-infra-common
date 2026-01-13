# Complete Example: AWS Regional Go Service

This is a complete, production-ready example showing how to use the `aws/apprunner-regional-go-service` module to deploy a Go application to AWS App Runner with a single region deployment (configurable via variables).

## What This Example Includes

1. **Sample Go Application** (`app/`)
   - Simple HTTP server with health checks
   - JSON API with region-aware responses
   - Web UI for browser testing
   - Environment variable configuration
   - **Ready to deploy** - works out of the box!

2. **Automatic Resource Creation**
   - Module automatically creates IAM roles (service and instance roles)
   - Module automatically creates and manages ECR repository
   - Configurable force delete option for development
   - Automatic image scanning enabled
   - X-Ray permissions automatically attached when observability enabled

3. **Minimal Configuration**
   - No need to define IAM roles separately
   - No need to create ECR repositories manually
   - Just define your application and deploy!

4. **Optional Features** (commented out by default)
   - Secrets management (Secrets Manager and SSM Parameter Store)
   - Custom application permissions (S3, DynamoDB) - just attach to the created roles
   - VPC connector for private resource access

5. **Configurable Options**
   - Region selection via variable (default: us-east-1)
   - Observability toggle (default: false)
   - Health checks
   - Auto-scaling
   - Resource allocation (CPU/Memory)

## Prerequisites

1. **AWS Account** with appropriate permissions
2. **Terraform** >= 1.12
3. **AWS CLI** configured with credentials
4. **Go application** ready to deploy

## Quick Start

> **💡 Tip:** This example includes a `Makefile` with helpful commands. Run `make help` to see all available commands!

### 0. Test the App Locally (Optional)

The example includes a working Go application you can test before deploying:

```bash
# Using make
make run-local

# Or manually
cd app
go run main.go
```

Then in another terminal:
```bash
# Test all endpoints
make test-local

# Or test manually
curl http://localhost:8080/
curl http://localhost:8080/health
open http://localhost:8080/ui  # Web UI
```

Press Ctrl+C to stop the server.

### 1. Navigate to Example Directory

```bash
cd terraform/public-modules/modules/aws/apprunner-regional-go-service/example
```

### 2. (Optional) Customize Configuration

The example works out of the box with sensible defaults! But you can customize:

```bash
# Deploy to a different region
terraform plan -var="region=us-west-2"

# Enable observability (X-Ray tracing)
terraform plan -var="observability_enabled=true"

# Both options together
terraform plan -var="region=eu-west-1" -var="observability_enabled=true"
```

**Note:** The module automatically creates an ECR repository and builds your Go application. No manual ECR setup required!

### 3. Initialize Terraform

```bash
terraform init
```

### 4. Review the Plan

```bash
terraform plan -out=plan.out
```

### 5. Deploy

```bash
terraform apply "plan.out"
```

### 6. Get Service URL

```bash
terraform output service_url
```

Example output:
```
service_url = "https://abc123xyz.us-east-1.awsapprunner.com"
```

### 7. Test Your Service

```bash
# Get the URL
SERVICE_URL=$(terraform output -raw service_url)

# Test JSON API
curl $SERVICE_URL/

# Test health check
curl $SERVICE_URL/health

# View the web UI
open $SERVICE_URL/ui  # On macOS
# Or visit the URL in your browser
```

You should see responses like:
```json
{
  "message": "Hello from AWS App Runner! 🚀",
  "region": "us-east-1",
  "environment": "production",
  "hostname": "abc123def",
  "timestamp": "2025-01-09T10:00:00Z",
  "version": "1.0.0"
}
```

## Key Features

### Automatic Resource Management

The module automatically handles:
- **IAM Role Creation**: Automatically creates service and instance roles with proper trust policies
- **ECR Repository Creation**: No need to manually create or configure ECR repositories
- **Image Building**: Uses `ko` to build Go containers directly from source
- **Image Signing**: Automatically signs images with `cosign` for supply chain security
- **Policy Attachments**: Automatically attaches ECR access and X-Ray policies as needed
- **Smart Defaults**: Works out of the box with minimal configuration

### Flexible Configuration

- **Region Selection**: Deploy to any AWS region via variable
- **Observability Toggle**: Enable/disable X-Ray tracing as needed
- **ECR Management**: Option to use existing repositories or let module create them
- **Lifecycle Control**: Configure whether to keep ECR images on destroy

## Customization Options

### Adding Application-Specific IAM Permissions

The module creates base IAM roles automatically. To add permissions for your app's specific AWS resource access:

```hcl
# Grant your application access to S3, DynamoDB, etc.
resource "aws_iam_role_policy" "app_permissions" {
  role = module.my_go_service.instance_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["s3:GetObject", "s3:PutObject"]
        Resource = ["arn:aws:s3:::my-bucket/*"]
      },
      {
        Effect = "Allow"
        Action = ["dynamodb:GetItem", "dynamodb:PutItem"]
        Resource = ["arn:aws:dynamodb:*:${data.aws_caller_identity.current.account_id}:table/my-table"]
      }
    ]
  })
}
```

### ECR Repository Options

The module automatically creates an ECR repository. You can customize this behavior:

```hcl
module "my_go_service" {
  # ...

  # Use an existing ECR repository instead of creating one
  create_ecr_repository = false
  container = {
    source = {
      # ...
      repo = "123456789.dkr.ecr.us-east-1.amazonaws.com/existing-repo"
    }
  }

  # OR: Let module create it with a custom name
  create_ecr_repository = true
  ecr_repository_name   = "my-custom-repo-name"

  # OR: Enable force delete for development (deletes repo even with images)
  create_ecr_repository = true
  ecr_force_delete      = true  # Use with caution!
}
```

### Using Custom IAM Roles

For advanced use cases, you can disable automatic role creation:

```hcl
module "my_go_service" {
  # ...

  # Disable automatic role creation
  create_service_role  = false
  create_instance_role = false

  # Provide your own roles
  service_role_arn  = aws_iam_role.custom_service.arn
  instance_role_arn = aws_iam_role.custom_instance.arn
}
```

### Enable VPC Access

To access private resources like RDS or ElastiCache:

1. Uncomment the VPC connector section in `main.tf`
2. Update the VPC and subnet filters to match your environment
3. Update the module configuration:

```hcl
module "my_go_service" {
  # ...

  vpc_connector_arn = aws_apprunner_vpc_connector.this.arn
  egress            = "VPC"  # Route traffic through VPC
}
```

### Adjust Resources

For smaller or larger workloads:

```hcl
# Small service (0.25 vCPU, 512 MB)
cpu    = 256
memory = 512

# Medium service (1 vCPU, 2 GB) - Default
cpu    = 1024
memory = 2048

# Large service (4 vCPU, 8 GB)
cpu    = 4096
memory = 8192
```

### Configure Auto-Scaling

Adjust based on your traffic patterns:

```hcl
autoscaling = {
  min_instances   = 1    # Minimum instances (0 to pause during low traffic)
  max_instances   = 25   # Maximum instances (AWS default quota)
  max_concurrency = 100  # Concurrent requests per instance
}
```

### Private Service (VPC Only)

For internal-only services:

```hcl
ingress = "PRIVATE"  # Not accessible from internet
egress  = "VPC"      # Access VPC resources
```

## File Structure

```
example/
├── app/
│   ├── main.go      # Sample Go application with HTTP server
│   ├── go.mod       # Go module definition
│   └── README.md    # Application documentation
├── main.tf          # Complete Terraform configuration with IAM, secrets, and module usage
├── Makefile         # Helpful commands for development and deployment
└── README.md        # This file
```

## Environment Variables

Your Go application will have access to:

### Configured Environment Variables
- `PORT` - Port to listen on (8080)
- `ENVIRONMENT` - Environment name (production)
- `REGION` - AWS region where the service is deployed (e.g., us-east-1)

### Optional Secrets (from AWS)
If you uncomment the secrets resources in `main.tf`:
- `DATABASE_URL` - From Secrets Manager
- `API_KEY` - From SSM Parameter Store

### AWS-Provided Variables
App Runner automatically provides:
- `AWS_REGION` - Current AWS region
- `AWS_DEFAULT_REGION` - Current AWS region
- Plus standard AWS SDK environment variables

## Example Go Application

Your Go application should look something like this:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    // Access secrets
    dbURL := os.Getenv("DATABASE_URL")
    apiKey := os.Getenv("API_KEY")

    // Access region-specific config
    regionName := os.Getenv("REGION_NAME")
    regionEndpoint := os.Getenv("REGION_ENDPOINT")

    log.Printf("Starting server in %s on port %s", regionName, port)

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "OK\n")
    })

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello from %s!\n", regionName)
    })

    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## Monitoring

### View Logs

```bash
# Follow logs in real-time
aws logs tail /aws/apprunner/my-go-service --follow --region us-east-1
```

### View Service Status

```bash
# Get service details
aws apprunner describe-service \
  --service-arn $(terraform output -raw service_arn) \
  --region $(terraform output -raw region)
```

### View Deployments

```bash
# List recent operations
aws apprunner list-operations \
  --service-arn $(terraform output -raw service_arn) \
  --region $(terraform output -raw region)
```

### X-Ray Traces

Only available if `observability_enabled = true`:

1. Open AWS Console
2. Navigate to X-Ray → Service Map
3. View traces and performance metrics

## Cleanup

To destroy all resources:

```bash
terraform destroy
```

**Warning**: This will delete:
- App Runner service
- ECR repository (only if `ecr_force_delete = true`, otherwise will fail if images exist)
- IAM roles and policies
- Secrets Manager secrets (if uncommented)
- SSM parameters (if uncommented)
- VPC connectors (if created)

**Note**: If you get an error about ECR repository containing images, either:
1. Set `ecr_force_delete = true` in the module configuration, or
2. Manually delete images from ECR before running `terraform destroy`

## Additional Resources

- [AWS App Runner Documentation](https://docs.aws.amazon.com/apprunner/)
- [ko Build Tool](https://github.com/ko-build/ko)
- [cosign Image Signing](https://github.com/sigstore/cosign)
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | 5.0 |
| <a name="requirement_cosign"></a> [cosign](#requirement\_cosign) | 0.0.20 |
| <a name="requirement_ko"></a> [ko](#requirement\_ko) | 0.0.19 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_my_go_service"></a> [my\_go\_service](#module\_my\_go\_service) | ../ | n/a |

## Resources

| Name | Type |
|------|------|
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/5.0/docs/data-sources/caller_identity) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_observability_enabled"></a> [observability\_enabled](#input\_observability\_enabled) | Enable AWS X-Ray tracing for observability | `bool` | `true` | no |
| <a name="input_region"></a> [region](#input\_region) | AWS region to deploy the service | `string` | `"us-east-1"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_built_image"></a> [built\_image](#output\_built\_image) | Built and signed container image |
| <a name="output_ecr_repository_url"></a> [ecr\_repository\_url](#output\_ecr\_repository\_url) | ECR repository URL |
| <a name="output_region"></a> [region](#output\_region) | AWS region where the service is deployed |
| <a name="output_service_arn"></a> [service\_arn](#output\_service\_arn) | App Runner service ARN |
| <a name="output_service_status"></a> [service\_status](#output\_service\_status) | App Runner service status |
| <a name="output_service_url"></a> [service\_url](#output\_service\_url) | App Runner service URL |
<!-- END_TF_DOCS -->
