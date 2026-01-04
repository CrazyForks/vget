//go:build linux

package auth

import (
	"os"
	"os/exec"
	"strings"
)

// getMachineID returns the machine-id on Linux.
// This is generated on first boot and persists across reboots.
// Note: Changes on OS reinstall.
func getMachineID() string {
	// Try /etc/machine-id first (systemd)
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		return strings.TrimSpace(string(data))
	}

	// Fallback to /var/lib/dbus/machine-id
	if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		return strings.TrimSpace(string(data))
	}

	return ""
}

// getHardwareSerial returns the hardware serial on Linux.
// Requires dmidecode which typically needs root access.
// Returns empty string if not available (graceful degradation).
func getHardwareSerial() string {
	// Try reading from sysfs first (doesn't need root)
	if data, err := os.ReadFile("/sys/class/dmi/id/product_serial"); err == nil {
		serial := strings.TrimSpace(string(data))
		if serial != "" && serial != "To Be Filled By O.E.M." {
			return serial
		}
	}

	// Fallback to dmidecode (needs root, may fail)
	out, err := exec.Command("dmidecode", "-s", "system-serial-number").Output()
	if err != nil {
		return ""
	}

	serial := strings.TrimSpace(string(out))
	if serial == "To Be Filled By O.E.M." || serial == "Not Specified" {
		return ""
	}
	return serial
}
