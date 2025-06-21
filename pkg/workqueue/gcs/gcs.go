/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/chainguard-dev/clog"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

// ClientInterface is an interface that abstracts the GCS client.
type ClientInterface interface {
	Object(name string) *storage.ObjectHandle
	Objects(ctx context.Context, q *storage.Query) *storage.ObjectIterator
}

// NewWorkQueue creates a new GCS-backed workqueue.
func NewWorkQueue(client ClientInterface, limit int) workqueue.Interface {
	return &wq{
		client: client,
		limit:  limit,
	}
}

type wq struct {
	client ClientInterface
	limit  int
}

var _ workqueue.Interface = (*wq)(nil)

// RefreshInterval is the period on which we refresh the lease of owned objects
// It is surfaced as a global, so that it can be mutated by tests and exposed as
// a flag by binaries wrapping this library.  However, binary authors should use
// caution to pass consistent values to the key ingress and dispatchers, or they
// may see unexpected behavior.
// TODO(mattmoor): What's the right balance here?
var RefreshInterval = 5 * time.Minute

const (
	queuedPrefix          = "queued/"
	inProgressPrefix      = "in-progress/"
	deadLetterPrefix      = "dead-letter/"
	expirationMetadataKey = "lease-expiration"
	attemptsMetadataKey   = "attempts"
	priorityMetadataKey   = "priority"
	notBeforeMetadataKey  = "not-before"
	failedTimeMetadataKey = "failed-time"
)

var noPriority = fmt.Sprintf("%08d", 0)
var noNotBefore = time.Time{}.UTC().Format(time.RFC3339)

// Queue implements workqueue.Interface.
func (w *wq) Queue(ctx context.Context, key string, opts workqueue.Options) error {
	writer := w.client.Object(fmt.Sprintf("%s%s", queuedPrefix, key)).If(storage.Conditions{
		DoesNotExist: true,
	}).NewWriter(ctx)

	writer.Metadata = map[string]string{
		// Zero-pad the priority to 8 digits to ensure lexicographic ordering,
		// so that we don't have to parse it to order things.
		priorityMetadataKey: fmt.Sprintf("%08d", opts.Priority),
	}
	writer.Metadata[notBeforeMetadataKey] = opts.NotBefore.UTC().Format(time.RFC3339)

	mAddedKeys.With(prometheus.Labels{
		"service_name":  env.KnativeServiceName,
		"revision_name": env.KnativeRevisionName,
	}).Add(1)

	if _, err := writer.Write([]byte("")); err != nil {
		return fmt.Errorf("Write() = %w", err)
	}
	if exists, err := checkPreconditionFailedOk(writer.Close()); err != nil {
		return fmt.Errorf("Close() = %w", err)
	} else if exists {
		clog.DebugContextf(ctx, "Key %q already exists", key)
		mDedupedKeys.With(prometheus.Labels{
			"service_name":  env.KnativeServiceName,
			"revision_name": env.KnativeRevisionName,
		}).Add(1)

		if err := updateMetadata(ctx, w.client, key, writer.Metadata); err != nil {
			if errors.Is(err, storage.ErrObjectNotExist) {
				clog.InfoContextf(ctx, "Key %q was deleted before we could fetch the duplicate, recursing.", key)
				return w.Queue(ctx, key, opts)
			}
			return fmt.Errorf("updateMetadata() = %w", err)
		}
	}
	return nil
}

