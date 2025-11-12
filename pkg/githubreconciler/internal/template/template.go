/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"text/template"
)

// Template provides template execution and data embedding/extraction capabilities.
// It is parameterized by type T which represents the data type to be embedded/extracted.
type Template[T any] struct {
	identity     string
	markerSuffix string
	entityType   string
	regex        *regexp.Regexp
}

// New creates a new Template instance with the specified identity, markerSuffix, and entityType.
// The identity distinguishes between different manager instances.
// The markerSuffix differentiates between different entity types (e.g., "-pr-data" or "-issue-data").
// The entityType is used in error messages (e.g., "PR" or "issue").
// Returns an error if identity validation fails or regex compilation fails.
func New[T any](identity string, markerSuffix string, entityType string) (*Template[T], error) {
	marker := identity + markerSuffix
	pattern := fmt.Sprintf(`(?s)<!--%s-->\s*<!--\s*(.+?)\s*-->\s*<!--/%s-->`, regexp.QuoteMeta(marker), regexp.QuoteMeta(marker))
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compiling regex: %w", err)
	}

	return &Template[T]{
		identity:     identity,
		markerSuffix: markerSuffix,
		entityType:   entityType,
		regex:        regex,
	}, nil
}

// Execute executes the given template with the provided data.
func (t *Template[T]) Execute(tmpl *template.Template, data *T) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}

// Embed embeds the given data as JSON in HTML comments within the body.
// The embedded data is placed at the end of the body using the configured marker.
func (t *Template[T]) Embed(body string, data *T) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling data: %w", err)
	}

	marker := t.identity + t.markerSuffix
	embedded := fmt.Sprintf("\n\n<!--%s-->\n<!--\n%s\n-->\n<!--/%s-->", marker, string(jsonData), marker)

	return body + embedded, nil
}

// Extract extracts embedded data from the body text.
// Returns an error if the data cannot be found or parsed.
func (t *Template[T]) Extract(body string) (*T, error) {
	matches := t.regex.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("embedded data not found in %s body", t.entityType)
	}

	var data T
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return nil, fmt.Errorf("unmarshaling data: %w", err)
	}

	return &data, nil
}
