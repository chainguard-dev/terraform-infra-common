/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package rotate_test

import (
	"time"

	"github.com/chainguard-dev/terraform-infra-common/pkg/rotate"
)

func ExampleNewUploader() {
	// NewUploader creates an Uploader that periodically uploads log files
	// from source to a cloud blob bucket.
	u := rotate.NewUploader("/var/log/myapp", "gs://my-bucket/logs", 5*time.Minute)
	_ = u
}
