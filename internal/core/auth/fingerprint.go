package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/user"
	"strings"
)

// GetDeviceFingerprint generates a stable device fingerprint.
// Uses machine ID (OS-level) and hardware serial (hardware-level) when available.
func GetDeviceFingerprint() string {
	parts := []string{getMachineID()}

	if serial := getHardwareSerial(); serial != "" {
		parts = append(parts, serial)
	}

	// Add hostname and username as additional signals
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		parts = append(parts, hostname)
	}
	if u, err := user.Current(); err == nil && u != nil {
		parts = append(parts, u.Username)
	}

	h := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(h[:16]) // 32 hex chars
}
