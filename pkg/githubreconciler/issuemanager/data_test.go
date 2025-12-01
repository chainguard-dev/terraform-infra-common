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

	im, err := New[testData]("test-manager", titleTmpl, bodyTmpl, WithLabelTemplates[testData](labelTmpl1, labelTmpl2))
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

func Test_truncatePathForLabel(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantLen  int
		wantSame bool
	}{{
		name:     "path at 30 chars unchanged",
		path:     "123456789012345678901234567890",
		wantLen:  30,
		wantSame: true,
	}, {
		name:     "very long path truncated",
		path:     strings.Repeat("a", 100),
		wantLen:  24,
		wantSame: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncatePathForLabel(tt.path)
			if len(result) != tt.wantLen {
				t.Errorf("truncatePathForLabel() length = %d, want %d", len(result), tt.wantLen)
			}
			if tt.wantSame && result != tt.path {
				t.Errorf("truncatePathForLabel() should not have changed path, got = %q, want = %q", result, tt.path)
			}
			if !tt.wantSame && result == tt.path {
				t.Error("truncatePathForLabel() should have changed path but didn't")
			}
		})
	}
}

func Test_truncatePathForLabel_consistency(t *testing.T) {
	// Test that the same path always produces the same hash
	path := "this/is/a/very/long/path/that/exceeds/thirty/characters"
	result1 := truncatePathForLabel(path)
	result2 := truncatePathForLabel(path)

	if result1 != result2 {
		t.Errorf("truncatePathForLabel() not consistent: first = %q, second = %q", result1, result2)
	}

	// Test that different paths produce different hashes
	path2 := "this/is/a/different/very/long/path/that/exceeds/thirty/characters"
	result3 := truncatePathForLabel(path2)

	if result1 == result3 {
		t.Error("truncatePathForLabel() should produce different results for different paths")
	}
}
