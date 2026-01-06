/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package issuemanager

import (
	"strings"
	"testing"
	"text/template"
)

type testData struct {
	Foo string
	Bar string
	Baz string
}

// Equal implements the Comparable interface for testData.
// It compares Foo and Bar to determine if two instances represent the same issue.
func (d testData) Equal(other testData) bool {
	return d.Foo == other.Foo && d.Bar == other.Bar
}

func Test_executeLabelTemplates(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.Foo}}"))
	bodyTmpl := template.Must(template.New("body").Parse("Update"))
	labelTmpl1 := template.Must(template.New("label1").Parse("label1:{{.Baz}}"))
	labelTmpl2 := template.Must(template.New("label2").Parse("label2:{{.Bar}}"))

	im, err := New("test-manager", titleTmpl, bodyTmpl, WithLabelTemplates[testData](labelTmpl1, labelTmpl2))
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		Foo: "foo",
		Bar: "bar",
		Baz: "baz",
	}

	// Test executing label templates
	label1, err := im.templateExecutor.Execute(labelTmpl1, data)
	if err != nil {
		t.Fatalf("Execute for label1 failed: %v", err)
	}
	if label1 != "label1:baz" {
		t.Errorf("label1: got = %q, wanted = %q", label1, "label1:baz")
	}

	label2, err := im.templateExecutor.Execute(labelTmpl2, data)
	if err != nil {
		t.Fatalf("Execute for label2 failed: %v", err)
	}
	if label2 != "label2:bar" {
		t.Errorf("label2: got = %q, wanted = %q", label2, "label2:bar")
	}
}

func Test_embedData_withComparableInterface(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.Foo}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Bar}}"))

	im, err := New[testData]("test-manager", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	data := &testData{
		Foo: "foo",
		Bar: "bar",
		Baz: "baz",
	}

	body := "This is the issue body"
	embedded, err := im.templateExecutor.Embed(body, data)
	if err != nil {
		t.Fatalf("Embed() failed: %v", err)
	}

	// Verify the original body is present
	if !strings.Contains(embedded, body) {
		t.Errorf("embedded body missing original content")
	}

	// Verify the markers are present
	if !strings.Contains(embedded, "<!--test-manager-issue-data-->") {
		t.Errorf("embedded body missing start marker")
	}
	if !strings.Contains(embedded, "<!--/test-manager-issue-data-->") {
		t.Errorf("embedded body missing end marker")
	}

	// Verify JSON data is embedded
	if !strings.Contains(embedded, `"Foo": "foo"`) {
		t.Errorf("embedded body missing Foo field")
	}
	if !strings.Contains(embedded, `"Bar": "bar"`) {
		t.Errorf("embedded body missing Bar field")
	}
}

func Test_extractData_withComparableInterface(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.Foo}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Bar}}"))

	im, err := New[testData]("test-manager", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	body := `This is the issue body

<!--test-manager-issue-data-->
<!--
{
  "Foo": "bar",
  "Bar": "baz",
  "Baz": "foo"
}
-->
<!--/test-manager-issue-data-->`

	extracted, err := im.templateExecutor.Extract(body)
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	if extracted.Foo != "bar" {
		t.Errorf("Foo: got = %q, wanted = %q", extracted.Foo, "bar")
	}
	if extracted.Bar != "baz" {
		t.Errorf("Bar: got = %q, wanted = %q", extracted.Bar, "baz")
	}
	if extracted.Baz != "foo" {
		t.Errorf("Baz: got = %q, wanted = %q", extracted.Baz, "foo")
	}
}

func Test_embedThenExtract_roundTrip(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.Foo}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Bar}}"))

	im, err := New[testData]("test-manager", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	original := &testData{
		Foo: "baz",
		Bar: "foo",
		Baz: "bar",
	}

	// Embed the data
	body := "Original body content"
	embedded, err := im.templateExecutor.Embed(body, original)
	if err != nil {
		t.Fatalf("Embed() failed: %v", err)
	}

	// Extract the data
	extracted, err := im.templateExecutor.Extract(embedded)
	if err != nil {
		t.Fatalf("Extract() failed: %v", err)
	}

	// Verify all fields match
	if extracted.Foo != original.Foo {
		t.Errorf("Foo: got = %q, wanted = %q", extracted.Foo, original.Foo)
	}
	if extracted.Bar != original.Bar {
		t.Errorf("Bar: got = %q, wanted = %q", extracted.Bar, original.Bar)
	}
	if extracted.Baz != original.Baz {
		t.Errorf("Baz: got = %q, wanted = %q", extracted.Baz, original.Baz)
	}
}

