# GCS Workqueue Implementation

This package implements a Google Cloud Storage (GCS) backed workqueue that provides reliable, persistent task processing with state management.

## Bucket Organization

The GCS workqueue uses object prefixes to organize tasks by their state within a single bucket:

### Prefixes

- **`queued/`** - Tasks waiting to be processed
- **`in-progress/`** - Tasks currently being processed by a worker
- **`dead-letter/`** - Tasks that have failed after exceeding maximum retry attempts

### State Transitions

```
queued/{key} → in-progress/{key} → [completed] (deleted)
                      ↓
                 dead-letter/{key} (on failure)
                      ↑
                 queued/{key} (on requeue)
```

## Object Metadata

Each object stores metadata to track task state:

- **`priority`** - Zero-padded 8-digit priority for lexicographic ordering (higher = processed first)
- **`attempts`** - Number of processing attempts
- **`lease-expiration`** - When the current lease expires (for in-progress tasks)
- **`not-before`** - Earliest time the task should be processed (RFC3339 format)
- **`failed-time`** - When the task was moved to dead letter queue (RFC3339 format)
- **`last-attempted`** - Unix timestamp of last processing attempt

## Key Features

- **Priority-based processing** - Higher priority tasks processed first
- **Lease-based ownership** - In-progress tasks have renewable leases to prevent multiple workers processing the same task
- **Automatic retry with backoff** - Failed tasks automatically requeued with exponential backoff
- **Dead letter handling** - Tasks exceeding retry limits moved to dead letter queue
- **Orphan detection** - Detects and handles tasks with expired leases
- **Deduplication** - Duplicate queue requests update priority/timing instead of creating duplicates

## Metrics

The implementation exports Prometheus metrics for:

- Queue sizes (queued, in-progress, dead-lettered)
- Processing latency and wait times
- Retry attempts and completion rates
- Task priorities and attempt distributions
