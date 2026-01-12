//go:build cgo && !noai

package transcriber

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/go-audio/wav"
	"github.com/guiyumin/vget/internal/core/config"
	"github.com/hajimehoshi/go-mp3"
	"github.com/mewkiz/flac"
	"github.com/tetratelabs/wazero"
)

// WhisperTranscriber implements Transcriber using whisper.cpp.
type WhisperTranscriber struct {
	model     whisper.Model
	modelPath string
	language  string
}

// NewWhisperTranscriber creates a new whisper.cpp transcriber.
func NewWhisperTranscriber(modelPath, language string) (*WhisperTranscriber, error) {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("whisper model not found: %s", modelPath)
	}

	model, err := whisper.New(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load whisper model: %w", err)
	}

	return &WhisperTranscriber{
		model:     model,
		modelPath: modelPath,
		language:  language,
	}, nil
}

// NewWhisperTranscriberFromConfig creates a WhisperTranscriber from config.
func NewWhisperTranscriberFromConfig(cfg config.LocalASRConfig, modelsDir string) (*WhisperTranscriber, error) {
	modelName := cfg.Model
	if modelName == "" {
		modelName = DefaultModel
	}

	// Look up model in registry to get the correct filename
	model := GetModel(modelName)
	var modelFile string
	if model != nil {
		modelFile = model.DirName
	} else {
		// Assume it's a direct path or filename
		modelFile = modelName
		if !strings.HasSuffix(modelFile, ".bin") {
			modelFile = modelFile + ".bin"
		}
	}

	modelPath := filepath.Join(modelsDir, modelFile)

	language := cfg.Language
	if language == "" {
		language = "auto"
	}

	return NewWhisperTranscriber(modelPath, language)
}

// Name returns the provider name.
func (w *WhisperTranscriber) Name() string {
	return "whisper.cpp"
}

// Transcribe converts an audio file to text using whisper.cpp.
func (w *WhisperTranscriber) Transcribe(ctx context.Context, filePath string) (*Result, error) {
	// Check for context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Read audio samples (supports WAV, MP3 natively; other formats via ffmpeg)
	samples, sampleRate, err := w.readAudioSamples(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio: %w", err)
	}
	fmt.Printf("  Audio: %d samples, %d Hz, %.1f seconds\n", len(samples), sampleRate, float64(len(samples))/float64(sampleRate))

	// Create whisper context
	wctx, err := w.model.NewContext()
	if err != nil {
		return nil, fmt.Errorf("failed to create whisper context: %w", err)
	}

	// ===== WHISPER CONFIGURATION (v2025-12-31) =====
	fmt.Println("  ========================================")
	fmt.Println("  WHISPER.CPP TRANSCRIPTION CONFIG v2")
	fmt.Println("  ========================================")

	// Performance optimization: use all available CPU cores
	numThreads := runtime.NumCPU()
	if numThreads > 8 {
		numThreads = 8 // Cap at 8 threads for diminishing returns
	}
	wctx.SetThreads(uint(numThreads))
	fmt.Printf("  Threads: %d\n", numThreads)

	// CRITICAL: Disable translation - transcribe in original language, NOT English
	wctx.SetTranslate(false)
	fmt.Println("  Translate: false (transcribe, not translate)")

	// CRITICAL: Use sentence-level segments, not token-level fragments
	wctx.SetTokenTimestamps(false)
	fmt.Println("  TokenTimestamps: false (sentence-level output)")

	// Set language (required)
	if w.language != "" {
		if err := wctx.SetLanguage(w.language); err != nil {
			fmt.Printf("  Warning: failed to set language %s: %v\n", w.language, err)
		} else {
			fmt.Printf("  Language: %s\n", w.language)
		}
	}
	fmt.Println("  ========================================")

	// Progress callback for real-time feedback
	lastProgress := -1
	progressCb := func(progress int) {
		if progress != lastProgress && progress%10 == 0 {
			fmt.Printf("  Whisper progress: %d%%\n", progress)
			lastProgress = progress
		}
	}

	// Process audio with progress callback
	if err := wctx.Process(samples, nil, nil, progressCb); err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	// Collect segments
	var segments []Segment
	var fullText strings.Builder

	segCount := 0
	for {
		segment, err := wctx.NextSegment()
		if err != nil {
			break
		}
		segCount++

		segments = append(segments, Segment{
			Start: segment.Start,
			End:   segment.End,
			Text:  segment.Text,
		})

		fullText.WriteString(segment.Text)
		fullText.WriteString(" ")
	}
	fmt.Printf("  Segments: %d\n", segCount)

	// Calculate duration
	duration := time.Duration(float64(len(samples))/float64(sampleRate)) * time.Second

	return &Result{
		RawText:  strings.TrimSpace(fullText.String()),
		Segments: segments,
		Language: w.language,
		Duration: duration,
	}, nil
}

