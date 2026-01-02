//go:build !cgo

package transcriber

import (
	"fmt"
	"strings"

	"github.com/guiyumin/vget/internal/core/config"
)

// NewLocal creates a local transcriber using embedded binaries.
// This works without CGO by using exec.Command to run embedded binaries:
// - whisper-* models → whisper.cpp (Metal on macOS, CUDA on Windows)
// - parakeet-* models → sherpa-onnx (CoreML on macOS, CUDA on Windows)
func NewLocal(cfg config.LocalASRConfig) (Transcriber, error) {
	modelsDir := cfg.ModelsDir
	if modelsDir == "" {
		var err error
		modelsDir, err = DefaultModelsDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get models directory: %w", err)
		}
	}

	// Determine engine from model name
	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}

	fmt.Printf("=== LOCAL ASR CONFIG (CGO_ENABLED=0) ===\n")
	fmt.Printf("  Using Model: %q\n", model)
	fmt.Printf("  Models Dir: %s\n", modelsDir)

	// Route to appropriate engine based on model name
	if strings.HasPrefix(model, "whisper") {
		fmt.Printf("  Using Engine: whisper.cpp (embedded binary)\n")
		fmt.Printf("=========================================\n")
		return NewWhisperRunnerFromConfig(cfg, modelsDir)
	} else if strings.HasPrefix(model, "parakeet") {
		fmt.Printf("  Using Engine: sherpa-onnx (embedded binary)\n")
		fmt.Printf("=========================================\n")
		return NewSherpaRunnerFromConfig(cfg, modelsDir)
	}

	return nil, fmt.Errorf("unsupported model: %q (supported: whisper-*, parakeet-*)", model)
}
