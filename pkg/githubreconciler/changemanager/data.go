/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package changemanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"text/template"
)

// executeTemplate executes the given template with the provided data.
func (cm *CM[T]) executeTemplate(tmpl *template.Template, data *T) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}

// embedData embeds the given data as JSON in HTML comments within the body.
// The embedded data is placed at the end of the body using markers based on the identity.
func (cm *CM[T]) embedData(body string, data *T) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling data: %w", err)
	}

	marker := cm.identity + "-pr-data"
	embedded := fmt.Sprintf("\n\n<!--%s-->\n<!--\n%s\n-->\n<!--/%s-->", marker, string(jsonData), marker)

	return body + embedded, nil
}

// extractData extracts embedded data from the PR body.
// Returns an error if the data cannot be found or parsed.
func (cm *CM[T]) extractData(body string) (*T, error) {
	marker := cm.identity + "-pr-data"
	pattern := fmt.Sprintf(`(?s)<!--%s-->\s*<!--\s*(.+?)\s*-->\s*<!--/%s-->`, regexp.QuoteMeta(marker), regexp.QuoteMeta(marker))

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compiling regex: %w", err)
	}

	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return nil, errors.New("embedded data not found in PR body")
	}

	var data T
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return nil, fmt.Errorf("unmarshaling data: %w", err)
	}

	return &data, nil
}
