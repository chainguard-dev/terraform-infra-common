/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sethvargo/go-envconfig"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	// https://cloud.google.com/run/docs/container-contract#services-env-vars
	KnativeServiceName  string `env:"K_SERVICE, default=unknown"`
	KnativeRevisionName string `env:"K_REVISION, default=unknown"`
}{})

var (
	// TODO(mattmoor): Inspiration:
	// https://pkg.go.dev/k8s.io/client-go/util/workqueue#MetricsProvider

	mInProgressKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_in_progress_keys",
			Help: "The number of keys currently being processed by this workqueue.",
		},
		[]string{"service_name", "revision_name"},
	)
	mQueuedKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_queued_keys",
			Help: "The number of keys currently in the backlog of this workqueue.",
		},
		[]string{"service_name", "revision_name"},
	)
	mNotBeforeKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_notbefore_keys",
			Help: "The number of keys waiting on a 'not before' in the backlog of this workqueue.",
		},
		[]string{"service_name", "revision_name"},
	)
	mMaxAttempts = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_max_attempts",
			Help: "The maximum number of attempts for any queued or in-progress task.",
		},
		[]string{"service_name", "revision_name"},
	)
	mTaskMaxAttempts = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_task_max_attempts",
			Help: "The maximum number of attempts for a given task above 20.",
		},
		[]string{"service_name", "revision_name", "task_id"},
	)
	mWorkLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workqueue_process_latency_seconds",
			Help:    "The duration taken to process a key.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60, 120, 240, 480, 960, 3600, 7200},
		},
		[]string{"service_name", "revision_name", "priority_class"},
	)
	mWaitLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workqueue_wait_latency_seconds",
			Help:    "The duration the key waited to start.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60, 120, 240, 480, 960, 3600, 7200},
		},
		[]string{"service_name", "revision_name", "priority_class"},
	)
	mWaitLatencyFromScheduled = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workqueue_wait_latency_from_scheduled_seconds",
			Help:    "The duration the key waited to start from its scheduled time.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60, 120, 240, 480, 960, 3600, 7200},
		},
		[]string{"service_name", "revision_name", "priority_class"},
	)
	mAddedKeys = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workqueue_added_keys",
			Help: "The total number of queue requests.",
		},
		[]string{"service_name", "revision_name"},
	)
	mDedupedKeys = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workqueue_deduped_keys",
			Help: "The total number of keys that were deduped.",
		},
		[]string{"service_name", "revision_name"},
	)
	mCompletionAttempts = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workqueue_attempts_at_completion",
			Help:    "The number of attempts for successfully completed tasks",
			Buckets: []float64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024},
		},
		[]string{"service_name", "revision_name"},
	)
	mDeadLetteredKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_dead_lettered_keys",
			Help: "The number of keys currently in the dead letter queue",
		},
		[]string{"service_name", "revision_name"},
	)
	mTimeToCompletion = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workqueue_time_to_completion_seconds",
			Help:    "The time from first queue to final outcome (success or dead-letter). The metric captures the full lifecycle duration including all retry attempts and backoff delays.",
			Buckets: []float64{5, 10, 20, 30, 45, 60, 120, 240, 480, 960, 3600 /* 1h */, 7200 /* 2h */, 14400 /* 4h */, 28800 /* 8h */, 43200 /* 12h */, 86400 /* 1d */, 172800 /* 2d */, 259200 /* 3d */},
		},
		[]string{"service_name", "revision_name", "priority_class", "status"},
	)
)

// priorityClass converts a priority value to a priority class label.
func priorityClass(priority int64) string {
	return fmt.Sprintf("%dxx", priority/100)
}
