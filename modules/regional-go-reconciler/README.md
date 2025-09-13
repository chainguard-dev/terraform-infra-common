# Regional Go Reconciler Module

This module combines a workqueue with a regional Go service to create a complete reconciler setup. It stands up both the workqueue infrastructure (receiver and dispatcher) and a reconciler service that processes events from the workqueue.

## Usage

```hcl
module "my-reconciler" {
  source = "chainguard-dev/terraform-infra-common//modules/regional-go-reconciler"

  project_id = var.project_id
  name       = "my-reconciler"
  regions    = var.regions

  service_account = google_service_account.reconciler.email

  containers = {
    "reconciler" = {
      source = {
        working_dir = path.module
        importpath  = "./cmd/reconciler"
      }
      ports = [{
        container_port = 8080
      }]
    }
  }

  notification_channels = [var.notification_channel]
}
```

## Features

- **Integrated Workqueue**: Automatically sets up a workqueue with receiver and dispatcher services
- **Regional Deployment**: Deploys reconciler services in multiple regions with workqueue infrastructure
- **Flexible Configuration**: Supports both regional and global workqueue scopes
- **Built from Source**: Uses ko to build Go binaries directly from source
- **Monitoring**: Includes dashboards and metrics for both workqueue and service

## Architecture

The module creates:
1. A workqueue (using the `workqueue` module) with:
   - Receiver service (`${name}-rcv`) that accepts events
   - Dispatcher service (`${name}-dsp`) that processes the queue
   - GCS buckets for queue storage
   - Pub/Sub topics and subscriptions
2. A reconciler service (using the `regional-go-service` module) that:
   - Receives events from the dispatcher
   - Processes them according to your business logic
   - Can be configured with custom containers and environment variables

## Workqueue Service Protocol

Your reconciler should implement the workqueue proto service:

```go
type Reconciler struct {
    workqueue.UnimplementedWorkqueueServiceServer
    // your fields
}

func (r *Reconciler) Process(ctx context.Context, req *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
    // Process the event
    err := r.reconcile(ctx, req.Key)
    if err != nil {
        // Requeue with delay
        if delay, ok := workqueue.GetRequeueDelay(err); ok {
            return &workqueue.ProcessResponse{
                RequeueAfterSeconds: int64(delay.Seconds()),
            }, nil
        }
        // Non-retriable error
        if workqueue.IsNonRetriableError(err) {
            return nil, status.Error(codes.InvalidArgument, err.Error())
        }
        // Retriable error
        return nil, err
    }
    return &workqueue.ProcessResponse{}, nil
}
```

## Monitoring

A dedicated dashboard module is available for monitoring:

```hcl
module "my-reconciler-dashboard" {
  source = "chainguard-dev/terraform-infra-common//modules/dashboard/workqueue"

  name       = "my-reconciler"
  project_id = var.project_id

  max_retry       = module.my-reconciler.max_retry
  concurrent_work = module.my-reconciler.concurrent_work
  scope           = module.my-reconciler.scope
}
```

## Variables

See [variables.tf](./variables.tf) for all available configuration options.

## Outputs

| Name | Description |
|------|-------------|
| `workqueue-receiver` | The name of the workqueue receiver service |
| `reconciler-uri` | The URI of the reconciler service |
| `workqueue-topic` | The workqueue topic name |
| `workqueue-dashboards` | Dashboard outputs for monitoring |