/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package changemanager

import (
	"text/template"

	internaltemplate "github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler/internal/template"
)

// executeTemplate executes the given template with the provided data.
func (cm *CM[T]) executeTemplate(tmpl *template.Template, data *T) (string, error) {
	return internaltemplate.ExecuteTemplate(tmpl, data)
}

// embedData embeds the given data as JSON in HTML comments within the body.
// The embedded data is placed at the end of the body using markers based on the identity.
func (cm *CM[T]) embedData(body string, data *T) (string, error) {
	return internaltemplate.EmbedData(body, cm.identity, "-pr-data", data)
}

// extractData extracts embedded data from the PR body.
// Returns an error if the data cannot be found or parsed.
func (cm *CM[T]) extractData(body string) (*T, error) {
	return internaltemplate.ExtractData[T](body, cm.identity, "-pr-data", "PR")
}
