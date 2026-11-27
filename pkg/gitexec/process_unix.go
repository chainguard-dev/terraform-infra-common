//go:build unix

/*
Copyright 2026 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gitexec

import (
	"errors"
	"os/exec"
	"syscall"
)

// stopGroupOnCancel starts git in its own process group and, when the
// command's context ends, signals the whole group. git runs helpers such as
// git-remote-https that inherit its stderr; killing git alone leaves them
// holding the pipe Run reads, so Run would wait until they exit on their own.
func stopGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// A caller that replaced SysProcAttr may have turned the group off,
		// and then there is no group to signal.
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = stopGracePeriod
}

// killGroup kills what is left of cmd's process group. WaitDelay kills only
// git itself, so a helper that ignored SIGTERM survives it.
//
// git has been reaped by now, but its pid stays reserved as the group's id
// for as long as any member, such as a surviving helper, is alive. Only once
// the group is empty, with nothing left to kill, could the kernel hand the
// number to a new group leader before this signal, which would take the pid
// space wrapping in that window.
func killGroup(cmd *exec.Cmd) {
	// Only a group git leads is git's to kill: with a Pgid set, git joined
	// an existing group, and its pid was never a group id.
	if cmd.Process == nil || cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setpgid || cmd.SysProcAttr.Pgid != 0 {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
