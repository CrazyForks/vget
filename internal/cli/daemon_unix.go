//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr sets platform-specific process attributes for daemon mode
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
}
