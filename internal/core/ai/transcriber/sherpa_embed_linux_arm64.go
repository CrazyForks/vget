//go:build !cgo && linux && arm64

package transcriber

import "fmt"

// AI features are not available on Linux.
func extractSherpaBinary() (string, error) {
	return "", fmt.Errorf("AI features are not available on Linux")
}
