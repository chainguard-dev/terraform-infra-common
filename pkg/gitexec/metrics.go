/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	outcomeSuccess = "success"
	outcomeFailure = "failure"
)

var (
	operationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "git_operations_total",
		Help: "Total local git operations executed, labeled by subcommand and outcome.",
	}, []string{"op", "outcome"})

	operationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "git_operation_duration_seconds",
		Help:    "Wall-clock duration of local git operations, in seconds.",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
	}, []string{"op"})
)
