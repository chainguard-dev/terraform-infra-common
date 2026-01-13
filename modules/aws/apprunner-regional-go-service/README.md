# AWS Regional Go Service Module

This Terraform module deploys Go applications to AWS App Runner. It's the AWS equivalent of the `regional-go-service` module for GCP Cloud Run.

For multi-container services or more complex orchestration, consider ECS Fargate instead.

## Features

- **Automatic IAM Role Creation**: Creates required service and instance roles by default (or use your own)
- **Automatic ECR Repository Management**: Optionally creates and manages ECR repositories automatically
- **Go build integration**: Uses [ko](https://github.com/ko-build/ko) to build Go container images from source
- **Image signing**: Automatically signs built images with [cosign](https://github.com/sigstore/cosign)
- **AWS App Runner**: Fully managed container service (closest AWS equivalent to GCP Cloud Run)
- **Simple configuration**: Minimal setup required - no VPC plumbing unless needed
- **Auto-scaling**: Built-in request-based auto-scaling with configurable concurrency
- **HTTPS by default**: Automatic TLS certificates and load balancing
- **Observability**: Optional AWS X-Ray tracing and CloudWatch logging

## Usage

### Basic Example (Fully Automated)

The module automatically creates all required resources including IAM roles and ECR repository:

```hcl
module "my_service" {
  source = "./modules/aws/apprunner-regional-go-service"

  name = "my-go-service"
  team = "platform"

  # IAM roles and ECR repository are created automatically!
  # No need to define them separately

  container = {
    source = {
      working_dir = path.module
      importpath  = "github.com/my-org/my-service"
    }
    port = 8080
    env = [
      {
        name  = "PORT"
        value = "8080"
      },
      {
        name  = "ENVIRONMENT"
        value = "production"
      }
    ]
    health_check = {
      protocol            = "HTTP"
      path                = "/health"
      interval            = 10
      timeout             = 5
      healthy_threshold   = 1
      unhealthy_threshold = 3
    }
  }

  cpu    = 1024
  memory = 2048

  autoscaling = {
    min_instances   = 1
    max_instances   = 10
    max_concurrency = 100
  }

  observability_enabled = true  # Enable X-Ray tracing
}

# Optionally add application-specific IAM permissions
resource "aws_iam_role_policy" "app_permissions" {
  role = module.my_service.instance_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["s3:GetObject", "s3:PutObject"]
      Resource = ["arn:aws:s3:::my-bucket/*"]
    }]
  })
}
```

### Using Custom IAM Roles

If you need custom IAM configurations, you can provide your own roles:

```hcl
# Define custom IAM roles
resource "aws_iam_role" "custom_service_role" {
  name = "custom-apprunner-service-role"
  # ... custom configuration
}

resource "aws_iam_role" "custom_instance_role" {
  name = "custom-apprunner-instance-role"
  # ... custom configuration
}

module "my_service_custom_roles" {
  source = "./modules/aws/apprunner-regional-go-service"

  name = "my-service"
  team = "platform"

  # Disable automatic role creation
  create_service_role  = false
  create_instance_role = false

  # Provide your own roles
  service_role_arn  = aws_iam_role.custom_service_role.arn
  instance_role_arn = aws_iam_role.custom_instance_role.arn

  container = {
    source = {
      working_dir = path.module
      importpath  = "github.com/my-org/my-service"
    }
    port = 8080
  }
}
```

### Using Existing ECR Repository

```hcl
module "my_service_existing_ecr" {
  source = "./modules/aws/apprunner-regional-go-service"

  name = "my-service"
  team = "platform"

  # Use existing ECR repository
  create_ecr_repository = false

  container = {
    source = {
      working_dir = path.module
      importpath  = "github.com/my-org/my-service"
      repo        = "123456789.dkr.ecr.us-east-1.amazonaws.com/my-existing-repo"
    }
    port = 8080
  }
}
```

### With Secrets

```hcl
module "my_service_with_secrets" {
  source = "./modules/aws/apprunner-regional-go-service"

  name = "my-service"
  team = "platform"

  container = {
    source = {
      working_dir = path.module
      importpath  = "github.com/my-org/my-service"
    }
    port = 8080
    env = [
      {
        name  = "ENVIRONMENT"
        value = "production"
      }
    ]
    # Secrets from AWS Secrets Manager or SSM Parameter Store
    secrets = [
      {
        name  = "DATABASE_URL"
        value = "arn:aws:secretsmanager:us-east-1:123456789:secret:db-url-abc123"
      },
      {
        name  = "API_KEY"
        value = "arn:aws:ssm:us-east-1:123456789:parameter/api-key"
      }
    ]
    health_check = {
      protocol = "TCP"
    }
  }
}

# Grant the instance role permission to read secrets
resource "aws_iam_role_policy" "secrets_access" {
  role = module.my_service_with_secrets.instance_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "secretsmanager:GetSecretValue",
        "ssm:GetParameter"
      ]
      Resource = [
        "arn:aws:secretsmanager:us-east-1:123456789:secret:db-url-abc123",
        "arn:aws:ssm:us-east-1:123456789:parameter/api-key"
      ]
    }]
  })
}
```

### With VPC Access (Private Resources)

```hcl
# First, create a VPC Connector for private resource access
resource "aws_apprunner_vpc_connector" "this" {
  vpc_connector_name = "my-service-vpc-connector"
  subnets            = ["subnet-abc", "subnet-def"]
  security_groups    = ["sg-123"]
}

module "my_service_with_vpc" {
  source = "./modules/aws/apprunner-regional-go-service"

  name = "my-service"
  team = "platform"

  # VPC connector for private resource access
  vpc_connector_arn = aws_apprunner_vpc_connector.this.arn

  # Service still publicly accessible, but can access private resources
  ingress = "PUBLIC"
  egress  = "VPC"

  container = {
    source = {
      working_dir = path.module
      importpath  = "github.com/my-org/my-service"
    }
    port = 8080
    env = [
      {
        name  = "DATABASE_HOST"
        value = "rds-instance.internal.example.com"
      }
    ]
  }
}
```

### Private Service (VPC Only)

```hcl
module "my_private_service" {
  source = "./modules/aws/apprunner-regional-go-service"

  name = "my-internal-service"
  team = "platform"

  # VPC connector for private resource access
  vpc_connector_arn = aws_apprunner_vpc_connector.this.arn

  # Not publicly accessible - only from VPC
  ingress = "PRIVATE"
  egress  = "VPC"

  container = {
    source = {
      working_dir = path.module
      importpath  = "github.com/my-org/my-internal-service"
    }
    port = 8080
  }
}
```

## IAM Role Management

### Automatic IAM Role Creation (Default)

By default, the module automatically creates the required IAM roles:

- **Service Role**: Used by App Runner to pull images from ECR and write logs to CloudWatch
- **Instance Role**: Used by your running containers for AWS service access

The module also automatically attaches:
- ECR access policy to the service role
- X-Ray write access to the instance role (when `observability_enabled = true`)

You can add application-specific permissions using the exported role names:

```hcl
module "my_service" {
  source = "./modules/aws/apprunner-regional-go-service"
  # ... configuration
}

# Add custom permissions to the instance role
resource "aws_iam_role_policy" "app_permissions" {
  role = module.my_service.instance_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["s3:GetObject", "dynamodb:GetItem"]
      Resource = ["arn:aws:s3:::my-bucket/*", "arn:aws:dynamodb:*:*:table/my-table"]
    }]
  })
}
```

### Custom IAM Roles (Advanced)

For advanced use cases requiring custom trust policies or specific role configurations, you can disable automatic role creation and provide your own:

```hcl
module "my_service" {
  source = "./modules/aws/apprunner-regional-go-service"

  name = "my-service"
  team = "platform"

  # Disable automatic role creation
  create_service_role  = false
  create_instance_role = false

  # Provide your own roles
  service_role_arn  = aws_iam_role.custom_service.arn
  instance_role_arn = aws_iam_role.custom_instance.arn

  # ... rest of configuration
}
```

<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | n/a |
| <a name="provider_cosign"></a> [cosign](#provider\_cosign) | n/a |
| <a name="provider_ko"></a> [ko](#provider\_ko) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_apprunner_auto_scaling_configuration_version.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/apprunner_auto_scaling_configuration_version) | resource |
| [aws_apprunner_observability_configuration.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/apprunner_observability_configuration) | resource |
| [aws_apprunner_service.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/apprunner_service) | resource |
| [aws_ecr_repository.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecr_repository) | resource |
| [aws_iam_role.apprunner_instance](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role.apprunner_service](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.apprunner_instance_xray](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_iam_role_policy_attachment.apprunner_service_ecr](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [cosign_sign.this](https://registry.terraform.io/providers/chainguard-dev/cosign/latest/docs/resources/sign) | resource |
| [ko_build.this](https://registry.terraform.io/providers/ko-build/ko/latest/docs/resources/build) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_auto_deployments_enabled"></a> [auto\_deployments\_enabled](#input\_auto\_deployments\_enabled) | Enable automatic deployments when new image pushed to ECR | `bool` | `true` | no |
| <a name="input_autoscaling"></a> [autoscaling](#input\_autoscaling) | Autoscaling configuration for the service | <pre>object({<br/>    min_instances   = optional(number, 1)<br/>    max_instances   = optional(number, 25)<br/>    max_concurrency = optional(number, 100) # Concurrent requests per instance<br/>  })</pre> | <pre>{<br/>  "max_concurrency": 100,<br/>  "max_instances": 25,<br/>  "min_instances": 1<br/>}</pre> | no |
| <a name="input_container"></a> [container](#input\_container) | The container configuration for the service. App Runner supports one container per service. | <pre>object({<br/>    source = object({<br/>      base_image  = optional(string, "cgr.dev/chainguard/static:latest-glibc@sha256:a301031ffd4ed67f35ca7fa6cf3dad9937b5fa47d7493955a18d9b4ca5412d1a")<br/>      working_dir = string<br/>      importpath  = string<br/>      repo        = optional(string) # Override the default ko repository for this container<br/>      env         = optional(list(string), [])<br/>    })<br/>    args = optional(list(string), [])<br/>    port = optional(number, 8080)<br/>    env = optional(list(object({<br/>      name  = string<br/>      value = optional(string)<br/>    })), [])<br/>    # App Runner secrets from Secrets Manager or SSM Parameter Store<br/>    secrets = optional(list(object({<br/>      name  = string<br/>      value = string # ARN of the secret<br/>    })), [])<br/>    health_check = optional(object({<br/>      protocol            = optional(string, "TCP") # TCP or HTTP<br/>      path                = optional(string, "/")   # For HTTP health checks<br/>      interval            = optional(number, 5)     # Seconds between health checks<br/>      timeout             = optional(number, 2)     # Seconds to wait for response<br/>      healthy_threshold   = optional(number, 1)     # Consecutive successes needed<br/>      unhealthy_threshold = optional(number, 5)     # Consecutive failures needed<br/>    }))<br/>  })</pre> | n/a | yes |
| <a name="input_cpu"></a> [cpu](#input\_cpu) | The CPU units for the service. Valid values: 256 (0.25 vCPU), 512 (0.5 vCPU), 1024 (1 vCPU), 2048 (2 vCPU), 4096 (4 vCPU) | `number` | `1024` | no |
| <a name="input_create_ecr_repository"></a> [create\_ecr\_repository](#input\_create\_ecr\_repository) | Whether to create an ECR repository for the container images. Set to false if using an existing repository. | `bool` | `true` | no |
| <a name="input_create_instance_role"></a> [create\_instance\_role](#input\_create\_instance\_role) | Whether to create the IAM instance role for the running containers. Set to false to provide your own via instance\_role\_arn. | `bool` | `true` | no |
| <a name="input_create_service_role"></a> [create\_service\_role](#input\_create\_service\_role) | Whether to create the IAM service role for App Runner. Set to false to provide your own via service\_role\_arn. | `bool` | `true` | no |
| <a name="input_ecr_force_delete"></a> [ecr\_force\_delete](#input\_ecr\_force\_delete) | If true, will delete the ECR repository even if it contains images. Use with caution in production. | `bool` | `false` | no |
| <a name="input_ecr_repository_name"></a> [ecr\_repository\_name](#input\_ecr\_repository\_name) | Name of the ECR repository. If not provided, defaults to the service name. | `string` | `null` | no |
| <a name="input_egress"></a> [egress](#input\_egress) | Network egress configuration. DEFAULT for internet, VPC for private resources | `string` | `"DEFAULT"` | no |
| <a name="input_image_repository_type"></a> [image\_repository\_type](#input\_image\_repository\_type) | The type of image repository. ECR for private AWS ECR, ECR\_PUBLIC for public ECR | `string` | `"ECR"` | no |
| <a name="input_ingress"></a> [ingress](#input\_ingress) | Network ingress configuration. PUBLIC for internet access, PRIVATE for VPC only | `string` | `"PUBLIC"` | no |
| <a name="input_instance_role_arn"></a> [instance\_role\_arn](#input\_instance\_role\_arn) | The ARN of the IAM role that the running service will assume. Only required if create\_instance\_role is false. | `string` | `null` | no |
| <a name="input_memory"></a> [memory](#input\_memory) | The memory in MB for the service. Valid values: 512, 1024, 2048, 3072, 4096, 6144, 8192, 10240, 12288 | `number` | `2048` | no |
| <a name="input_name"></a> [name](#input\_name) | The name of the App Runner service | `string` | n/a | yes |
| <a name="input_observability_enabled"></a> [observability\_enabled](#input\_observability\_enabled) | Enable AWS X-Ray tracing | `bool` | `true` | no |
| <a name="input_product"></a> [product](#input\_product) | Product label to apply to resources | `string` | n/a | yes |
| <a name="input_service_role_arn"></a> [service\_role\_arn](#input\_service\_role\_arn) | The ARN of the IAM role that App Runner will use (for ECR access and CloudWatch logs). Only required if create\_service\_role is false. | `string` | `null` | no |
| <a name="input_tags"></a> [tags](#input\_tags) | Tags to apply to resources | `map(string)` | `{}` | no |
| <a name="input_team"></a> [team](#input\_team) | Team label to apply to resources | `string` | n/a | yes |
| <a name="input_vpc_connector_arn"></a> [vpc\_connector\_arn](#input\_vpc\_connector\_arn) | Optional VPC connector ARN for private resource access | `string` | `null` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_autoscaling_config_arn"></a> [autoscaling\_config\_arn](#output\_autoscaling\_config\_arn) | Auto-scaling configuration ARN |
| <a name="output_built_image"></a> [built\_image](#output\_built\_image) | Built and signed container image reference |
| <a name="output_ecr_repository_arn"></a> [ecr\_repository\_arn](#output\_ecr\_repository\_arn) | ECR repository ARN (if created by module) |
| <a name="output_ecr_repository_url"></a> [ecr\_repository\_url](#output\_ecr\_repository\_url) | ECR repository URL (if created by module) |
| <a name="output_instance_role_arn"></a> [instance\_role\_arn](#output\_instance\_role\_arn) | IAM instance role ARN used by the running containers |
| <a name="output_instance_role_name"></a> [instance\_role\_name](#output\_instance\_role\_name) | IAM instance role name (if created by module) |
| <a name="output_service_arn"></a> [service\_arn](#output\_service\_arn) | App Runner service ARN |
| <a name="output_service_id"></a> [service\_id](#output\_service\_id) | App Runner service ID |
| <a name="output_service_name"></a> [service\_name](#output\_service\_name) | App Runner service name |
| <a name="output_service_role_arn"></a> [service\_role\_arn](#output\_service\_role\_arn) | IAM service role ARN used by App Runner for ECR access and logs |
| <a name="output_service_role_name"></a> [service\_role\_name](#output\_service\_role\_name) | IAM service role name (if created by module) |
| <a name="output_service_status"></a> [service\_status](#output\_service\_status) | App Runner service status |
| <a name="output_service_url"></a> [service\_url](#output\_service\_url) | App Runner service URL |
<!-- END_TF_DOCS -->
