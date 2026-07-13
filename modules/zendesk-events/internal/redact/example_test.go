/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package redact_test

import (
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/modules/zendesk-events/internal/redact"
)

func ExampleBody() {
	// Free-text fields (subject) are dropped; allowlisted technical signal
	// (account_id, id, detail.status, detail.organization_id) is retained.
	in := []byte(`{"account_id":12345,"id":"evt-1","subject":"Ada cannot pull","detail":{"status":"open","organization_id":"7"}}`)
	fmt.Println(string(redact.Body(in)))
	// Output: {"account_id":12345,"detail":{"organization_id":"7","status":"open"},"id":"evt-1"}
}

func ExampleString() {
	// String scrubs identifying tokens (emails, IPs) from a retained value or
	// the CloudEvent subject attribute.
	fmt.Println(redact.String("contact ada@acme.com from 10.1.2.3 or 2001:db8::1"))
	// Output: contact <EMAIL> from <IP> or <IP>
}
