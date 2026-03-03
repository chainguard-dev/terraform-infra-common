/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package octosts

import "regexp"

// botPattern matches "octo-sts[bot]", "octo-sts-2[bot]", "octo-sts-99[bot]", etc.
var botPattern = regexp.MustCompile(`^octo-sts(-\d+)?\[bot\]$`)

// IsBotUser reports whether the given login belongs to an octo-sts bot user.
func IsBotUser(login string) bool {
	return botPattern.MatchString(login)
}
