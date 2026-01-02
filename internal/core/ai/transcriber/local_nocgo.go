//go:build !cgo

package transcriber

import (
	"fmt"
	"strings"

	"github.com/guiyumin/vget/internal/core/config"
)

// NewLocal creates a local transcriber using embedded whisper.cpp binary.
// This works without CGO by using exec.Command to run the embedded whisper binary.
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

	// Only whisper models are supported in non-CGO builds
	if !strings.HasPrefix(model, "whisper") {
		return nil, fmt.Errorf("only whisper models are supported in CGO_ENABLED=0 builds (got %q)", model)
	}

	fmt.Printf("  Using Engine: whisper.cpp (embedded binary)\n")
	fmt.Printf("=========================================\n")

	return NewWhisperRunnerFromConfig(cfg, modelsDir)
}