// Close releases the model resources.
func (w *WhisperTranscriber) Close() error {
	if w.model != nil {
		return w.model.Close()
	}
	return nil
}

// readAudioSamples reads audio samples from various formats.
// Supports WAV, MP3, FLAC natively (pure Go, no external dependencies).
// For other formats (M4A, AAC, OGG, etc.), uses embedded ffmpeg WASM.
func (w *WhisperTranscriber) readAudioSamples(filePath string) ([]float32, int, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".wav":
		return w.readWAVSamples(filePath)
	case ".mp3":
		return w.readMP3Samples(filePath)
	case ".flac":
		return w.readFLACSamples(filePath)
	default:
		// Use embedded ffmpeg WASM for other formats (m4a, aac, ogg, etc.)
		return w.readWithEmbeddedFFmpeg(filePath)
	}
}

// readWAVSamples reads a WAV file and returns float32 samples.
func (w *WhisperTranscriber) readWAVSamples(filePath string) ([]float32, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open WAV file: %w", err)
	}
	defer file.Close()

	decoder := wav.NewDecoder(file)
	if !decoder.IsValidFile() {
		return nil, 0, fmt.Errorf("invalid WAV file")
	}

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode WAV: %w", err)
	}

	// Convert to float32 (normalize 16-bit samples)
	const maxInt16 = 32768.0
	samples := make([]float32, len(buf.Data))
	for i, sample := range buf.Data {
		samples[i] = float32(sample) / maxInt16
	}

	sampleRate := int(decoder.SampleRate)

	// Resample to 16kHz if needed (whisper expects 16kHz)
	if sampleRate != 16000 {
		samples = resampleTo16kHz(samples, sampleRate)
		sampleRate = 16000
	}

	return samples, sampleRate, nil
}

// readMP3Samples reads an MP3 file and returns float32 samples at 16kHz.
func (w *WhisperTranscriber) readMP3Samples(filePath string) ([]float32, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open MP3 file: %w", err)
	}
	defer file.Close()

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode MP3: %w", err)
	}

	// Read all samples (MP3 decoder outputs 16-bit stereo PCM at original sample rate)
	sampleRate := decoder.SampleRate()

	// Read all bytes
	data, err := io.ReadAll(decoder)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read MP3 data: %w", err)
	}

	// Convert bytes to float32 samples
	// go-mp3 outputs 16-bit stereo PCM (4 bytes per sample pair: L16 + R16)
	numSamples := len(data) / 4 // stereo 16-bit = 4 bytes per sample pair
	samples := make([]float32, numSamples)

	const maxInt16 = 32768.0
	for i := 0; i < numSamples; i++ {
		// Read left channel (16-bit little-endian), ignore right channel (mono mix)
		left := int16(data[i*4]) | int16(data[i*4+1])<<8
		right := int16(data[i*4+2]) | int16(data[i*4+3])<<8
		// Mix to mono
		mono := (int32(left) + int32(right)) / 2
		samples[i] = float32(mono) / maxInt16
	}

	// Resample to 16kHz (whisper expects 16kHz)
	if sampleRate != 16000 {
		samples = resampleTo16kHz(samples, sampleRate)
	}

	return samples, 16000, nil
}

