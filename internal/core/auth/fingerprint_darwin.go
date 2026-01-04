//go:build darwin

package auth

import (
	"os/exec"
	"strings"
)

// getMachineID returns the IOPlatformUUID on macOS.
// This is hardware-tied and survives OS reinstalls.
func getMachineID() string {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return ""
	}

	// Parse IOPlatformUUID from output
	// Example line: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				uuid := strings.TrimSpace(parts[1])
				uuid = strings.Trim(uuid, `"`)
				return uuid
			}
		}
	}
	return ""
}

// getHardwareSerial returns the hardware serial number on macOS.
func getHardwareSerial() string {
	out, err := exec.Command("system_profiler", "SPHardwareDataType").Output()
	if err != nil {
		return ""
	}

	// Parse Serial Number from output
	// Example line: Serial Number (system): XXXXXXXXXXXX
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Serial Number") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
