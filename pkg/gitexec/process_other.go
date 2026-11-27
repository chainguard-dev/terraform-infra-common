//go:build !unix

/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import "os/exec"

// stopGroupOnCancel bounds how long Run waits for git's output after the
// command's context ends or git exits. Without process groups, a helper that
// outlives git is not signalled.
func stopGroupOnCancel(cmd *exec.Cmd) {
	cmd.WaitDelay = stopGracePeriod
}

// killGroup does nothing without process groups.
func killGroup(*exec.Cmd) {}
