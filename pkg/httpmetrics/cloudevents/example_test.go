/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package cloudevents_test

import (
	"context"

	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics/cloudevents"
)

func ExampleNewClientHTTP() {
	c, err := cloudevents.NewClientHTTP("my-client")
	if err != nil {
		panic(err)
	}
	_ = c
}

func ExampleWithTarget() {
	ctx := context.Background()
	opts := cloudevents.WithTarget(ctx, "http://localhost:8080")
	_ = opts
}