func updateMetadata(ctx context.Context, client ClientInterface, key string, metadata map[string]string) error {
	switch {
	case metadata[priorityMetadataKey] != noPriority:
		// If the priority was set, then attempt to merge it with the queued
		// key.
		break

	case metadata[notBeforeMetadataKey] != noNotBefore:
		// If the not before was set, then attempt to merge it with the queued
		// key.  This is largely for Queue operations, as the NotBefore is
		// cleared when we start processing the key.
		break

	default:
		// No options, so don't bother fetching the queued object.
		return nil
	}

	attrs, err := client.Object(fmt.Sprintf("%s%s", queuedPrefix, key)).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("Attrs() = %w", err)
	}
	// Inialialize the metadata map if it's nil.
	if attrs.Metadata == nil {
		attrs.Metadata = make(map[string]string, 2)
	}
	update := false
	if p, ok := attrs.Metadata[priorityMetadataKey]; !ok || p < metadata[priorityMetadataKey] {
		clog.InfoContextf(ctx, "Updating %s priority from %q to %q", key, p, metadata[priorityMetadataKey])
		attrs.Metadata[priorityMetadataKey] = metadata[priorityMetadataKey]
		update = true
	}
	if ts, ok := attrs.Metadata[notBeforeMetadataKey]; ok && ts < metadata[notBeforeMetadataKey] {
		clog.InfoContextf(ctx, "Updating %s not-before from %q to %q", key, ts, metadata[notBeforeMetadataKey])
		attrs.Metadata[notBeforeMetadataKey] = metadata[notBeforeMetadataKey]
		update = true
	}
	if update {
		if _, err := client.Object(fmt.Sprintf("%s%s", queuedPrefix, key)).Update(ctx, storage.ObjectAttrsToUpdate{
			Metadata: attrs.Metadata,
		}); err != nil {
			return fmt.Errorf("Update() = %w", err)
		}
	}
	return nil
}

func checkPreconditionFailedOk(err error) (bool, error) {
	// No error is OK.
	if err == nil {
		return false, nil
	}
	// If the error is a googleapi.Error, and it's a PreconditionFailed,
	// then it's OK.
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		if gerr.Code == http.StatusPreconditionFailed {
			return true, nil
		}
	}
	return false, err
}

// Enumerate implements workqueue.Interface.
func (w *wq) Enumerate(ctx context.Context) ([]workqueue.ObservedInProgressKey, []workqueue.QueuedKey, error) {
	iter := w.client.Objects(ctx, nil)

	wip := make([]workqueue.ObservedInProgressKey, 0, w.limit)
	qd := make([]*queuedKey, 0, w.limit+1)

	queued, notbefore, deadlettered := 0, 0, 0
	maxAttempts := 0 // Track the maximum number of attempts
	for {
		objAttrs, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		} else if err != nil {
			return nil, nil, fmt.Errorf("Next() = %w", err)
		}
		var priority int64
		if p, ok := objAttrs.Metadata[priorityMetadataKey]; ok {
			priority, err = strconv.ParseInt(p, 10, 64)
			if err != nil {
				clog.WarnContextf(ctx, "Failed to parse priority: %v", err)
			}
		}
		// Only check for max attempts if this is not a deadlettered item
		if !strings.HasPrefix(objAttrs.Name, deadLetterPrefix) {
			// Check for attempts and track maximum
			if att, ok := objAttrs.Metadata[attemptsMetadataKey]; ok && att != "" {
				attempts, err := strconv.Atoi(att)
				if err != nil {
					clog.WarnContextf(ctx, "Failed to parse attempts: %v", err)
				} else if attempts > maxAttempts {
					maxAttempts = attempts
				}
			}
		}

		switch {
		case strings.HasPrefix(objAttrs.Name, inProgressPrefix):
			wip = append(wip, &inProgressKey{
				client:   w.client,
				attrs:    objAttrs,
				priority: priority,
			})

		case strings.HasPrefix(objAttrs.Name, queuedPrefix):
			if nbf, ok := objAttrs.Metadata[notBeforeMetadataKey]; ok && nbf != "" {
				if notBefore, err := time.Parse(time.RFC3339, nbf); err != nil {
					clog.WarnContextf(ctx, "Failed to parse not-before: %v", err)
				} else if time.Now().UTC().Before(notBefore) {
					clog.InfoContextf(ctx, "Skipping key %q until %v", objAttrs.Name, notBefore)
					notbefore++
					continue
				}
			}

			qd = append(qd, &queuedKey{
				client:   w.client,
				attrs:    objAttrs,
				priority: priority,
			})
			sort.Slice(qd, func(i, j int) bool {
				if lhs, rhs := qd[i].Priority(), qd[j].Priority(); lhs != rhs {
					// First consider priority.
					return lhs > rhs
				}
				if !qd[i].attrs.Created.Equal(qd[j].attrs.Created) {
					return qd[i].attrs.Created.Before(qd[j].attrs.Created)
				}
				return qd[i].attrs.Name < qd[j].attrs.Name
			})
			if len(qd) > w.limit {
				qd = qd[:w.limit]
			}
			queued++

		case strings.HasPrefix(objAttrs.Name, deadLetterPrefix):
			// Count the dead-lettered keys
			deadlettered++
		}
	}

	qk := make([]workqueue.QueuedKey, 0, len(qd))
	for _, qi := range qd {
		qk = append(qk, qi)
	}

	// Set all metrics
	labels := prometheus.Labels{
		"service_name":  env.KnativeServiceName,
		"revision_name": env.KnativeRevisionName,
	}
	mInProgressKeys.With(labels).Set(float64(len(wip)))
	mQueuedKeys.With(labels).Set(float64(queued))
	mNotBeforeKeys.With(labels).Set(float64(notbefore))
	mDeadLetteredKeys.With(labels).Set(float64(deadlettered))
	// Set the max attempts metric
	mMaxAttempts.With(labels).Set(float64(maxAttempts))
	return wip, qk, nil
}

