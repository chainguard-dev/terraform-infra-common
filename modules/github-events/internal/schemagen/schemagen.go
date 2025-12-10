/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package schemagen

import (
	"os"
	"strings"

	"cloud.google.com/go/bigquery"
)

// Generate writes a bigquery schema to the given path.
func Generate(path string, v any) error {
	s, err := bigquery.InferSchema(v)
	if err != nil {
		return err
	}
	s = relax(s)
	b, err := s.ToJSONFields()
	if err != nil {
		return err
	}
	b = append(b, '\n') // or else EOF newline linter yells at us.

	return os.WriteFile(path, b, 0644) //nolint:gosec
}

func relax(s bigquery.Schema) bigquery.Schema {
	for _, fs := range s {
		fs.Required = false
		fs.Name = strings.ToLower(string(fs.Name[0])) + fs.Name[1:]
		if fs.Type == bigquery.RecordFieldType {
			fs.Schema = relax(fs.Schema)
		}
	}
	return s
}
