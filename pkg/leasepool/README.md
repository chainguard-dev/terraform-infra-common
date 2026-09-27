# Lease pools

A pool holds independent named leases. `Interface.TryAcquire` attempts once;
`ErrBusy` means another acquisition owns the name. Storage failures remain errors.
`gcs.New` coordinates separate processes through a shared bucket and prefix.
Prefixes are limited to 255 bytes so every valid lease name fits a GCS object name.
`memory.New` coordinates callers sharing the returned pool, using exactly the same
lease protocol over a mutex-protected store.

```go
pool, err := gcs.New(client.Bucket(bucket), "maintenance", leasepool.Config{})
if err != nil {
    return err
}
ctx, cancel := context.WithTimeout(ctx, time.Hour)
defer cancel()
err = leasepool.Run(ctx, pool, "cleanup", reconcileContinuously)
if errors.Is(err, leasepool.ErrBusy) {
    return nil // Another scheduled execution is performing this work.
}
return err
```

`Run` renews while the callback runs, cancels its context on ownership loss, and
releases after the callback returns. Normal parent cancellation is returned to the
caller, which decides whether a bounded tenure ending is success. The callback
must stop and join all protected work before returning. Direct acquisitions need
an explicit `Release` after work stops. Parent cancellation stops new renewals
but does not release early. Release lets an in-flight renewal finish within an
AcquireTimeout budget, then cancels a stalled RPC and joins it. The conditional
tombstone has a separate AcquireTimeout budget; it uses the last confirmed
generation, so an ambiguous committed renewal causes a conflict. The caller
context bounds both phases together. Run uses these internal bounds rather than
spending the tombstone budget on the drain. A successful no-op candidate does not
prove reconciliation is healthy: consumers should monitor actual successful
reconciliation progress.

Acquisition storage I/O has a five-second deadline by default, so candidates do
not accumulate during a storage outage. This deadline does not bound the lifetime
of an acquired lease.

Defaults: 30-second TTL, renewal every 10 seconds, and a 2-second clock-skew
allowance. All participants must agree on an upper bound on pairwise wall-clock
difference. Acquisition waits beyond the published expiration by that allowance;
ownership deadlines use elapsed local time starting **before** the write. Arbitrary
clock jumps or paused processes cannot be made safe by a cooperative context.

Every acquisition, renewal and release uses a generation-conditional write.
Released records remain as tombstones. The GCS client retries transient errors
on conditional writes within the operation context. A replay after an ambiguous
commit conflicts and fails closed. A terminal renewal error cancels ownership. A separate
watchdog cancels even when storage I/O stalls. An ambiguous acquisition returns an
error and may leave the name unavailable until expiration. An unambiguously confirmed acquisition that cannot be handed to the caller is
conditionally released with a bounded cleanup context. Cleanup never adopts
an unknown generation or overwrites a successor. An expired owner cannot renew
its ownership after the local deadline. Malformed records fail closed.

This is **not external-operation fencing**. A canceled context cannot revoke an
in-flight external API request or stop a suspended process from resuming. Callers
still need idempotent, recoverable operations and, where required, fencing enforced
at the mutation destination. Bucket IAM is the trust boundary; participants with
write access can overwrite records outside the protocol. Do not restore old lease
objects or delete active lease namespaces. Independent regional buckets provide
independent ownership, not cross-region failover coordination.

Run hermetic protocol and GCS adapter tests:

```sh
go test -race ./pkg/leasepool/...
```

Run the same public contract against GCS with ambient credentials and a dedicated
bucket (creates objects under a random namespace and deletes only that namespace):

```sh
LEASEPOOL_TEST_BUCKET=my-test-bucket go test -race ./pkg/leasepool/gcs -run TestRealGCS
```

The HTTP adapter tests exercise the real storage client against a narrow GCS wire
model. They are not a substitute for the opt-in real-service contract test.
