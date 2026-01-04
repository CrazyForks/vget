//go:build windows

package auth

import (
	"os/exec"
	"strings"
)

// getMachineID returns the MachineGuid on Windows.
// This is generated during Windows installation.
// Note: Changes on OS reinstall.
func getMachineID() string {
	out, err := exec.Command("reg", "query",
		`HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Cryptography`,
		"/v", "MachineGuid").Output()
	if err != nil {
		return ""
	}

	// Parse MachineGuid from output
	// Example: MachineGuid    REG_SZ    xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "MachineGuid") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}

// getHardwareSerial returns the BIOS serial number on Windows.
func getHardwareSerial() string {
	out, err := exec.Command("wmic", "bios", "get", "serialnumber").Output()
	if err != nil {
		return ""
	}

	// Parse serial from output (second line after header)
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 {
			continue // Skip header
		}
		serial := strings.TrimSpace(line)
		if serial != "" && serial != "To Be Filled By O.E.M." {
			return serial
		}
	}
	return ""
}
