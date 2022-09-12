//go:build !linux && !windows
// +build !linux,!windows

package jobsreceiver

import (
	"os/exec"
	"syscall"
)

// SetProcessGroup sets the process group of the command process
func SetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
