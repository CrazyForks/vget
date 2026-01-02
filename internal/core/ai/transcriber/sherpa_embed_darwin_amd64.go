//go:build !cgo && darwin && amd64

package transcriber

import "fmt"

// AI features are not available on Intel Macs.
func extractSherpaBinary() (string, error) {
	return "", fmt.Errorf("AI features are not available on Intel Macs. Please use a Mac with Apple Silicon (M1/M2/M3/M4)")
}
