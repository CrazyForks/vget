//go:build !cgo && darwin && arm64

package transcriber

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed bin/sherpa-darwin-arm64
var sherpaBinary []byte

func extractSherpaBinary() (string, error) {
	if len(sherpaBinary) == 0 {
		return "", fmt.Errorf("sherpa-onnx binary not embedded - build with GitHub Actions")
	}

	// Extract to ~/.config/vget/bin/
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	binDir := filepath.Join(configDir, "vget", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", err
	}

	binaryPath := filepath.Join(binDir, "sherpa-darwin-arm64")

	// Check if already extracted and same size
	if info, err := os.Stat(binaryPath); err == nil {
		if info.Size() == int64(len(sherpaBinary)) {
			return binaryPath, nil
		}
	}

	// Extract binary
	if err := os.WriteFile(binaryPath, sherpaBinary, 0755); err != nil {
		return "", err
	}

	return binaryPath, nil
}
