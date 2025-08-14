/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package slacktemplate

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid simple template",
			template: "Hello {{.name}}",
			wantErr:  false,
		},
		{
			name:     "valid template with helper functions",
			template: "Budget {{.budgetName}} exceeded {{printf \"%.0f\" (mul .threshold 100)}}%",
			wantErr:  false,
		},
		{
			name:        "invalid template syntax",
			template:    "Hello {{.name",
			wantErr:     true,
			errContains: "failed to parse template",
		},
		{
			name:        "template with unknown function",
			template:    "Hello {{unknown .name}}",
			wantErr:     true,
			errContains: "failed to parse template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := New(tt.template)
			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("New() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}
			if executor == nil {
				t.Errorf("New() returned nil executor")
			}
		})
	}
}

func TestExecutor_Execute(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "simple field substitution",
			template: "Hello {{.name}}",
			data:     map[string]interface{}{"name": "World"},
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "budget alert template",
			template: "Budget {{.budgetDisplayName}} exceeded {{printf \"%.0f\" (mul .alertThresholdExceeded 100)}}%",
			data: map[string]interface{}{
				"budgetDisplayName":      "Q4 Marketing",
				"alertThresholdExceeded": 0.75,
			},
			want:    "Budget Q4 Marketing exceeded 75%",
			wantErr: false,
		},
		{
			name:     "math operations",
			template: "Add: {{add 10 5}}, Sub: {{sub 10 5}}, Mul: {{mul 10 5}}, Div: {{div 10 5}}",
			data:     map[string]interface{}{},
			want:     "Add: 15, Sub: 5, Mul: 50, Div: 2",
			wantErr:  false,
		},
		{
			name:     "round function",
			template: "Rounded: {{round 3.7}}",
			data:     map[string]interface{}{},
			want:     "Rounded: 4",
			wantErr:  false,
		},
		{
			name:     "printf formatting",
			template: "Cost: {{printf \"$%.2f\" .cost}}",
			data:     map[string]interface{}{"cost": 123.456},
			want:     "Cost: $123.46",
			wantErr:  false,
		},
		{
			name:     "missing field",
			template: "Hello {{.missing}}",
			data:     map[string]interface{}{"name": "World"},
			want:     "Hello <no value>",
			wantErr:  false,
		},
		{
			name:     "conditional template",
			template: "{{if .urgent}}🚨 URGENT: {{end}}{{.message}}",
			data: map[string]interface{}{
				"urgent":  true,
				"message": "System down",
			},
			want:    "🚨 URGENT: System down",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := New(tt.template)
			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}

			got, err := executor.Execute(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExecutor_ExecuteWithTimeout(t *testing.T) {
	// Test timeout behavior
	executor, err := New("{{range .items}}{{.}}{{end}}")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	ctx := context.Background()

	// This should complete quickly
	result, err := executor.ExecuteWithTimeout(ctx, map[string]interface{}{
		"items": []string{"a", "b", "c"},
	}, 1*time.Second)

	if err != nil {
		t.Errorf("ExecuteWithTimeout() unexpected error: %v", err)
	}
	if result != "abc" {
		t.Errorf("ExecuteWithTimeout() = %q, want %q", result, "abc")
	}
}

func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "multiply with floats",
			template: "{{mul 3.5 2}}",
			data:     nil,
			want:     "7",
			wantErr:  false,
		},
		{
			name:     "divide by zero",
			template: "{{div 10 0}}",
			data:     nil,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "round positive",
			template: "{{round 3.7}}",
			data:     nil,
			want:     "4",
			wantErr:  false,
		},
		{
			name:     "round negative",
			template: "{{round -3.7}}",
			data:     nil,
			want:     "-4",
			wantErr:  false,
		},
		{
			name:     "printf with multiple args",
			template: "{{printf \"%s: $%.2f\" \"Total\" 123.456}}",
			data:     nil,
			want:     "Total: $123.46",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := New(tt.template)
			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}

			got, err := executor.Execute(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Execute() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Execute() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    float64
		wantErr bool
	}{
		{"float64", 3.14, 3.14, false},
		{"float32", float32(3.14), 3.140000104904175, false}, // float32 precision
		{"int", 42, 42.0, false},
		{"int32", int32(42), 42.0, false},
		{"int64", int64(42), 42.0, false},
		{"uint", uint(42), 42.0, false},
		{"uint32", uint32(42), 42.0, false},
		{"uint64", uint64(42), 42.0, false},
		{"string", "42", 0, true},
		{"nil", nil, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toFloat64(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("toFloat64() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("toFloat64() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("toFloat64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Benchmark template execution
func BenchmarkExecutor_Execute(b *testing.B) {
	executor, err := New("Budget {{.budgetDisplayName}} exceeded {{printf \"%.0f\" (mul .alertThresholdExceeded 100)}}%")
	if err != nil {
		b.Fatalf("New() unexpected error: %v", err)
	}

	data := map[string]interface{}{
		"budgetDisplayName":      "Q4 Marketing",
		"alertThresholdExceeded": 0.75,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.Execute(data)
		if err != nil {
			b.Fatalf("Execute() unexpected error: %v", err)
		}
	}
}
