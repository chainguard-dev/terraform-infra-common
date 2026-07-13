/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package trampoline_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/chainguard-dev/terraform-infra-common/modules/zendesk-events/internal/trampoline"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

func ExampleNewServer() {
	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		panic(err)
	}

	s := trampoline.NewServer(client, [][]byte{[]byte("my-secret")})

	// The server implements http.Handler and rejects requests with missing
	// or invalid signatures.
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	fmt.Println(w.Code)
	// Output: 403
}
