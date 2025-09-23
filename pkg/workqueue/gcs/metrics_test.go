/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"fmt"
	"testing"
)

func TestPriorityClass(t *testing.T) {
	tests := []struct {
		priority int64
		want     string
	}{
		{priority: 0, want: "0xx"},
		{priority: 1, want: "0xx"},
		{priority: 99, want: "0xx"},
		{priority: 100, want: "1xx"},
		{priority: 199, want: "1xx"},
		{priority: 200, want: "2xx"},
		{priority: 999, want: "9xx"},
		{priority: 1000, want: "10xx"},
		{priority: -1, want: "0xx"},
		{priority: -100, want: "-1xx"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("priority_%d", tt.priority), func(t *testing.T) {
			got := priorityClass(tt.priority)
			if got != tt.want {
				t.Errorf("priorityClass(%d) = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}
