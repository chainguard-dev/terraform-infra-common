/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package octosts_test

import (
	"fmt"

	"github.com/chainguard-dev/terraform-infra-common/modules/github-bots/sdk/octosts"
)

func ExampleIsBotUser() {
	fmt.Println(octosts.IsBotUser("octo-sts[bot]"))
	fmt.Println(octosts.IsBotUser("octo-sts-2[bot]"))
	fmt.Println(octosts.IsBotUser("regular-user"))
	// Output:
	// true
	// true
	// false
}
