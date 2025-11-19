/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"net/http"
	"sync"

	"github.com/chainguard-dev/terraform-infra-common/pkg/workqueue"
)

func Handler(wq workqueue.Interface, concurrency, batchSize int, f Callback, maxRetry int) http.Handler {
	return &handler{
		wq:          wq,
		concurrency: concurrency,
		batchSize:   batchSize,
		f:           f,
		maxRetry:    maxRetry,
	}
}

type handler struct {
	doWork sync.Mutex
	doWait sync.Mutex

	wq          workqueue.Interface
	concurrency int
	batchSize   int
	f           Callback
	maxRetry    int
}

var _ http.Handler = (*handler)(nil)

// ServeHTTP implements http.Handler
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// If something else is already dispatching, then we enter the waiting room.
	if !h.doWork.TryLock() {
		// If there is already someone in the waiting room, then we deduplicate
		// ourselves with that event by returning and allowing the waiting
		// request to trigger any subsequent processing.
		if !h.doWait.TryLock() {
			w.WriteHeader(http.StatusOK)
			return
		}
		// As the waiting room occupant, wait patiently to acquire the work lock
		// and once we have it, then we can vacate the waiting room.
		// Note: it is conceivably possible for two races to occur:
		// 1. A new request acquires the lock instead of us, and we continue to
		//   occupy the waiting room.
		// 2. A new request comes in while we hold both locks, and nothing fills
		//   the waiting room.
		// In both of these cases, there isn't really a risk of data loss, so
		// this is acceptable.
		h.doWork.Lock()
		h.doWait.Unlock()
	}

	// Launch the dispatch future while holding the lock.
	future := func() Future {
		// We do this via a defer to ensure that it is invoked in the event of
		// a panic during HandleAsync.
		defer h.doWork.Unlock()
		return HandleAsync(r.Context(), h.wq, h.concurrency, h.batchSize, h.f, h.maxRetry)
	}()

	// Once we have initiated the dispatch, allow other dispatches to
	// initiate while we wait on this round of results.
	if err := future(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