// readFLACSamples reads a FLAC file and returns float32 samples at 16kHz.
func (w *WhisperTranscriber) readFLACSamples(filePath string) ([]float32, int, error) {
	// Open and decode FLAC file
	stream, err := flac.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open FLAC file: %w", err)
	}
	defer stream.Close()

	sampleRate := int(stream.Info.SampleRate)
	nChannels := int(stream.Info.NChannels)
	bitsPerSample := int(stream.Info.BitsPerSample)

	// Read all frames
	var samples []float32
	maxVal := float32(int64(1) << (bitsPerSample - 1))

	for {
		frame, err := stream.ParseNext()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse FLAC frame: %w", err)
		}

		// Convert samples to float32 mono
		nSamples := len(frame.Subframes[0].Samples)
		for i := 0; i < nSamples; i++ {
			var mono int64
			for ch := 0; ch < nChannels; ch++ {
				mono += int64(frame.Subframes[ch].Samples[i])
			}
			mono /= int64(nChannels)
			samples = append(samples, float32(mono)/maxVal)
		}
	}

	// Resample to 16kHz if needed
	if sampleRate != 16000 {
		samples = resampleTo16kHz(samples, sampleRate)
	}

	return samples, 16000, nil
}

// readWithEmbeddedFFmpeg uses embedded ffmpeg WASM to convert audio formats.
// No external ffmpeg installation required.
func (w *WhisperTranscriber) readWithEmbeddedFFmpeg(filePath string) ([]float32, int, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	fmt.Printf("  Converting %s using embedded ffmpeg...\n", ext)

	// Create temp WAV file for output
	tmpFile, err := os.CreateTemp("", "whisper-*.wav")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Get absolute paths (required for WASM filesystem)
	absInput, err := filepath.Abs(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get absolute path: %w", err)
	}
	absOutput, err := filepath.Abs(tmpPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Get directories for mounting
	inputDir := filepath.Dir(absInput)
	outputDir := filepath.Dir(absOutput)

	// Run embedded ffmpeg to convert to 16kHz mono WAV
	ctx := context.Background()
	args := wasm.Args{
		Stderr: io.Discard,
		Stdout: io.Discard,
		Args: []string{
			"-i", absInput,
			"-ar", "16000",
			"-ac", "1",
			"-c:a", "pcm_s16le",
			"-y",
			absOutput,
		},
		// Mount filesystem directories for WASM access
		Config: func(cfg wazero.ModuleConfig) wazero.ModuleConfig {
			cfg = cfg.WithFSConfig(wazero.NewFSConfig().
				WithDirMount(inputDir, inputDir).
				WithDirMount(outputDir, outputDir))
			return cfg
		},
	}

	rc, err := ffmpreg.Ffmpeg(ctx, args)
	if err != nil {
		return nil, 0, fmt.Errorf("ffmpeg WASM failed: %w", err)
	}
	if rc != 0 {
		return nil, 0, fmt.Errorf("ffmpeg WASM exited with code %d", rc)
	}

	// Read the converted WAV
	return w.readWAVSamples(tmpPath)
}

// resampleTo16kHz resamples audio to 16kHz using linear interpolation.
// This is a simple resampler - good enough for speech recognition.
func resampleTo16kHz(samples []float32, srcRate int) []float32 {
	if srcRate == 16000 {
		return samples
	}

	ratio := float64(srcRate) / 16000.0
	newLen := int(float64(len(samples)) / ratio)
	resampled := make([]float32, newLen)

	for i := 0; i < newLen; i++ {
		srcPos := float64(i) * ratio
		srcIdx := int(srcPos)
		frac := float32(srcPos - float64(srcIdx))

		if srcIdx+1 < len(samples) {
			// Linear interpolation
			resampled[i] = samples[srcIdx]*(1-frac) + samples[srcIdx+1]*frac
		} else if srcIdx < len(samples) {
			resampled[i] = samples[srcIdx]
		}
	}

	return resampled
}

// SupportsLanguage returns true - Whisper supports 99+ languages.
func (w *WhisperTranscriber) SupportsLanguage(lang string) bool {
	return true
}

// MaxFileSize returns 0 - local whisper.cpp has no file size limit.
func (w *WhisperTranscriber) MaxFileSize() int64 {
	return 0
}
