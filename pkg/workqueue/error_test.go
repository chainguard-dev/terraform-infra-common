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
		wantDelay time.Duration // 0 means immediate retry, >0 means requeue with that delay
	}{{
		name:      "zero delay triggers immediate retry",
		delay:     0,
		wantDelay: 0,
	}, {
		name:      "500ms delay triggers immediate retry",
		delay:     500 * time.Millisecond,
		wantDelay: 0,
	}, {
		name:      "999ms delay triggers immediate retry",
		delay:     999 * time.Millisecond,
		wantDelay: 0,
	}, {
		name:      "1 second delay uses requeue",
		delay:     1 * time.Second,
		wantDelay: 1 * time.Second,
	}, {
		name:      "5 second delay uses requeue",
		delay:     5 * time.Second,
		wantDelay: 5 * time.Second,
	}, {
		name:      "1 minute delay uses requeue",
		delay:     time.Minute,
		wantDelay: time.Minute,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequeueAfter(tt.delay)
			if err == nil {
				t.Fatal("Expected non-nil error")
			}

			gotDelay, gotRequeue := GetRequeueDelay(err)
			wantRequeue := tt.wantDelay > 0

			if gotRequeue != wantRequeue {
				t.Errorf("requeue type: got = %v, wanted = %v", gotRequeue, wantRequeue)
			}
			if gotDelay != tt.wantDelay {
				t.Errorf("delay: got = %v, wanted = %v", gotDelay, tt.wantDelay)
			}

			// Check error message
			if wantRequeue {
				if got := err.Error(); got != "requeue requested" {
					t.Errorf("error message: got = %q, wanted = %q", got, "requeue requested")
				}
			} else {
				if got := err.Error(); got != "immediate retry requested" {
					t.Errorf("error message: got = %q, wanted = %q", got, "immediate retry requested")
				}
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
	}{{
		name:      "requeue error with 10s delay",
		err:       RequeueAfter(10 * time.Second),
		wantDelay: 10 * time.Second,
		wantOk:    true,
	}, {
		name:      "immediate retry from zero delay",
		err:       RequeueAfter(0),
		wantDelay: 0,
		wantOk:    false, // Should be a regular error, not a requeue error
	}, {
		name:      "immediate retry from 500ms delay",
		err:       RequeueAfter(500 * time.Millisecond),
		wantDelay: 0,
		wantOk:    false, // Should be a regular error, not a requeue error
	}, {
		name:      "regular error",
		err:       errors.New("some error"),
		wantDelay: 0,
		wantOk:    false,
	}, {
		name:      "nil error",
		err:       nil,
		wantDelay: 0,
		wantOk:    false,
	}, {
		name:      "wrapped requeue error",
		err:       fmt.Errorf("operation failed: %w", RequeueAfter(15*time.Second)),
		wantDelay: 15 * time.Second,
		wantOk:    true,
	}, {
		name:      "double wrapped requeue error",
		err:       fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", RequeueAfter(20*time.Second))),
		wantDelay: 20 * time.Second,
		wantOk:    true,
	}, {
		name:      "wrapped regular error",
		err:       fmt.Errorf("wrapped: %w", errors.New("some error")),
		wantDelay: 0,
		wantOk:    false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDelay, gotOk := GetRequeueDelay(tt.err)
			if gotOk != tt.wantOk {
				t.Errorf("ok: got = %v, wanted = %v", gotOk, tt.wantOk)
			}
			if gotDelay != tt.wantDelay {
				t.Errorf("delay: got = %v, wanted = %v", gotDelay, tt.wantDelay)
			}
		})
	}
}
