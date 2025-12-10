//go:build windows

package cli

import (
	"os/exec"
)

// setSysProcAttr sets platform-specific process attributes for daemon mode
// On Windows, we don't set Setsid as it's Unix-only
func setSysProcAttr(cmd *exec.Cmd) {
	// No special attributes needed for Windows
}