type inProgressKey struct {
	client      ClientInterface
	ownerCtx    context.Context
	ownerCancel context.CancelFunc

	priority int64

	// Once we start to heartbeat things, then that thread may update attrs,
	// so use the RWMutex to protect it from concurrent access.
	rw    sync.RWMutex
	attrs *storage.ObjectAttrs
}

var _ workqueue.ObservedInProgressKey = (*inProgressKey)(nil)
var _ workqueue.OwnedInProgressKey = (*inProgressKey)(nil)

// Name implements workqueue.Key.
func (o *inProgressKey) Name() string {
	o.rw.RLock()
	defer o.rw.RUnlock()
	return strings.TrimPrefix(o.attrs.Name, inProgressPrefix)
}

// Priority implements workqueue.Key.
func (o *inProgressKey) Priority() int64 {
	return o.priority
}

// GetAttempts implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) GetAttempts() int {
	o.rw.RLock()
	defer o.rw.RUnlock()

	if o.attrs == nil || o.attrs.Metadata == nil {
		return 0
	}

	attempts, err := strconv.Atoi(o.attrs.Metadata[attemptsMetadataKey])
	if err != nil {
		clog.WarnContextf(o.ownerCtx, "Malformed attempts on %s: %v", o.Name(), err)
		return 0
	}
	return attempts
}

// Requeue implements workqueue.InProgressKey.
func (o *inProgressKey) Requeue(ctx context.Context) error {
	// Use RequeueWithOptions with an empty options struct to get default behavior
	return o.RequeueWithOptions(ctx, workqueue.Options{})
}

// RequeueWithOptions implements workqueue.InProgressKey.
func (o *inProgressKey) RequeueWithOptions(ctx context.Context, opts workqueue.Options) error {
	if o.ownerCancel != nil {
		o.ownerCancel()
	}
	o.rw.RLock()
	defer o.rw.RUnlock()

	// We'll move from the in-progress to the queued prefix.
	key := strings.TrimPrefix(o.attrs.Name, inProgressPrefix)
	copier := o.client.Object(fmt.Sprintf("%s%s", queuedPrefix, key)).If(storage.Conditions{
		DoesNotExist: true,
	}).CopierFrom(o.client.Object(o.attrs.Name))

	// Preserve metadata
	copier.Metadata = o.attrs.Metadata
	if copier.Metadata == nil {
		copier.Metadata = make(map[string]string)
	}
	// Clear the lease expiration when copying the object back to avoid
	// confusion since the object is no longer in progress.
	delete(copier.Metadata, expirationMetadataKey)

	// Handle custom delay if specified
	if opts.Delay > 0 {
		notBefore := time.Now().UTC().Add(opts.Delay)
		copier.Metadata[notBeforeMetadataKey] = notBefore.Format(time.RFC3339)
	} else if p, ok := copier.Metadata[priorityMetadataKey]; ok && p != noPriority {
		// If no custom delay and priority is set, use the standard backoff
		attempts, err := strconv.Atoi(copier.Metadata[attemptsMetadataKey])
		if err != nil {
			clog.WarnContextf(ctx, "Malformed attempts on %s: %v", key, err)
			attempts = 1
		}
		backoffDelay := time.Duration(attempts * int(workqueue.BackoffPeriod))
		if backoffDelay > workqueue.MaximumBackoffPeriod {
			backoffDelay = workqueue.MaximumBackoffPeriod
		}
		copier.Metadata[notBeforeMetadataKey] = time.Now().UTC().Add(backoffDelay).Format(time.RFC3339)
	}

	// Update priority if specified
	if opts.Priority != 0 {
		copier.Metadata[priorityMetadataKey] = strconv.FormatInt(opts.Priority, 10)
	}

	_, err := copier.Run(ctx)
	if exists, err := checkPreconditionFailedOk(err); err != nil {
		return fmt.Errorf("Run() = %w", err)
	} else if exists {
		if err := updateMetadata(ctx, o.client, key, copier.Metadata); err != nil {
			if errors.Is(err, storage.ErrObjectNotExist) {
				clog.InfoContextf(ctx, "Key %q was deleted before we could fetch the duplicate, recursing.", key)
				return o.RequeueWithOptions(ctx, opts)
			}
			return fmt.Errorf("updateMetadata() = %w", err)
		}
	}
	return o.client.Object(o.attrs.Name).Delete(ctx)
}

