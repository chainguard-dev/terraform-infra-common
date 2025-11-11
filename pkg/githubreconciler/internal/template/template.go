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

// ExecuteTemplate executes the given template with the provided data.
// This is a standalone function that doesn't require any receiver.
func ExecuteTemplate[T any](tmpl *template.Template, data *T) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}

// EmbedData embeds the given data as JSON in HTML comments within the body.
// The embedded data is placed at the end of the body using the provided marker.
// The markerSuffix parameter allows customization (e.g., "-pr-data" or "-issue-data").
func EmbedData[T any](body string, identity string, markerSuffix string, data *T) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling data: %w", err)
	}

	marker := identity + markerSuffix
	embedded := fmt.Sprintf("\n\n<!--%s-->\n<!--\n%s\n-->\n<!--/%s-->", marker, string(jsonData), marker)

	return body + embedded, nil
}

// ExtractData extracts embedded data from the body text.
// The markerSuffix parameter should match what was used in EmbedData.
// The entityType parameter is used for error messages (e.g., "PR" or "issue").
// Returns an error if the data cannot be found or parsed.
func ExtractData[T any](body string, identity string, markerSuffix string, entityType string) (*T, error) {
	marker := identity + markerSuffix
	pattern := fmt.Sprintf(`(?s)<!--%s-->\s*<!--\s*(.+?)\s*-->\s*<!--/%s-->`, regexp.QuoteMeta(marker), regexp.QuoteMeta(marker))

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compiling regex: %w", err)
	}

	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("embedded data not found in %s body", entityType)
	}

	var data T
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return nil, fmt.Errorf("unmarshaling data: %w", err)
	}

	return &data, nil
}
