/*
Copyright 2023 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package quit

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	OtelSidecarPort = 31415
)

// quit.Sidecar explicitly manages the lifecycle of an always-on sidecar that
// serves a /quitquitquit endpoint. When a Job is injected with an an always-on
// container, such Job will never complete.
//
// This utility function sends a POST to the sidecar's /quitquitquit
// endpoint. Jobs can use this to terminate the sidecar upon completion.
func Sidecar(port int) func() {
	return func() {
		var err error
		var resp *http.Response
		for i := 0; i < 5; i++ {
			if i > 1 {
				time.Sleep(1 * time.Second)
			}
			resp, err := http.Post(fmt.Sprintf("http://localhost:%d/quitquitquit", port), "application/json", nil)
			// if err is nil and resp is OK
			if err == nil && resp.StatusCode == http.StatusOK {
				log.Println("successfully POST /quitquitquit to otel-collector sidecar")
				return
			}
		}
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		// This can happen because we don't always run the otel-collector sidecar.
		log.Printf("cannot POST /quitquitquit to the otel-collector sidecar: err=%v, code=%d", err, code)
	}
}

// quit.OtelSidecar is quit.Sidecar for the otel sidecar.
func OtelSidecar() func() {
	return Sidecar(OtelSidecarPort)
}