// IsOrphaned implements workqueue.ObservedInProgressKey.
func (o *inProgressKey) IsOrphaned() bool {
	o.rw.RLock()
	defer o.rw.RUnlock()

	exp, ok := o.attrs.Metadata[expirationMetadataKey]
	if !ok {
		// No expiration metadata should be treated as orphaned.
		return true
	}
	expiry, err := time.Parse(time.RFC3339, exp)
	if err != nil {
		// Malformed expiration metadata should be treated as orphaned.
		return true
	}

	// If the expiration time is in the past, then this key is orphaned.
	return time.Now().UTC().After(expiry)
}

// deadLetterKey returns the dead letter key for this in-progress key
func (o *inProgressKey) deadLetterKey() string {
	key := strings.TrimPrefix(o.attrs.Name, inProgressPrefix)
	return fmt.Sprintf("%s%s", deadLetterPrefix, key)
}

// Complete implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) Complete(ctx context.Context) error {
	o.ownerCancel()
	o.rw.RLock()
	defer o.rw.RUnlock()

	mWorkLatency.With(prometheus.Labels{
		"service_name":  env.KnativeServiceName,
		"revision_name": env.KnativeRevisionName,
	}).Observe(time.Now().UTC().Sub(o.attrs.Created).Seconds())

	// Best-effort delete of the dead-letter object, if it exists.
	deadLetterKey := o.deadLetterKey()
	if err := o.client.Object(deadLetterKey).Delete(ctx); err != nil {
		if !errors.Is(err, storage.ErrObjectNotExist) {
			clog.WarnContextf(ctx, "Failed to delete dead-letter object %q: %v", deadLetterKey, err)
		}
	}

	return o.client.Object(o.attrs.Name).Delete(ctx)
}

// Deadletter implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) Deadletter(ctx context.Context) error {
	if o.ownerCancel != nil {
		o.ownerCancel()
	}
	o.rw.RLock()
	defer o.rw.RUnlock()

	key := strings.TrimPrefix(o.attrs.Name, inProgressPrefix)
	deadLetterKey := o.deadLetterKey()

	clog.InfoContextf(ctx, "Moving key %q to dead letter queue as %q", key, deadLetterKey)

	// Copy the in-progress task to the dead letter queue
	copier := o.client.Object(deadLetterKey).CopierFrom(o.client.Object(o.attrs.Name))

	// Preserve metadata
	copier.Metadata = o.attrs.Metadata
	if copier.Metadata == nil {
		copier.Metadata = make(map[string]string)
	}

	// Clear the lease expiration when copying the object
	delete(copier.Metadata, expirationMetadataKey)

	// Add metadata about when the key was dead-lettered
	copier.Metadata[failedTimeMetadataKey] = time.Now().UTC().Format(time.RFC3339)

	// Create the dead letter entry
	_, err := copier.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to create dead letter entry: %w", err)
	}

	// Delete the in-progress task
	return o.client.Object(o.attrs.Name).Delete(ctx)
}

