/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package httpmetrics_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"google.golang.org/grpc"
)

func ExampleServerRevisionHandler() {
	// In Cloud Run, supply the process's K_REVISION value at startup.
	handler := httpmetrics.ServerRevisionHandler("api-00042-abc", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	fmt.Println(response.Header().Get(httpmetrics.ServerRevisionHeader))
	// Output: api-00042-abc
}

func ExampleServerRevisionUnaryInterceptor() {
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		// Add authentication before the revision interceptor.
		httpmetrics.ServerRevisionUnaryInterceptor("api-00042-abc"),
	))
	defer server.Stop()
}

func ExampleServerRevisionStreamInterceptor() {
	server := grpc.NewServer(grpc.ChainStreamInterceptor(
		// Add authentication before the revision interceptor.
		httpmetrics.ServerRevisionStreamInterceptor("api-00042-abc"),
	))
	defer server.Stop()
}
