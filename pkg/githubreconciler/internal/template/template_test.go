/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package template

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

func Test_ExecuteTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmplStr  string
		data     *testData
		expected string
		wantErr  bool
	}{{
		name:     "simple template",
		tmplStr:  "{{.Foo}}/{{.Bar}}",
		data:     &testData{Foo: "foo", Bar: "bar"},
		expected: "foo/bar",
		wantErr:  false,
	}, {
		name:     "template with other words",
		tmplStr:  "{{.Foo}} in {{.Bar}}",
		data:     &testData{Foo: "bar", Bar: "baz"},
		expected: "bar in baz",
		wantErr:  false,
	}, {
		name:     "template with all fields",
		tmplStr:  "Update {{.Foo}} from {{.Bar}} ({{.Baz}})",
		data:     &testData{Foo: "baz", Bar: "foo", Baz: "bar"},
		expected: "Update baz from foo (bar)",
		wantErr:  false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := template.Must(template.New("test").Parse(tt.tmplStr))
			executor, err := New[testData]("test-identity", "-test-data", "test-entity")
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			result, err := executor.Execute(tmpl, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error: got = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("Execute() result: got = %q, wanted = %q", result, tt.expected)
			}
		})
	}
}

func Test_EmbedData(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		identity     string
		markerSuffix string
		data         *testData
		wantMarker   string
		entityType   string
	}{{
		name:         "embed data with first marker",
		body:         "This is the body",
		identity:     "first-identity",
		markerSuffix: "-data",
		data: &testData{
			Foo: "foo",
			Bar: "bar",
			Baz: "baz",
		},
		wantMarker: "first-identity-data",
		entityType: "entity",
	}, {
		name:         "embed data with second marker",
		body:         "This is another body",
		identity:     "second-identity",
		markerSuffix: "-other-data",
		data: &testData{
			Foo: "bar",
			Bar: "baz",
			Baz: "foo",
		},
		wantMarker: "second-identity-other-data",
		entityType: "entity",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := New[testData](tt.identity, tt.markerSuffix, tt.entityType)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			embedded, err := executor.Embed(tt.body, tt.data)
			if err != nil {
				t.Fatalf("Embed() failed: %v", err)
			}

			// Verify the original body is present
			if !strings.Contains(embedded, tt.body) {
				t.Errorf("embedded body missing original content")
			}

			// Verify the markers are present
			startMarker := "<!--" + tt.wantMarker + "-->"
			endMarker := "<!--/" + tt.wantMarker + "-->"
			if !strings.Contains(embedded, startMarker) {
				t.Errorf("embedded body missing start marker: %s", startMarker)
			}
			if !strings.Contains(embedded, endMarker) {
				t.Errorf("embedded body missing end marker: %s", endMarker)
			}

			// Verify we can extract the data back
			extracted, extractErr := executor.Extract(embedded)
			if extractErr != nil {
				t.Fatalf("Extract() failed: %v", extractErr)
			}

			if extracted.Foo != tt.data.Foo {
				t.Errorf("Foo: got = %q, wanted = %q", extracted.Foo, tt.data.Foo)
			}
			if extracted.Bar != tt.data.Bar {
				t.Errorf("Bar: got = %q, wanted = %q", extracted.Bar, tt.data.Bar)
			}
			if extracted.Baz != tt.data.Baz {
				t.Errorf("Baz: got = %q, wanted = %q", extracted.Baz, tt.data.Baz)
			}
		})
	}
}

func Test_ExtractData(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		identity        string
		markerSuffix    string
		entityType      string
		wantData        *testData
		wantErr         bool
		wantErrContains string
	}{{
		name: "extract data successfully",
		body: `This is the body

<!--test-bot-data-->
<!--
{
  "Foo": "foo",
  "Bar": "bar",
  "Baz": "baz"
}
-->
<!--/test-bot-data-->`,
		identity:     "test-bot",
		markerSuffix: "-data",
		entityType:   "entity",
		wantData: &testData{
			Foo: "foo",
			Bar: "bar",
			Baz: "baz",
		},
		wantErr: false,
	}, {
		name: "extract data with different values",
		body: `This is another body

<!--security-bot-other-data-->
<!--
{
  "Foo": "baz",
  "Bar": "foo",
  "Baz": "bar"
}
-->
<!--/security-bot-other-data-->`,
		identity:     "security-bot",
		markerSuffix: "-other-data",
		entityType:   "entity",
		wantData: &testData{
			Foo: "baz",
			Bar: "foo",
			Baz: "bar",
		},
		wantErr: false,
	}, {
		name: "extract with extra whitespace",
		body: `Body content

<!--test-bot-data-->
<!--
{
  "Foo": "bar",
  "Bar": "baz",
  "Baz": "foo"
}
-->
<!--/test-bot-data-->`,
		identity:     "test-bot",
		markerSuffix: "-data",
		entityType:   "entity",
		wantData: &testData{
			Foo: "bar",
			Bar: "baz",
			Baz: "foo",
		},
		wantErr: false,
	}, {
		name:            "body without embedded data",
		body:            "This is a body without embedded data",
		identity:        "test-bot",
		markerSuffix:    "-data",
		entityType:      "entity",
		wantErr:         true,
		wantErrContains: "entity",
	}, {
		name:            "body without embedded data different marker",
		body:            "This is another body without embedded data",
		identity:        "security-bot",
		markerSuffix:    "-other-data",
		entityType:      "entity",
		wantErr:         true,
		wantErrContains: "entity",
	}, {
		name:            "body with wrong marker",
		body:            "<!--wrong-marker-->\n<!--\n{}\n-->\n<!--/wrong-marker-->",
		identity:        "test-bot",
		markerSuffix:    "-data",
		entityType:      "entity",
		wantErr:         true,
		wantErrContains: "entity",
	}, {
		name:            "invalid JSON",
		body:            "Original body\n\n<!--test-bot-data-->\n<!--\nthis is not valid JSON\n-->\n<!--/test-bot-data-->",
		identity:        "test-bot",
		markerSuffix:    "-data",
		entityType:      "entity",
		wantErr:         true,
		wantErrContains: "unmarshaling data",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := New[testData](tt.identity, tt.markerSuffix, tt.entityType)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}
			extracted, err := executor.Extract(tt.body)

			if tt.wantErr {
				// Test error cases
				if err == nil {
					t.Error("Extract() should have failed")
				} else if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error message should contain %q: %v", tt.wantErrContains, err)
				}
			} else {
				// Test success cases
				if err != nil {
					t.Fatalf("Extract() failed: %v", err)
				}

				if extracted.Foo != tt.wantData.Foo {
					t.Errorf("Foo: got = %q, wanted = %q", extracted.Foo, tt.wantData.Foo)
				}
				if extracted.Bar != tt.wantData.Bar {
					t.Errorf("Bar: got = %q, wanted = %q", extracted.Bar, tt.wantData.Bar)
				}
				if extracted.Baz != tt.wantData.Baz {
					t.Errorf("Baz: got = %q, wanted = %q", extracted.Baz, tt.wantData.Baz)
				}
			}
		})
	}
}
