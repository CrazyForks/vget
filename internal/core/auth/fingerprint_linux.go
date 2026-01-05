//go:build linux

package auth

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// isDocker detects if we're running inside a Docker container.
func isDocker() bool {
	// Check for /.dockerenv file (most reliable)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker/containerd
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}

	return false
}

// getDockerDeviceID returns a persistent device ID for Docker containers.
// The ID is stored in ~/.config/vget/device_id and persists across container restarts
// as long as the config volume is mounted.
func getDockerDeviceID() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	deviceIDPath := filepath.Join(home, ".config", "vget", "device_id")

	// Try to read existing device ID
	if data, err := os.ReadFile(deviceIDPath); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id
		}
	}

	// Generate new device ID
	newID := uuid.New().String()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(deviceIDPath), 0755); err != nil {
		return newID // Return without persisting
	}

	// Save device ID for future use
	if err := os.WriteFile(deviceIDPath, []byte(newID), 0600); err != nil {
		return newID // Return without persisting
	}

	return newID
}

// getMachineID returns the machine-id on Linux.
// In Docker containers, uses a persistent device ID file.
// On native Linux, uses /etc/machine-id.
func getMachineID() string {
	// Docker container: use persistent device ID
	if isDocker() {
		return getDockerDeviceID()
	}

	// Native Linux: try /etc/machine-id first (systemd)
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
