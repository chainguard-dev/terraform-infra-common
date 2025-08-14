/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package slacktemplate

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"
)

// Executor handles template parsing and execution with basic helper functions
type Executor struct {
	tmpl *template.Template
}

// New creates a new template executor with helper functions
func New(templateText string) (*Executor, error) {
	funcMap := template.FuncMap{
		"mul":    multiply,
		"div":    divide,
		"add":    add,
		"sub":    subtract,
		"printf": fmt.Sprintf,
		"round":  round,
	}

	tmpl, err := template.New("slack").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &Executor{tmpl: tmpl}, nil
}

// Execute executes the template with the provided data
func (e *Executor) Execute(data interface{}) (string, error) {
	return e.ExecuteWithTimeout(context.Background(), data, 5*time.Second)
}

// ExecuteWithTimeout executes the template with a timeout to prevent hanging
func (e *Executor) ExecuteWithTimeout(ctx context.Context, data interface{}, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var buf bytes.Buffer
	done := make(chan error, 1)

	go func() {
		done <- e.tmpl.Execute(&buf, data)
	}()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("template execution failed: %w", err)
		}
		return buf.String(), nil
	case <-ctx.Done():
		return "", fmt.Errorf("template execution timed out after %v", timeout)
	}
}

// Helper functions for templates

// multiply multiplies two numbers
func multiply(a, b interface{}) (float64, error) {
	aVal, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	bVal, err := toFloat64(b)
	if err != nil {
		return 0, err
	}
	return aVal * bVal, nil
}

// divide divides two numbers
func divide(a, b interface{}) (float64, error) {
	aVal, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	bVal, err := toFloat64(b)
	if err != nil {
		return 0, err
	}
	if bVal == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return aVal / bVal, nil
}

// add adds two numbers
func add(a, b interface{}) (float64, error) {
	aVal, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	bVal, err := toFloat64(b)
	if err != nil {
		return 0, err
	}
	return aVal + bVal, nil
}

// subtract subtracts two numbers
func subtract(a, b interface{}) (float64, error) {
	aVal, err := toFloat64(a)
	if err != nil {
		return 0, err
	}
	bVal, err := toFloat64(b)
	if err != nil {
		return 0, err
	}
	return aVal - bVal, nil
}

// round rounds a number to the nearest integer
func round(f interface{}) (int64, error) {
	fVal, err := toFloat64(f)
	if err != nil {
		return 0, err
	}
	if fVal >= 0 {
		return int64(fVal + 0.5), nil
	}
	return int64(fVal - 0.5), nil
}

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
