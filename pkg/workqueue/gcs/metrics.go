/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"

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
	mWorkLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workqueue_process_latency_seconds",
			Help:    "The duration taken to process a key.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60, 120, 240, 480, 960},
		},
		[]string{"service_name", "revision_name"},
	)
	mWaitLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workqueue_wait_latency_seconds",
			Help:    "The duration the key waited to start.",
			Buckets: []float64{.25, .5, 1, 2.5, 5, 10, 20, 30, 45, 60, 120, 240, 480, 960},
		},
		[]string{"service_name", "revision_name"},
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
)
