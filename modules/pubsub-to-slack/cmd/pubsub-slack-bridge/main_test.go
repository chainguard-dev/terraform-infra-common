/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"testing"
)

func TestProcessMessage(t *testing.T) {
	data := map[string]interface{}{
		"budgetDisplayName":      "Test Budget",
		"alertThresholdExceeded": 0.75,
		"costAmount":             123.45,
		"currencyCode":           "USD",
	}

	template := "Alert: {{.budgetDisplayName}} exceeded threshold ({{.costAmount}} {{.currencyCode}})"

	result, err := processMessage(data, template)
	if err != nil {
		t.Fatalf("processMessage failed: %v", err)
	}

	expected := "Alert: Test Budget exceeded threshold (123.45 USD)"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
