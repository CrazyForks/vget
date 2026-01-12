//go:build noai

// Package transcriber provides speech-to-text transcription.
// This file contains stub implementations when built with -tags noai.
package transcriber

import (
	"fmt"

	"github.com/guiyumin/vget/internal/core/config"
)

// LocalTranscriber is a stub for noai builds.
type LocalTranscriber struct {
	Transcriber
	modelName string
}

// SetProgressReporter is a no-op for noai builds.
func (lt *LocalTranscriber) SetProgressReporter(reporter *ProgressReporter) {
	// No-op
}

// GetModelName returns empty string for noai builds.
func (lt *LocalTranscriber) GetModelName() string {
	return ""
}

// NewLocal returns an error in noai builds - local AI is not available.
func NewLocal(cfg config.LocalASRConfig) (*LocalTranscriber, error) {
	return nil, fmt.Errorf("local AI transcription is not available in this build. Please use the GPU-enabled container or cloud transcription (OpenAI)")
}

// TUIProgressReporter is a stub for noai builds.
type TUIProgressReporter struct {
	*ProgressReporter
}

// NewTUIProgressReporter creates a stub TUI progress reporter for noai builds.
func NewTUIProgressReporter() *TUIProgressReporter {
	return &TUIProgressReporter{
		ProgressReporter: &ProgressReporter{},
	}
}

// RunTranscribeTUI is a stub for noai builds - does nothing and returns nil.
func RunTranscribeTUI(filename, model string, reporter *TUIProgressReporter) error {
	return nil
}
