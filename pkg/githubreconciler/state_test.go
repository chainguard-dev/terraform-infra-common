/*
Copyright 2025 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package githubreconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-github/v72/github"
)

// mockTransport allows us to intercept HTTP calls
type mockTransport struct {
	responses     map[string]mockResponse
	listCallCount int
	roundTripFunc func(*http.Request) (*http.Response, error)
	returnError   error // error to return from RoundTrip
}

type mockResponse struct {
	statusCode int
	body       string
	nextPage   int
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.roundTripFunc != nil {
		return m.roundTripFunc(req)
	}

	path := req.URL.Path
	// Include query string in the lookup key
	if req.URL.RawQuery != "" {
		path = path + "?" + req.URL.RawQuery
	}

	// Track list calls
	if strings.Contains(path, "/comments") && req.Method == "GET" {
		m.listCallCount++
	}

	resp, ok := m.responses[path]
	if !ok {
		// Default 404 response
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	}

	header := make(http.Header)
	if resp.nextPage > 0 {
		// Use the base path without query for the Link header
		basePath := req.URL.Path
		header.Set("Link", fmt.Sprintf(`<https://api.github.com%s?page=%d>; rel="next"`, basePath, resp.nextPage))
	}

	return &http.Response{
		StatusCode: resp.statusCode,
		Body:       io.NopCloser(strings.NewReader(resp.body)),
		Header:     header,
	}, nil
}

// Helper function to create test resources
func testResource(number int) *Resource {
	return &Resource{
		Owner:  "testowner",
		Repo:   "testrepo",
		Type:   ResourceTypeIssue,
		Number: number,
		URL:    fmt.Sprintf("https://github.com/testowner/testrepo/issues/%d", number),
	}
}

// Helper function to create a comment with state
func commentWithState(id int64, identity string, state interface{}, message string) *github.IssueComment {
	stateJSON, _ := json.MarshalIndent(state, "", "  ")

	// Build comment body matching the actual format
	var body strings.Builder
	body.WriteString(fmt.Sprintf("<!--%s-->\n\n", identity))
	body.WriteString(message)
	body.WriteString("\n\n")
	body.WriteString(fmt.Sprintf("<!--%s-state-->\n<!--\n", identity))
	body.WriteString(string(stateJSON))
	body.WriteString(fmt.Sprintf("\n-->\n<!--/%s-state-->", identity))

	bodyStr := body.String()
	login := "test-user"
	return &github.IssueComment{
		ID:   &id,
		Body: &bodyStr,
		User: &github.User{Login: &login},
	}
}

func TestState_Fetch(t *testing.T) {
	type testState struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name       string
		identity   string
		comments   []*github.IssueComment
		want       *testState
		wantErr    bool
		statusCode int
	}{
		{
			name:     "fetch existing state",
			identity: "test-app",
			comments: []*github.IssueComment{
				commentWithState(1, "test-app", testState{Value: "hello", Count: 42}, "Test message"),
			},
			want:       &testState{Value: "hello", Count: 42},
			statusCode: 200,
		},
		{
			name:     "no state comment found",
			identity: "test-app",
			comments: []*github.IssueComment{
				commentWithState(1, "other-app", testState{Value: "ignored", Count: 99}, "Other message"),
			},
			want:       nil,
			statusCode: 200,
		},
		{
			name:     "multiple comments - use first match",
			identity: "test-app",
			comments: []*github.IssueComment{
				commentWithState(1, "other-app", testState{Value: "ignored", Count: 99}, "Other message"),
				commentWithState(2, "test-app", testState{Value: "first", Count: 1}, "First message"),
				commentWithState(3, "test-app", testState{Value: "second", Count: 2}, "Second message"),
			},
			want:       &testState{Value: "first", Count: 1},
			statusCode: 200,
		},
		{
			name:       "empty comments list",
			identity:   "test-app",
			comments:   []*github.IssueComment{},
			want:       nil,
			statusCode: 200,
		},
		{
			name:       "API error",
			identity:   "test-app",
			statusCode: 500,
			wantErr:    true,
		},
		{
			name:     "comment without state",
			identity: "test-app",
			comments: []*github.IssueComment{{
				ID:   github.Ptr(int64(1)),
				Body: github.Ptr("<!--test-app-->\n\nJust a message without state"),
			}},
			want:       nil,
			statusCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Create mock transport
			transport := &mockTransport{
				responses: make(map[string]mockResponse),
			}

			// Set up response
			if tt.statusCode != 0 {
				commentsJSON, _ := json.Marshal(tt.comments)
				// GitHub client adds per_page parameter
				transport.responses["/repos/testowner/testrepo/issues/123/comments?per_page=100"] = mockResponse{
					statusCode: tt.statusCode,
					body:       string(commentsJSON),
				}
			}

			httpClient := &http.Client{Transport: transport}
			client := github.NewClient(httpClient)

			state := NewState[testState](tt.identity, client, testResource(123))

			got, err := state.Fetch(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("State.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("State.Fetch() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestState_Commit(t *testing.T) {
	type testState struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	tests := []struct {
		name             string
		identity         string
		state            testState
		message          string
		existingComments []*github.IssueComment
		wantCreate       bool
		wantUpdate       bool
		wantErr          bool
		createError      int // HTTP status code for create
		updateError      int // HTTP status code for update
	}{
		{
			name:             "create new state comment",
			identity:         "test-app",
			state:            testState{Value: "new", Count: 1},
			message:          "Initial state",
			existingComments: []*github.IssueComment{},
			wantCreate:       true,
		},
		{
			name:     "update existing state comment",
			identity: "test-app",
			state:    testState{Value: "updated", Count: 2},
			message:  "Updated state",
			existingComments: []*github.IssueComment{
				commentWithState(123, "test-app", testState{Value: "old", Count: 1}, "Old message"),
			},
			wantUpdate: true,
		},
		{
			name:     "update with multiple comments",
			identity: "test-app",
			state:    testState{Value: "updated", Count: 3},
			message:  "Updated state",
			existingComments: []*github.IssueComment{
				commentWithState(100, "other-app", testState{Value: "ignored", Count: 99}, "Other message"),
				commentWithState(200, "test-app", testState{Value: "old", Count: 1}, "Old message"),
				commentWithState(300, "test-app", testState{Value: "old2", Count: 2}, "Old message 2"),
			},
			wantUpdate: true,
		},
		{
			name:             "create comment error",
			identity:         "test-app",
			state:            testState{Value: "new", Count: 1},
			message:          "Initial state",
			existingComments: []*github.IssueComment{},
			createError:      500,
			wantCreate:       true,
			wantErr:          true,
		},
		{
			name:     "update comment error",
			identity: "test-app",
			state:    testState{Value: "updated", Count: 2},
			message:  "Updated state",
			existingComments: []*github.IssueComment{
				commentWithState(123, "test-app", testState{Value: "old", Count: 1}, "Old message"),
			},
			updateError: 500,
			wantUpdate:  true,
			wantErr:     true,
		},
		{
			name:     "skip update if content unchanged",
			identity: "test-app",
			state:    testState{Value: "same", Count: 1},
			message:  "Same message",
			existingComments: []*github.IssueComment{
				commentWithState(123, "test-app", testState{Value: "same", Count: 1}, "Same message"),
			},
			wantUpdate: false,
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			createCalled := false
			updateCalled := false

			// Create mock transport
			transport := &mockTransport{
				responses: make(map[string]mockResponse),
			}

			// Default roundtrip behavior that checks responses map
			defaultRoundTrip := func(req *http.Request) (*http.Response, error) {
				if transport.returnError != nil {
					return nil, transport.returnError
				}

				path := req.URL.Path
				// Include query string in the lookup key
				if req.URL.RawQuery != "" {
					path = path + "?" + req.URL.RawQuery
				}

				// Track list calls
				if strings.Contains(path, "/comments") && req.Method == "GET" {
					transport.listCallCount++
				}

				resp, ok := transport.responses[path]
				if !ok {
					// Default 404 response
					return &http.Response{
						StatusCode: 404,
						Body:       io.NopCloser(strings.NewReader("")),
						Header:     make(http.Header),
					}, transport.returnError
				}

				header := make(http.Header)
				if resp.nextPage > 0 {
					// Use the base path without query for the Link header
					basePath := req.URL.Path
					header.Set("Link", fmt.Sprintf(`<https://api.github.com%s?page=%d>; rel="next"`, basePath, resp.nextPage))
				}

				return &http.Response{
					StatusCode: resp.statusCode,
					Body:       io.NopCloser(strings.NewReader(resp.body)),
					Header:     header,
				}, transport.returnError
			}

			// Track API calls
			transport.roundTripFunc = func(req *http.Request) (*http.Response, error) {
				if req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/comments") {
					createCalled = true
					if tt.createError != 0 {
						return &http.Response{
							StatusCode: tt.createError,
							Body:       io.NopCloser(strings.NewReader("{}")),
							Header:     make(http.Header),
						}, nil
					}
					// Check for POST-specific response
					if resp, ok := transport.responses[req.URL.Path+"-POST"]; ok {
						return &http.Response{
							StatusCode: resp.statusCode,
							Body:       io.NopCloser(strings.NewReader(resp.body)),
							Header:     make(http.Header),
						}, nil
					}
				}
				if req.Method == "PATCH" && strings.Contains(req.URL.Path, "/comments/") {
					updateCalled = true
					if tt.updateError != 0 {
						return &http.Response{
							StatusCode: tt.updateError,
							Body:       io.NopCloser(strings.NewReader("{}")),
							Header:     make(http.Header),
						}, nil
					}
				}
				// Default behavior
				return defaultRoundTrip(req)
			}

			// Set up list response
			commentsJSON, _ := json.Marshal(tt.existingComments)
			// GitHub client adds per_page parameter
			transport.responses["/repos/testowner/testrepo/issues/123/comments?per_page=100"] = mockResponse{
				statusCode: 200,
				body:       string(commentsJSON),
			}

			// Set up create response for new comments
			if tt.wantCreate {
				createdComment := commentWithState(12345, tt.identity, tt.state, tt.message)
				createdJSON, _ := json.Marshal(createdComment)

				// This will be used for POST requests
				transport.responses["/repos/testowner/testrepo/issues/123/comments-POST"] = mockResponse{
					statusCode: 201,
					body:       string(createdJSON),
				}
			}

			// Set up update response for existing comments
			if tt.wantUpdate && len(tt.existingComments) > 0 {
				// Find the first comment with matching identity
				for _, comment := range tt.existingComments {
					if comment.Body != nil && strings.Contains(*comment.Body, fmt.Sprintf("<!--%s-->", tt.identity)) && comment.ID != nil {
						updatedComment := commentWithState(*comment.ID, tt.identity, tt.state, tt.message)
						updatedJSON, _ := json.Marshal(updatedComment)

						transport.responses[fmt.Sprintf("/repos/testowner/testrepo/issues/comments/%d", *comment.ID)] = mockResponse{
							statusCode: 200,
							body:       string(updatedJSON),
						}
						break
					}
				}
			}

			httpClient := &http.Client{Transport: transport}
			client := github.NewClient(httpClient)
			state := NewState[testState](tt.identity, client, testResource(123))

			err := state.Commit(ctx, &tt.state, tt.message)

			if (err != nil) != tt.wantErr {
				t.Errorf("State.Commit() error = %v, wantErr %v", err, tt.wantErr)
			}

			if createCalled != tt.wantCreate {
				t.Errorf("State.Commit() createCalled = %v, want %v", createCalled, tt.wantCreate)
			}

			if updateCalled != tt.wantUpdate {
				t.Errorf("State.Commit() updateCalled = %v, want %v", updateCalled, tt.wantUpdate)
			}
		})
	}
}

func TestState_Pagination(t *testing.T) {
	// Test that state handles pagination correctly
	ctx := context.Background()

	type testState struct {
		Value string `json:"value"`
	}

	// Create mock transport
	transport := &mockTransport{
		responses: make(map[string]mockResponse),
	}

	// First page - empty
	// GitHub client adds per_page parameter
	transport.responses["/repos/testowner/testrepo/issues/123/comments?per_page=100"] = mockResponse{
		statusCode: 200,
		body:       "[]",
		nextPage:   2,
	}

	// Second page - has our comment
	comments := []*github.IssueComment{
		commentWithState(1, "test-app", testState{Value: "found"}, "Target comment"),
	}
	commentsJSON, _ := json.Marshal(comments)
	transport.responses["/repos/testowner/testrepo/issues/123/comments?page=2&per_page=100"] = mockResponse{
		statusCode: 200,
		body:       string(commentsJSON),
	}

	httpClient := &http.Client{Transport: transport}
	client := github.NewClient(httpClient)
	state := NewState[testState]("test-app", client, testResource(123))

	got, err := state.Fetch(ctx)
	if err != nil {
		t.Fatalf("State.Fetch() error = %v", err)
	}

	if got == nil {
		t.Fatal("State.Fetch() returned nil, expected to find state")
	} else if got.Value != "found" {
		t.Errorf("State.Fetch() got value = %v, want %v", got.Value, "found")
	}

	// Should have made 2 calls due to pagination
	if transport.listCallCount != 2 {
		t.Errorf("Expected 2 ListComments calls for pagination, got %d", transport.listCallCount)
	}
}

func TestState_CommentFormat(t *testing.T) {
	// Test the exact format of state comments
	ctx := context.Background()
	var capturedBody string

	// Create mock transport
	transport := &mockTransport{
		responses: make(map[string]mockResponse),
	}

	// Default roundtrip behavior
	defaultRoundTrip := func(req *http.Request) (*http.Response, error) {
		if transport.returnError != nil {
			return nil, transport.returnError
		}

		path := req.URL.Path
		// Include query string in the lookup key
		if req.URL.RawQuery != "" {
			path = path + "?" + req.URL.RawQuery
		}

		// Check for method-specific response first
		if resp, ok := transport.responses[path+"-"+req.Method]; ok {
			return &http.Response{
				StatusCode: resp.statusCode,
				Body:       io.NopCloser(strings.NewReader(resp.body)),
				Header:     make(http.Header),
			}, transport.returnError
		}

		resp, ok := transport.responses[path]
		if !ok {
			// Default 404 response
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, transport.returnError
		}

		return &http.Response{
			StatusCode: resp.statusCode,
			Body:       io.NopCloser(strings.NewReader(resp.body)),
			Header:     make(http.Header),
		}, transport.returnError
	}

	// Capture the request body
	transport.roundTripFunc = func(req *http.Request) (*http.Response, error) {
		if req.Method == "POST" && strings.HasSuffix(req.URL.Path, "/comments") {
			// Read the body
			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(strings.NewReader(string(body))) // Reset body for actual request

			var comment struct {
				Body string `json:"body"`
			}
			if err := json.Unmarshal(body, &comment); err == nil {
				capturedBody = comment.Body
			}
		}
		// Use default behavior
		return defaultRoundTrip(req)
	}

	// Set up empty comments response for GET
	// GitHub client adds per_page parameter
	transport.responses["/repos/testowner/testrepo/issues/789/comments?per_page=100"] = mockResponse{
		statusCode: 200,
		body:       "[]",
	}

	// Set up create response for POST
	transport.responses["/repos/testowner/testrepo/issues/789/comments-POST"] = mockResponse{
		statusCode: 201,
		body:       `{"id": 12345}`,
	}

	httpClient := &http.Client{Transport: transport}
	client := github.NewClient(httpClient)

	type testData struct {
		Key   string `json:"key"`
		Value int    `json:"value"`
	}

	state := NewState[testData]("my-bot", client, testResource(789))

	err := state.Commit(ctx, &testData{Key: "test", Value: 42}, "Test message for state")
	if err != nil {
		t.Fatalf("State.Commit() error = %v", err)
	}

	if capturedBody == "" {
		t.Fatal("No comment body was captured")
	}

	// Check identity marker
	if !strings.HasPrefix(capturedBody, "<!--my-bot-->") {
		t.Errorf("Comment doesn't start with identity marker: %q", capturedBody)
	}

	// Check message
	if !strings.Contains(capturedBody, "Test message for state") {
		t.Errorf("Comment doesn't contain message: %q", capturedBody)
	}

	// Check state markers
	if !strings.Contains(capturedBody, "<!--my-bot-state-->") {
		t.Errorf("Comment doesn't contain state start marker: %q", capturedBody)
	}

	if !strings.Contains(capturedBody, "<!--/my-bot-state-->") {
		t.Errorf("Comment doesn't contain state end marker: %q", capturedBody)
	}

	// Extract and verify JSON
	startMarker := "<!--my-bot-state-->\n<!--\n"
	endMarker := "\n-->\n<!--/my-bot-state-->"

	startIdx := strings.Index(capturedBody, startMarker)
	if startIdx == -1 {
		t.Fatal("Could not find state start marker")
	}
	startIdx += len(startMarker)

	endIdx := strings.Index(capturedBody[startIdx:], endMarker)
	if endIdx == -1 {
		t.Fatal("Could not find state end marker")
	}

	jsonData := capturedBody[startIdx : startIdx+endIdx]

	var decoded testData
	if err := json.Unmarshal([]byte(jsonData), &decoded); err != nil {
		t.Errorf("Failed to decode JSON from comment: %v\nJSON: %s", err, jsonData)
	}

	if decoded.Key != "test" || decoded.Value != 42 {
		t.Errorf("Decoded data = %+v, want {Key:test Value:42}", decoded)
	}
}

func TestStateManager(t *testing.T) {
	sm := NewStateManager("test-identity")

	if sm.Identity() != "test-identity" {
		t.Errorf("StateManager.Identity() = %v, want %v", sm.Identity(), "test-identity")
	}
}