func Test_Equal_method_matching(t *testing.T) {
	tests := []struct {
		name     string
		d1       testData
		d2       testData
		expected bool
	}{{
		name:     "same Foo and Bar matches",
		d1:       testData{Foo: "foo", Bar: "bar", Baz: "baz"},
		d2:       testData{Foo: "foo", Bar: "bar", Baz: "different"},
		expected: true,
	}, {
		name:     "different Foo does not match",
		d1:       testData{Foo: "foo", Bar: "bar", Baz: "baz"},
		d2:       testData{Foo: "bar", Bar: "bar", Baz: "baz"},
		expected: false,
	}, {
		name:     "different Bar does not match",
		d1:       testData{Foo: "foo", Bar: "bar", Baz: "baz"},
		d2:       testData{Foo: "foo", Bar: "baz", Baz: "baz"},
		expected: false,
	}, {
		name:     "both fields match even with different Baz",
		d1:       testData{Foo: "bar", Bar: "baz", Baz: "foo"},
		d2:       testData{Foo: "bar", Bar: "baz", Baz: "different"},
		expected: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.d1.Equal(tt.d2)
			if result != tt.expected {
				t.Errorf("Equal(): got = %v, wanted = %v", result, tt.expected)
			}
		})
	}
}

func Test_extractData_notFound(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.Foo}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Bar}}"))

	im, err := New[testData]("test-manager", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	body := "This is an issue body without embedded data"
	_, err = im.templateExecutor.Extract(body)
	if err == nil {
		t.Error("Extract() should have failed for body without embedded data")
	}
}

func Test_extractData_invalidJSON(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.Foo}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Bar}}"))

	im, err := New[testData]("test-manager", titleTmpl, bodyTmpl)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	body := `This is the issue body

<!--test-manager-issue-data-->
<!--
this is not valid JSON
-->
<!--/test-manager-issue-data-->`

	_, err = im.templateExecutor.Extract(body)
	if err == nil {
		t.Error("Extract() should have failed for invalid JSON")
	}

	if !strings.Contains(err.Error(), "unmarshaling data") {
		t.Errorf("error should mention unmarshaling: %v", err)
	}
}

func Test_identityLengthValidation(t *testing.T) {
	titleTmpl := template.Must(template.New("title").Parse("{{.Foo}}"))
	bodyTmpl := template.Must(template.New("body").Parse("{{.Bar}}"))

	tests := []struct {
		name      string
		identity  string
		shouldErr bool
	}{{
		name:      "identity within limit (20 chars)",
		identity:  "12345678901234567890",
		shouldErr: false,
	}, {
		name:      "identity exceeds limit (21 chars)",
		identity:  "123456789012345678901",
		shouldErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New[testData](tt.identity, titleTmpl, bodyTmpl)
			if tt.shouldErr && err == nil {
				t.Error("New() should have failed for identity exceeding 20 characters")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("New() should not have failed: %v", err)
			}
			if tt.shouldErr && err != nil && !strings.Contains(err.Error(), "20 characters or less") {
				t.Errorf("error should mention character limit: %v", err)
			}
		})
	}
}

func Test_constructPathLabel(t *testing.T) {
	tests := []struct {
		name         string
		identity     string
		path         string
		wantLen      int
		wantContains string
	}{{
		name:         "short path unchanged",
		identity:     "test",
		path:         "short/path",
		wantLen:      15, // "test:short/path"
		wantContains: "test:short/path",
	}, {
		name:         "path at exactly maxGitHubLabelLength unchanged",
		identity:     "test",
		path:         "123456789012345678901234567890123456789012345",
		wantLen:      maxGitHubLabelLength,
		wantContains: "test:123456789012345678901234567890123456789012345",
	}, {
		name:         "long path truncated with hash",
		identity:     "test",
		path:         strings.Repeat("a", 100),
		wantLen:      maxGitHubLabelLength,
		wantContains: "test:",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := constructPathLabel(tt.identity, tt.path)
			if len(result) != tt.wantLen {
				t.Errorf("constructPathLabel() length = %d, want %d", len(result), tt.wantLen)
			}
			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("constructPathLabel() = %q, want to contain %q", result, tt.wantContains)
			}
			// Verify format is always identity:something
			parts := strings.SplitN(result, ":", 2)
			if len(parts) != 2 || parts[0] != tt.identity {
				t.Errorf("constructPathLabel() = %q, want format %q:*", result, tt.identity)
			}
		})
	}
}

func Test_constructPathLabel_consistency(t *testing.T) {
	identity := "test"
	// Test that the same path always produces the same label
	path := "this/is/a/very/long/path/that/exceeds/fifty/characters/when/combined/with/identity"
	result1 := constructPathLabel(identity, path)
	result2 := constructPathLabel(identity, path)

	if result1 != result2 {
		t.Errorf("constructPathLabel() not consistent: first = %q, second = %q", result1, result2)
	}

	// Test that different paths produce different labels
	path2 := "this/is/a/different/very/long/path/that/exceeds/fifty/characters/when/combined/with/identity"
	result3 := constructPathLabel(identity, path2)

	if result1 == result3 {
		t.Error("constructPathLabel() should produce different results for different paths")
	}

	// Verify both results are exactly maxGitHubLabelLength characters
	if len(result1) != maxGitHubLabelLength {
		t.Errorf("constructPathLabel() length = %d, want %d", len(result1), maxGitHubLabelLength)
	}
	if len(result3) != maxGitHubLabelLength {
		t.Errorf("constructPathLabel() length = %d, want %d", len(result3), maxGitHubLabelLength)
	}
}
