# memusage

`memusage.Log(ctx)` records kernel memory high-water marks through the context's
`clog` logger. Call it when an operation finishes to capture peaks even when the
operation completes between monitoring samples:

```go
import "github.com/chainguard-dev/terraform-infra-common/pkg/memusage"

func run(ctx context.Context) error {
    defer memusage.Log(ctx)
    // Perform the operation.
    return nil
}
```

The `memory usage` log entry contains:

| Field | Meaning |
| --- | --- |
| `process_peak_rss_bytes` | Linux `/proc/self/status` `VmHWM`, converted from KiB to bytes. Excludes child processes and unmapped scratch files. |
| `process_peak_rss_error` | Error reading or parsing process accounting, or null on success. |
| `container_peak_memory_bytes` | Cgroup v2 `memory.peak`, falling back to v1 `memory.max_usage_in_bytes` when v2 accounting is absent. Includes charged scratch files and child processes. |
| `container_peak_memory_source` | Accounting file path relative to `/`, or empty if no supported controller was found. |
| `container_peak_memory_error` | Error reading or verifying container accounting, or null on success. |

Each file read is bounded. The reader verifies current-process membership and the
controller mount root; nested or unverified layouts remain unknown instead of
reporting a parent's peak. Missing, malformed, or unsupported accounting is logged
as null with an error, never as zero. Non-Linux platforms may report neither value.

These counters describe the process and cgroup lifetimes, respectively, rather
than one operation. Calls do not reset the counters. A hard kill or OOM can prevent
a deferred log, so continue checking execution status and monitoring metrics.

The package keeps no shared state and is safe for concurrent use.
