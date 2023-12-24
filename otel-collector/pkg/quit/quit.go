/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package quit

import (
	"bytes"
	"log"
	"net/http"
	"time"
)

// Quit explicitly manages the lifecycle of the otel-collector sidecar. When a Job is
// injected with an Istio sidecar, which is an always-on container, such Job
// will never complete.
//
// This utility function sends a POST to otel-collector sidecar's /quitquitquit endpoint.
// Jobs can use this to terminate the sidecar upon completion.
func Quit() {
	var err error
	for i := 0; i < 5; i++ {
		if i > 1 {
			time.Sleep(1 * time.Second)
		}
		_, err = http.Post("http://localhost:31415/quitquitquit", "application/json", nil)
		if err == nil {
			log.Println("successfully POST /quitquitquit to otel-collector sidecar")
			return
		}
	}
	// This can happen because we don't always run the otel-collector sidecar.
	log.Printf("cannot POST /quitquitquit to the otel-collector sidecar: %v", err)
}