// Context implements workqueue.OwnedInProgressKey.
func (o *inProgressKey) Context() context.Context {
	return o.ownerCtx
}

func (o *inProgressKey) startHeartbeat(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	o.ownerCtx = ctx
	o.ownerCancel = cancel

	go func() {
		ticker := time.NewTicker(RefreshInterval)
		defer ticker.Stop()
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				// The function invocation is to scope the defer
				if err := func() error {
					o.rw.Lock()
					defer o.rw.Unlock()

					attrs, err := o.client.Object(o.attrs.Name).If(storage.Conditions{
						// We are the only ones that should be updating the object,
						// so if we see anything manipulate the object, then assume
						// that we've lost ownership and cancel the context to
						// terminate the in-progress work.
						MetagenerationMatch: o.attrs.Metageneration,
					}).Update(ctx, storage.ObjectAttrsToUpdate{
						Metadata: map[string]string{
							expirationMetadataKey: time.Now().UTC().Add(3 * RefreshInterval).Format(time.RFC3339),
						},
					})
					if err != nil {
						return err
					}
					// This is what we're guarding with the write lock.
					o.attrs = attrs
					return nil
				}(); err != nil {
					clog.ErrorContextf(ctx, "Failed to update expiration: %v", err)
					return
				}
			}
		}
	}()
}

type queuedKey struct {
	client   ClientInterface
	attrs    *storage.ObjectAttrs
	priority int64
}

var _ workqueue.QueuedKey = (*queuedKey)(nil)

// Name implements workqueue.Key.
func (q *queuedKey) Name() string {
	return strings.TrimPrefix(q.attrs.Name, queuedPrefix)
}

// Priority implements workqueue.Key.
func (q *queuedKey) Priority() int64 {
	return q.priority
}

// Start implements workqueue.QueuedKey.
func (q *queuedKey) Start(ctx context.Context) (workqueue.OwnedInProgressKey, error) {
	// We'll move from the in-progress to the queued prefix.
	srcObject := q.attrs.Name
	key := strings.TrimPrefix(srcObject, queuedPrefix)
	targetObject := fmt.Sprintf("%s%s", inProgressPrefix, key)

	mWaitLatency.With(prometheus.Labels{
		"service_name":  env.KnativeServiceName,
		"revision_name": env.KnativeRevisionName,
	}).Observe(time.Now().UTC().Sub(q.attrs.Created).Seconds())

	// Create a copier to copy the source object, and then we will delete it
	// upon success.
	copier := q.client.Object(targetObject).If(storage.Conditions{
		DoesNotExist: true,
	}).CopierFrom(q.client.Object(srcObject))

	// Preserve metadata
	copier.Metadata = q.attrs.Metadata
	if copier.Metadata == nil {
		copier.Metadata = make(map[string]string, 2)
	}
	// Set the expiration metadata to 3x the refresh interval.
	copier.Metadata[expirationMetadataKey] = time.Now().UTC().Add(3 * RefreshInterval).Format(time.RFC3339)
	if att, ok := copier.Metadata[attemptsMetadataKey]; ok {
		prevAttempts, err := strconv.Atoi(att)
		if err != nil {
			clog.ErrorContextf(ctx, "Malformed attempts on %s: %v", srcObject, err)
			copier.Metadata[attemptsMetadataKey] = "1"
		} else {
			copier.Metadata[attemptsMetadataKey] = fmt.Sprint(prevAttempts + 1)
		}
	} else {
		copier.Metadata[attemptsMetadataKey] = "1"
	}
	// Never persist the not-before metadata to a running task.
	// We set it to the zero value instead of deleting it so that we can assume
	// the invariant that this key is always present and date-formatted.
	copier.Metadata[notBeforeMetadataKey] = noNotBefore

	attrs, err := copier.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("Run() = %w", err)
	}
	if err := q.client.Object(srcObject).Delete(ctx); err != nil {
		return nil, fmt.Errorf("Delete() = %w", err)
	}

	oip := &inProgressKey{
		client:   q.client,
		attrs:    attrs,
		priority: q.priority,
	}

	// start a process to heartbeat things, and set up a context that we can
	// cancel if the heartbeat observes a loss in ownership.
	oip.startHeartbeat(ctx)

	return oip, nil
}
