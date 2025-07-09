/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package workqueue

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestRequeueAfter(t *testing.T) {
	tests := []struct {
		name      string
		delay     time.Duration
		wantDelay time.Duration
	}{
		{
			name:      "5 second delay",
			delay:     5 * time.Second,
			wantDelay: 5 * time.Second,
		},
		{
			name:      "1 minute delay",
			delay:     time.Minute,
			wantDelay: time.Minute,
		},
		{
			name:      "zero delay",
			delay:     0,
			wantDelay: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequeueAfter(tt.delay)
			if err == nil {
				t.Fatal("Expected non-nil error")
			}

			gotDelay, ok := GetRequeueDelay(err)
			if !ok {
				t.Fatal("GetRequeueDelay returned false")
			}
			if gotDelay != tt.wantDelay {
				t.Errorf("Got delay %v, want %v", gotDelay, tt.wantDelay)
			}
		})
	}
}

func TestGetRequeueDelay(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantDelay time.Duration
		wantOk    bool
	}{
		{
			name:      "requeue error",
			err:       RequeueAfter(10 * time.Second),
			wantDelay: 10 * time.Second,
			wantOk:    true,
		},
		{
			name:      "regular error",
			err:       errors.New("some error"),
			wantDelay: 0,
			wantOk:    false,
		},
		{
			name:      "nil error",
			err:       nil,
			wantDelay: 0,
			wantOk:    false,
		},
		{
			name:      "wrapped requeue error",
			err:       fmt.Errorf("operation failed: %w", RequeueAfter(15*time.Second)),
			wantDelay: 15 * time.Second,
			wantOk:    true,
		},
		{
			name:      "double wrapped requeue error",
			err:       fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", RequeueAfter(20*time.Second))),
			wantDelay: 20 * time.Second,
			wantOk:    true,
		},
		{
			name:      "wrapped regular error",
			err:       fmt.Errorf("wrapped: %w", errors.New("some error")),
			wantDelay: 0,
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDelay, gotOk := GetRequeueDelay(tt.err)
			if gotOk != tt.wantOk {
				t.Errorf("Got ok=%v, want %v", gotOk, tt.wantOk)
			}
			if gotDelay != tt.wantDelay {
				t.Errorf("Got delay %v, want %v", gotDelay, tt.wantDelay)
			}
		})
	}
}
