//go:build !cgo && linux && amd64

package transcriber

import "fmt"

// Local transcription is not available on Linux CLI.
// Use cloud transcription (OpenAI) via Docker/API instead.
func extractSherpaBinary() (string, error) {
	return "", fmt.Errorf("local transcription is not available on Linux. Use cloud transcription (OpenAI) instead")
}
