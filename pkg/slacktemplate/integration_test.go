/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package slacktemplate

import (
	"testing"
)

// TestIntegrationBudgetAlert tests a realistic budget alert template
func TestIntegrationBudgetAlert(t *testing.T) {
	template := `🚨 Budget Alert: {{.budgetDisplayName}} exceeded {{printf "%.0f" (mul .alertThresholdExceeded 100)}}%
💰 Current spend: {{printf "%.2f" .costAmount}} {{.currencyCode}} / {{printf "%.2f" .budgetAmount}} {{.currencyCode}}
📊 Threshold: {{printf "%.1f" (mul .alertThresholdExceeded 100)}}%`

	data := map[string]interface{}{
		"budgetDisplayName":      "Q4 Marketing Budget",
		"alertThresholdExceeded": 0.85,
		"costAmount":             850.75,
		"budgetAmount":           1000.00,
		"currencyCode":           "USD",
	}

	executor, err := New(template)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	result, err := executor.Execute(data)
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	expected := `🚨 Budget Alert: Q4 Marketing Budget exceeded 85%
💰 Current spend: 850.75 USD / 1000.00 USD
📊 Threshold: 85.0%`

	if result != expected {
		t.Errorf("Execute() result mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

// TestIntegrationConditionalTemplate tests conditional formatting
func TestIntegrationConditionalTemplate(t *testing.T) {
	template := `{{if gt .alertThresholdExceeded 0.9}}🚨 CRITICAL{{else if gt .alertThresholdExceeded 0.75}}⚠️ WARNING{{else}}📊 NOTICE{{end}}: Budget {{.budgetDisplayName}} at {{printf "%.1f" (mul .alertThresholdExceeded 100)}}%`

	tests := []struct {
		name      string
		threshold float64
		want      string
	}{
		{
			name:      "critical threshold",
			threshold: 0.95,
			want:      "🚨 CRITICAL: Budget Test Budget at 95.0%",
		},
		{
			name:      "warning threshold",
			threshold: 0.80,
			want:      "⚠️ WARNING: Budget Test Budget at 80.0%",
		},
		{
			name:      "notice threshold",
			threshold: 0.50,
			want:      "📊 NOTICE: Budget Test Budget at 50.0%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := New(template)
			if err != nil {
				t.Fatalf("New() unexpected error: %v", err)
			}

			data := map[string]interface{}{
				"budgetDisplayName":      "Test Budget",
				"alertThresholdExceeded": tt.threshold,
			}

			result, err := executor.Execute(data)
			if err != nil {
				t.Fatalf("Execute() unexpected error: %v", err)
			}

			if result != tt.want {
				t.Errorf("Execute() = %q, want %q", result, tt.want)
			}
		})
	}
}
