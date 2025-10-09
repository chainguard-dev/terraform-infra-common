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

			gotDelay, ok, isError := GetRequeueDelay(err)
			if !ok {
				t.Fatal("GetRequeueDelay returned false")
			}
			if gotDelay != tt.wantDelay {
				t.Errorf("Got delay %v, want %v", gotDelay, tt.wantDelay)
			}
			if isError {
				t.Error("Expected isError = false for RequeueAfter, got true")
			}
		})
	}
}

func TestRetryAfter(t *testing.T) {
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
			err := RetryAfter(tt.delay)
			if err == nil {
				t.Fatal("Expected non-nil error")
			}

			gotDelay, ok, isError := GetRequeueDelay(err)
			if !ok {
				t.Fatal("GetRequeueDelay returned false")
			}
			if gotDelay != tt.wantDelay {
				t.Errorf("Got delay %v, want %v", gotDelay, tt.wantDelay)
			}
			if !isError {
				t.Error("Expected isError = true for RetryAfter, got false")
			}
		})
	}
}

func TestGetRequeueDelay(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantDelay   time.Duration
		wantOk      bool
		wantIsError bool
	}{
		{
			name:        "requeue error for polling",
			err:         RequeueAfter(10 * time.Second),
			wantDelay:   10 * time.Second,
			wantOk:      true,
			wantIsError: false,
		},
		{
			name:        "retry after error",
			err:         RetryAfter(30 * time.Second),
			wantDelay:   30 * time.Second,
			wantOk:      true,
			wantIsError: true,
		},
		{
			name:        "regular error",
			err:         errors.New("some error"),
			wantDelay:   0,
			wantOk:      false,
			wantIsError: false,
		},
		{
			name:        "nil error",
			err:         nil,
			wantDelay:   0,
			wantOk:      false,
			wantIsError: false,
		},
		{
			name:        "wrapped requeue error",
			err:         fmt.Errorf("operation failed: %w", RequeueAfter(15*time.Second)),
			wantDelay:   15 * time.Second,
			wantOk:      true,
			wantIsError: false,
		},
		{
			name:        "wrapped retry after error",
			err:         fmt.Errorf("rate limited: %w", RetryAfter(45*time.Second)),
			wantDelay:   45 * time.Second,
			wantOk:      true,
			wantIsError: true,
		},
		{
			name:        "double wrapped requeue error",
			err:         fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", RequeueAfter(20*time.Second))),
			wantDelay:   20 * time.Second,
			wantOk:      true,
			wantIsError: false,
		},
		{
			name:        "double wrapped retry after error",
			err:         fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", RetryAfter(60*time.Second))),
			wantDelay:   60 * time.Second,
			wantOk:      true,
			wantIsError: true,
		},
		{
			name:        "wrapped regular error",
			err:         fmt.Errorf("wrapped: %w", errors.New("some error")),
			wantDelay:   0,
			wantOk:      false,
			wantIsError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDelay, gotOk, gotIsError := GetRequeueDelay(tt.err)
			if gotOk != tt.wantOk {
				t.Errorf("ok: got = %v, wanted = %v", gotOk, tt.wantOk)
			}
			if gotDelay != tt.wantDelay {
				t.Errorf("delay: got = %v, wanted = %v", gotDelay, tt.wantDelay)
			}
			if gotIsError != tt.wantIsError {
				t.Errorf("isError: got = %v, wanted = %v", gotIsError, tt.wantIsError)
			}
		})
	}
}
