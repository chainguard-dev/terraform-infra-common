/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package issuemanager

import (
	"text/template"

	internaltemplate "github.com/chainguard-dev/terraform-infra-common/pkg/githubreconciler/internal/template"
)

// executeTemplate executes the given template with the provided data.
func (im *IM[T]) executeTemplate(tmpl *template.Template, data *T) (string, error) {
	return internaltemplate.ExecuteTemplate(tmpl, data)
}

// embedData embeds the given data as JSON in HTML comments within the body.
// The embedded data is placed at the end of the body using markers based on the identity.
func (im *IM[T]) embedData(body string, data *T) (string, error) {
	return internaltemplate.EmbedData(body, im.identity, "-issue-data", data)
}

// extractData extracts embedded data from the issue body.
// Returns an error if the data cannot be found or parsed.
func (im *IM[T]) extractData(body string) (*T, error) {
	return internaltemplate.ExtractData[T](body, im.identity, "-issue-data", "issue")
}
