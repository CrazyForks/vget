//go:build !cgo

package transcriber

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guiyumin/vget/internal/core/config"
)

// SherpaRunner transcribes audio using sherpa-onnx CLI binary.
// This is used when CGO is disabled (CGO_ENABLED=0).
// GPU-enabled binary is embedded: CoreML on macOS ARM64, CUDA on Windows.
type SherpaRunner struct {
	binaryPath string
	modelPath  string
	language   string
}

// NewSherpaRunner creates a new sherpa-onnx runner.
// Uses embedded GPU-enabled binary (CoreML on macOS ARM64, CUDA on Windows).
func NewSherpaRunner(modelPath, language string) (*SherpaRunner, error) {
	// Validate model directory exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("sherpa-onnx model not found: %s", modelPath)
	}

	// Validate required model files exist
	requiredFiles := []string{"encoder.int8.onnx", "decoder.int8.onnx", "joiner.int8.onnx", "tokens.txt"}
	for _, file := range requiredFiles {
		path := filepath.Join(modelPath, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("required model file not found: %s", path)
		}
	}

	// Extract embedded binary (GPU-enabled: CoreML/CUDA)
	binaryPath, err := extractSherpaBinary()
	if err != nil {
		return nil, err
	}

	return &SherpaRunner{
		binaryPath: binaryPath,
		modelPath:  modelPath,
		language:   language,
	}, nil
}

// NewSherpaRunnerFromConfig creates a SherpaRunner from config.
func NewSherpaRunnerFromConfig(cfg config.LocalASRConfig, modelsDir string) (*SherpaRunner, error) {
	modelName := cfg.Model
	if modelName == "" {
		modelName = "parakeet-v3"
	}

	// Look up model in registry to get the correct directory name
	model := GetModel(modelName)
	var modelDir string
	if model != nil {
		modelDir = model.DirName
	} else {
		modelDir = modelName
	}

	modelPath := filepath.Join(modelsDir, modelDir)

	language := cfg.Language
	if language == "" {
		language = "auto"
	}

	return NewSherpaRunner(modelPath, language)
}

// Name returns the provider name.
func (s *SherpaRunner) Name() string {
	return "sherpa-onnx"
}

// Transcribe converts an audio file to text using sherpa-onnx CLI.
func (s *SherpaRunner) Transcribe(ctx context.Context, filePath string) (*Result, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Convert audio to WAV if needed
	wavPath, cleanup, err := s.ensureWAV(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare audio: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	// Build command arguments
	// sherpa-onnx-offline --encoder=X --decoder=X --joiner=X --tokens=X --model-type=nemo_transducer audio.wav
	args := []string{
		fmt.Sprintf("--encoder=%s", filepath.Join(s.modelPath, "encoder.int8.onnx")),
		fmt.Sprintf("--decoder=%s", filepath.Join(s.modelPath, "decoder.int8.onnx")),
		fmt.Sprintf("--joiner=%s", filepath.Join(s.modelPath, "joiner.int8.onnx")),
		fmt.Sprintf("--tokens=%s", filepath.Join(s.modelPath, "tokens.txt")),
		"--model-type=nemo_transducer",
		"--decoding-method=greedy_search",
	}

	// Use available CPU threads
	numThreads := runtime.NumCPU()
	if numThreads > 8 {
		numThreads = 8
	}
	args = append(args, fmt.Sprintf("--num-threads=%d", numThreads))

	// Add the audio file as positional argument
	args = append(args, wavPath)

	fmt.Printf("  Running sherpa-onnx...\\n")
	fmt.Printf("  Model: %s\\n", filepath.Base(s.modelPath))
	fmt.Printf("  Threads: %d\\n", numThreads)

	// Run sherpa-onnx
	cmd := exec.CommandContext(ctx, s.binaryPath, args...)

	// Capture stdout for results
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Capture stderr for progress/errors
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start sherpa-onnx: %w", err)
	}

	// Read stderr for progress info
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "progress") || strings.Contains(line, "%") {
				fmt.Printf("  %s\\n", line)
			}
		}
	}()

	// Read stdout for transcription result
	var outputLines []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("sherpa-onnx failed: %w", err)
	}

	// Parse output - sherpa-onnx outputs the transcription text
	text := strings.TrimSpace(strings.Join(outputLines, " "))

	// sherpa-onnx doesn't output timestamps by default, create single segment
	var segments []Segment
	if text != "" {
		// Get audio duration
		duration, _ := getAudioDuration(wavPath)
		segments = []Segment{
			{
				Start: 0,
				End:   duration,
				Text:  text,
			},
		}
	}

	// Get audio duration
	duration, _ := getAudioDuration(wavPath)

	return &Result{
		RawText:  text,
		Segments: segments,
		Language: s.language,
		Duration: duration,
	}, nil
}

// Close is a no-op for the runner.
func (s *SherpaRunner) Close() error {
	return nil
}

// SupportsLanguage checks if Parakeet V3 supports a language.
// Parakeet V3 supports 25 European languages.
func (s *SherpaRunner) SupportsLanguage(lang string) bool {
	parakeetLangs := map[string]bool{
		"bg": true, "hr": true, "cs": true, "da": true, "nl": true,
		"en": true, "et": true, "fi": true, "fr": true, "de": true,
		"el": true, "hu": true, "it": true, "lv": true, "lt": true,
		"mt": true, "pl": true, "pt": true, "ro": true, "sk": true,
		"sl": true, "es": true, "sv": true, "ru": true, "uk": true,
	}
	return parakeetLangs[lang]
}

// MaxFileSize returns 0 - local sherpa-onnx has no file size limit.
func (s *SherpaRunner) MaxFileSize() int64 {
	return 0
}

// ensureWAV converts audio to WAV format if needed.
// Uses the same conversion logic as whisper_runner.go.
func (s *SherpaRunner) ensureWAV(filePath string) (string, func(), error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	// If already WAV, use as-is
	if ext == ".wav" {
		return filePath, nil, nil
	}

	// Convert to WAV using embedded ffmpeg
	tmpFile, err := os.CreateTemp("", "sherpa-*.wav")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpPath)
	}

	// Try pure Go decoders first
	var samples []float32
	var sampleRate int

	switch ext {
	case ".mp3":
		samples, sampleRate, err = readMP3Samples(filePath)
	case ".flac":
		samples, sampleRate, err = readFLACSamples(filePath)
	default:
		// Use embedded ffmpeg WASM for other formats
		err = convertWithFFmpeg(filePath, tmpPath)
		if err != nil {
			cleanup()
			return "", nil, err
		}
		return tmpPath, cleanup, nil
	}

	if err != nil {
		cleanup()
		return "", nil, err
	}

	// Resample to 16kHz if needed
	if sampleRate != 16000 {
		samples = resampleTo16kHz(samples, sampleRate)
	}

	// Write WAV file
	if err := writeWAV(tmpPath, samples, 16000); err != nil {
		cleanup()
		return "", nil, err
	}

	return tmpPath, cleanup, nil
}
