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

Log and ReadContainer keep no shared state. All functions are safe for concurrent use.

`memusage.ReadContainer()` returns the container cgroup's current usage, limit,
lifetime peak, and the `memory.stat` split of current usage (`Anon`, `Shmem`,
`File`, `InactiveFile`) for callers that sample memory themselves. It uses the same
bounded, verified reads: each value is a `*uint64` with its own error, and a
missing, malformed, or unlimited (`max`) value is nil with an error, never zero.

## Heartbeat

`memusage.Heartbeat(ctx)` samples the container cgroup every second and logs a
`memory heartbeat` entry while the working set is at or over half the limit: once
a minute, and at once whenever the working set grows by 1/32 of the limit over
its lowest sample since the last entry, so a climb that follows a drop still
logs. Growth entries are capped at 16 a minute, the most steps a climb from half
to the limit can take, so an instance logs at most 17 entries a minute even when
its usage churns. A climb is logged in full unless churn earlier in the same
minute spent the cap; then it waits for the next once-a-minute entry, which
carries the peak. Below half it logs nothing. The working set is
current usage minus inactive page cache, as the kubelet counts it, so a service
that only fills the page cache stays quiet. When it cannot read the usage or
the limit, it logs one `memory heartbeat unavailable` entry with the errors.
Only the first call in a process samples; later calls return at once.

`httpmetrics.SetupMetrics` and `httpmetrics.ServeMetrics` start it, so most
services get it without a code change.

Cloud Run SIGKILLs an instance that reaches its memory limit. When the memory
is files in the in-memory `/tmp` or child processes, it logs no out-of-memory
event, and these entries are the only record of what filled memory.

| Field | Meaning |
| --- | --- |
| `cgroup_current_bytes`, `cgroup_limit_bytes`, `cgroup_peak_bytes` | `Current`, `Limit`, and `Peak` from `ReadContainer`. |
| `cgroup_working_set_bytes` | `cgroup_current_bytes` minus `cgroup_inactive_file_bytes`; the value compared with the limit. |
| `cgroup_anon_bytes` | Anonymous memory of every process in the container: the Go heap and child processes such as `git`. |
| `cgroup_shmem_bytes` | Files in tmpfs mounts, such as Cloud Run's `/tmp`. |
| `cgroup_file_bytes` | Page cache, including `cgroup_shmem_bytes`. |
| `cgroup_inactive_file_bytes` | Page cache the kernel reclaims first under pressure. |
| `cgroup_*_error` | Why the matching value is null, or null when it was read. |
| `go_heap_alloc_bytes`, `go_sys_bytes`, `goroutines` | The Go runtime's own view, for comparison with `cgroup_anon_bytes`. |
