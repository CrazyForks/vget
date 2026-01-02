# PRD: AI-Powered vget CLI

## Overview

This document covers CLI-specific implementation for vget AI features. The goal is a **single, self-contained binary** with zero runtime dependencies.

**Key Design Decisions:**
- `CGO_ENABLED=0` - Pure Go binary, no C dependencies
- Runtime binaries downloaded on first use from Cloudflare R2
- Models downloaded on first use from Cloudflare R2
- Pure Go audio decoders for common formats
- Embedded ffmpeg WASM for format conversion

See [ai-powered-vget-prd.md](./ai-powered-vget-prd.md) for shared concepts.

---

## CLI Commands

### Model Management

```bash
# List downloaded models (local, no network)
vget ai models

# List models available to download (from remote)
vget ai models --remote
vget ai models -r

# Download a model (default: from Hugging Face)
vget ai models download whisper-large-v3-turbo
vget ai models download parakeet-v3
vget ai models download piper-en-us

# Download from vmirror.org (faster in China)
vget ai models download whisper-large-v3-turbo --from=vmirror

# Shortcut alias for download
vget ai download whisper-large-v3-turbo              # same as: vget ai models download
vget ai download whisper-large-v3-turbo --from=vmirror

# Remove a model
vget ai models rm whisper-large-v3-turbo
```

### Speech-to-Text

```bash
# Basic transcription (always outputs markdown with timestamps)
vget ai transcribe podcast.mp3                    # → podcast.transcript.md

# Specify language
vget ai transcribe podcast.mp3 --language zh

# Choose model (runtime is determined by model)
vget ai transcribe podcast.mp3 --model whisper-large-v3-turbo  # uses whisper.cpp
vget ai transcribe podcast.mp3 --model parakeet-v3             # uses sherpa-onnx

# Output to specific file
vget ai transcribe podcast.mp3 -o my-transcript.md
```

### Convert Transcript

```bash
# Convert markdown transcript to subtitle formats
vget ai convert podcast.transcript.md --to srt    # → podcast.srt
vget ai convert podcast.transcript.md --to vtt    # → podcast.vtt
vget ai convert podcast.transcript.md --to txt    # → podcast.txt (plain text, no timestamps)

# Specify output file
vget ai convert podcast.transcript.md --to srt -o subtitles.srt
```

### Text-to-Speech (TODO)

> Planned feature - not yet implemented

```bash
# Basic synthesis
vget ai speak "Hello, world" -o hello.wav

# From file
vget ai speak --file article.txt -o article.wav

# Choose voice
vget ai speak "Hello" --voice en-us-lessac -o hello.wav

# List available voices
vget ai voices

# Download a voice
vget ai models download piper-en-us-lessac
```

### OCR (TODO)

> Planned feature - not yet implemented

```bash
# Extract text from image
vget ai ocr screenshot.png

# Specify language(s)
vget ai ocr document.png --language eng,chi_sim

# Process PDF with OCR
vget ai ocr scanned.pdf -o text.md
```

### PDF Processing (TODO)

> Planned feature - not yet implemented

```bash
# Extract text from PDF
vget ai pdf extract document.pdf -o text.md

# OCR scanned PDF
vget ai pdf ocr scanned.pdf -o text.md
```

---

## Architecture

### Binary Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                      vget CLI Binary                            │
│                     (CGO_ENABLED=0)                             │
├─────────────────────────────────────────────────────────────────┤
│  Audio Decoders (Pure Go)                                       │
│  ├── MP3  → go-mp3 (hajimehoshi/go-mp3)                        │
│  ├── WAV  → go-audio/wav                                        │
│  ├── FLAC → mewkiz/flac                                         │
│  └── M4A/AAC/OGG → go-ffmpreg (embedded WASM, ~8MB)            │
├─────────────────────────────────────────────────────────────────┤
│  Runtime Manager                                                │
│  ├── Download binaries from R2 on first use                    │
│  ├── Verify checksums                                           │
│  ├── Extract to ~/.config/vget/bin/                            │
│  └── Execute via exec.Command()                                 │
├─────────────────────────────────────────────────────────────────┤
│  Model Manager                                                  │
│  ├── Download models from R2 on first use                      │
│  ├── Verify checksums                                           │
│  └── Store in ~/.config/vget/models/                           │
└─────────────────────────────────────────────────────────────────┘
```

### Package Structure

```
internal/core/ai/
├── ai.go                     # Main AI orchestrator
├── runtime/
│   ├── runtime.go            # Runtime interface and manager
│   ├── whisper.go            # whisper.cpp runtime
│   ├── sherpa.go             # sherpa-onnx runtime
│   ├── piper.go              # Piper TTS runtime
│   └── tesseract.go          # Tesseract OCR runtime
├── models/
│   ├── models.go             # Model interface and registry
│   ├── whisper.go            # Whisper models
│   ├── parakeet.go           # Parakeet models
│   ├── piper.go              # Piper voice models
│   └── tesseract.go          # Tesseract language data
├── transcriber/
│   ├── transcriber.go        # Transcriber interface
│   ├── whisper.go            # whisper.cpp transcriber
│   └── sherpa.go             # sherpa-onnx transcriber
├── synthesizer/
│   ├── synthesizer.go        # Synthesizer interface
│   └── piper.go              # Piper synthesizer
├── ocr/
│   ├── ocr.go                # OCR interface
│   └── tesseract.go          # Tesseract OCR
├── audio/
│   ├── decoder.go            # Audio decoder interface
│   ├── mp3.go                # MP3 decoder (pure Go)
│   ├── wav.go                # WAV decoder (pure Go)
│   ├── flac.go               # FLAC decoder (pure Go)
│   └── ffmpeg.go             # FFmpeg WASM fallback
├── chunker/
│   └── chunker.go            # Audio chunking
└── output/
    ├── transcript.go         # Transcript formatter
    ├── srt.go                # SRT generator
    └── vtt.go                # VTT generator
```

---

## Runtime Binaries

### Build Matrix

| Runtime | Platform | Acceleration | Size |
|---------|----------|--------------|------|
| whisper.cpp | darwin-arm64 | Metal | ~5MB |
| whisper.cpp | darwin-amd64 | Accelerate | ~5MB |
| whisper.cpp | linux-amd64 | OpenBLAS/AVX2 | ~8MB |
| whisper.cpp | linux-arm64 | OpenBLAS | ~6MB |
| whisper.cpp | windows-amd64 | OpenBLAS | ~8MB |
| sherpa-onnx | darwin-arm64 | CoreML | ~15MB |
| sherpa-onnx | darwin-amd64 | CPU | ~12MB |
| sherpa-onnx | linux-amd64 | CPU | ~12MB |
| sherpa-onnx | linux-arm64 | CPU | ~10MB |
| sherpa-onnx | windows-amd64 | CPU | ~12MB |
| piper | all platforms | CPU | ~8MB |
| tesseract | all platforms | CPU | ~5MB |

### Build Scripts

Runtime binaries are built in GitHub Actions and uploaded to Cloudflare R2.

```
.github/scripts/
├── build-whisper.sh          # Build whisper.cpp
├── build-sherpa.sh           # Build sherpa-onnx CLI
├── build-piper.sh            # Build Piper
├── build-tesseract.sh        # Build Tesseract
└── upload-to-r2.sh           # Upload to Cloudflare R2
```

### GitHub Actions Workflow

```yaml
# .github/workflows/build-runtimes.yml
name: Build AI Runtimes

on:
  workflow_dispatch:
  push:
    paths:
      - '.github/scripts/build-*.sh'

jobs:
  build-whisper:
    strategy:
      matrix:
        include:
          - os: macos-14
            goos: darwin
            goarch: arm64
            acceleration: metal
          - os: macos-13
            goos: darwin
            goarch: amd64
            acceleration: accelerate
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            acceleration: openblas
          - os: ubuntu-24.04-arm
            goos: linux
            goarch: arm64
            acceleration: openblas
          - os: windows-latest
            goos: windows
            goarch: amd64
            acceleration: openblas
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v5
      - name: Build whisper.cpp
        run: .github/scripts/build-whisper.sh
        env:
          ACCELERATION: ${{ matrix.acceleration }}
      - name: Upload to R2
        run: .github/scripts/upload-to-r2.sh
        env:
          R2_ACCOUNT_ID: ${{ secrets.R2_ACCOUNT_ID }}
          R2_ACCESS_KEY: ${{ secrets.R2_ACCESS_KEY }}
          R2_SECRET_KEY: ${{ secrets.R2_SECRET_KEY }}
```

---

## Runtime Manager Implementation

### Runtime Interface

```go
package runtime

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
)

// Runtime represents an external AI runtime binary
type Runtime interface {
    Name() string
    Version() string
    BinaryPath() string

    // EnsureInstalled downloads the binary if not present
    EnsureInstalled(ctx context.Context, progress ProgressFunc) error

    // Execute runs the binary with given arguments
    Execute(ctx context.Context, args ...string) ([]byte, error)
}

type ProgressFunc func(downloaded, total int64)

// Manager handles runtime binary management
type Manager struct {
    baseURL    string // https://dl.vget.dev/bin
    binDir     string // ~/.config/vget/bin
    httpClient *http.Client
}

func NewManager() (*Manager, error) {
    configDir, err := os.UserConfigDir()
    if err != nil {
        return nil, err
    }

    binDir := filepath.Join(configDir, "vget", "bin")
    if err := os.MkdirAll(binDir, 0755); err != nil {
        return nil, err
    }

    return &Manager{
        baseURL:    "https://dl.vget.dev/bin",
        binDir:     binDir,
        httpClient: &http.Client{Timeout: 30 * time.Minute},
    }, nil
}

// GetRuntime returns a runtime by name
func (m *Manager) GetRuntime(name string) (Runtime, error) {
    switch name {
    case "whisper":
        return NewWhisperRuntime(m), nil
    case "sherpa":
        return NewSherpaRuntime(m), nil
    case "piper":
        return NewPiperRuntime(m), nil
    case "tesseract":
        return NewTesseractRuntime(m), nil
    default:
        return nil, fmt.Errorf("unknown runtime: %s", name)
    }
}
```

### Whisper Runtime

```go
package runtime

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
)

type WhisperRuntime struct {
    manager *Manager
    version string
}

func NewWhisperRuntime(m *Manager) *WhisperRuntime {
    return &WhisperRuntime{
        manager: m,
        version: "1.8.2",
    }
}

func (r *WhisperRuntime) Name() string { return "whisper" }
func (r *WhisperRuntime) Version() string { return r.version }

func (r *WhisperRuntime) BinaryPath() string {
    name := fmt.Sprintf("whisper-%s-%s", runtime.GOOS, runtime.GOARCH)
    if runtime.GOOS == "windows" {
        name += ".exe"
    }
    return filepath.Join(r.manager.binDir, name)
}

func (r *WhisperRuntime) downloadURL() string {
    ext := "tar.gz"
    if runtime.GOOS == "windows" {
        ext = "zip"
    }
    return fmt.Sprintf("%s/whisper/v%s/whisper-%s-%s.%s",
        r.manager.baseURL, r.version, runtime.GOOS, runtime.GOARCH, ext)
}

func (r *WhisperRuntime) EnsureInstalled(ctx context.Context, progress ProgressFunc) error {
    path := r.BinaryPath()

    // Check if already installed
    if _, err := os.Stat(path); err == nil {
        return nil
    }

    // Download and extract
    return r.manager.downloadAndExtract(ctx, r.downloadURL(), r.manager.binDir, progress)
}

func (r *WhisperRuntime) Execute(ctx context.Context, args ...string) ([]byte, error) {
    if err := r.EnsureInstalled(ctx, nil); err != nil {
        return nil, fmt.Errorf("failed to install whisper: %w", err)
    }

    cmd := exec.CommandContext(ctx, r.BinaryPath(), args...)
    return cmd.CombinedOutput()
}

// Transcribe is a convenience method for transcription
func (r *WhisperRuntime) Transcribe(ctx context.Context, audioPath, modelPath string, opts TranscribeOpts) ([]byte, error) {
    args := []string{
        "-m", modelPath,
        "-f", audioPath,
        "-l", opts.Language,
    }

    switch opts.OutputFormat {
    case "srt":
        args = append(args, "-osrt")
    case "vtt":
        args = append(args, "-ovtt")
    case "json":
        args = append(args, "-oj")
    default:
        args = append(args, "-otxt")
    }

    if opts.OutputFile != "" {
        args = append(args, "-of", opts.OutputFile)
    }

    return r.Execute(ctx, args...)
}
```

---

## Model Manager Implementation

```go
package models

import (
    "context"
    "os"
    "path/filepath"
)

// Model represents an AI model
type Model struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Runtime     string `json:"runtime"`
    Size        int64  `json:"size"`
    Checksum    string `json:"checksum"`
    Description string `json:"description"`
}

// Registry of available models
var Registry = []Model{
    // Whisper models
    {
        ID:          "whisper-large-v3-turbo",
        Name:        "Whisper Large V3 Turbo",
        Runtime:     "whisper",
        Size:        1_600_000_000, // ~1.6GB
        Description: "Best accuracy, 99 languages",
    },
    {
        ID:          "whisper-medium",
        Name:        "Whisper Medium",
        Runtime:     "whisper",
        Size:        1_500_000_000, // ~1.5GB
        Description: "Balanced speed/accuracy",
    },
    {
        ID:          "whisper-small",
        Name:        "Whisper Small",
        Runtime:     "whisper",
        Size:        466_000_000, // ~466MB
        Description: "Fast, good for quick transcription",
    },
    // Parakeet models
    {
        ID:          "parakeet-v3-int8",
        Name:        "Parakeet V3 INT8",
        Runtime:     "sherpa",
        Size:        640_000_000, // ~640MB
        Description: "Fast, 25 European languages",
    },
    // Piper voices
    {
        ID:          "piper-en-us-lessac",
        Name:        "Piper English (Lessac)",
        Runtime:     "piper",
        Size:        60_000_000, // ~60MB
        Description: "English US voice",
    },
    // Tesseract data
    {
        ID:          "tesseract-eng",
        Name:        "Tesseract English",
        Runtime:     "tesseract",
        Size:        4_000_000, // ~4MB
        Description: "English OCR data",
    },
}

// Download sources
var Sources = map[string]string{
    "huggingface": "https://huggingface.co/vget/models/resolve/main",
    "vmirror":     "https://vmirror.org/models",
}

// Manager handles model downloads
type Manager struct {
    source    string // "huggingface" or "vmirror"
    modelsDir string // ~/.config/vget/models
}

func NewManager() (*Manager, error) {
    configDir, err := os.UserConfigDir()
    if err != nil {
        return nil, err
    }

    modelsDir := filepath.Join(configDir, "vget", "models")
    if err := os.MkdirAll(modelsDir, 0755); err != nil {
        return nil, err
    }

    return &Manager{
        baseURL:   "https://dl.vget.dev/models",
        modelsDir: modelsDir,
    }, nil
}

func (m *Manager) GetModel(id string) (*Model, error) {
    for _, model := range Registry {
        if model.ID == id {
            return &model, nil
        }
    }
    return nil, fmt.Errorf("unknown model: %s", id)
}

func (m *Manager) ModelPath(id string) string {
    return filepath.Join(m.modelsDir, id+".bin")
}

func (m *Manager) IsInstalled(id string) bool {
    _, err := os.Stat(m.ModelPath(id))
    return err == nil
}

func (m *Manager) Download(ctx context.Context, id string, source string, progress ProgressFunc) error {
    model, err := m.GetModel(id)
    if err != nil {
        return err
    }

    // Default to huggingface
    if source == "" {
        source = "huggingface"
    }

    baseURL, ok := Sources[source]
    if !ok {
        return fmt.Errorf("unknown source: %s (available: huggingface, vmirror)", source)
    }

    url := fmt.Sprintf("%s/%s/%s.bin", baseURL, model.Runtime, model.ID)
    return m.downloadFile(ctx, url, m.ModelPath(id), progress)
}

func (m *Manager) ListInstalled() []Model {
    var installed []Model
    for _, model := range Registry {
        if m.IsInstalled(model.ID) {
            installed = append(installed, model)
        }
    }
    return installed
}

// ListRemote fetches available models from R2
func (m *Manager) ListRemote(ctx context.Context) ([]Model, error) {
    url := fmt.Sprintf("%s/models.json", m.baseURL)
    resp, err := m.httpClient.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch remote models: %w", err)
    }
    defer resp.Body.Close()

    var models []Model
    if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
        return nil, err
    }
    return models, nil
}
```

---

## Audio Processing

### Pure Go Decoders

```go
package audio

import (
    "io"
    "os"

    mp3 "github.com/hajimehoshi/go-mp3"
    "github.com/go-audio/wav"
    "github.com/mewkiz/flac"
)

// DecodeToWAV converts audio file to 16kHz mono WAV for transcription
func DecodeToWAV(inputPath, outputPath string) error {
    ext := filepath.Ext(inputPath)

    switch ext {
    case ".mp3":
        return decodeMP3(inputPath, outputPath)
    case ".wav":
        return convertWAV(inputPath, outputPath)
    case ".flac":
        return decodeFLAC(inputPath, outputPath)
    default:
        // Use ffmpeg WASM for other formats
        return decodeWithFFmpeg(inputPath, outputPath)
    }
}

func decodeMP3(input, output string) error {
    f, err := os.Open(input)
    if err != nil {
        return err
    }
    defer f.Close()

    d, err := mp3.NewDecoder(f)
    if err != nil {
        return err
    }

    // Resample to 16kHz mono and write WAV
    return writeWAV(output, d, d.SampleRate(), 16000)
}
```

### FFmpeg WASM Fallback

For formats not supported by pure Go decoders (M4A, AAC, OGG, etc.):

```go
package audio

import (
    "github.com/nicholassm/go-ffmpreg"
)

func decodeWithFFmpeg(input, output string) error {
    // go-ffmpreg embeds ffmpeg as WASM (~8MB)
    return ffmpreg.Convert(input, output, ffmpreg.Options{
        SampleRate: 16000,
        Channels:   1,
        Format:     "wav",
    })
}
```

---

## TUI Progress

Show progress during downloads and processing:

```go
package tui

import (
    "github.com/charmbracelet/bubbles/progress"
    tea "github.com/charmbracelet/bubbletea"
)

type DownloadModel struct {
    name     string
    progress progress.Model
    percent  float64
}

func (m DownloadModel) View() string {
    return fmt.Sprintf(
        "Downloading %s...\n%s %.0f%%\n",
        m.name,
        m.progress.ViewAs(m.percent),
        m.percent*100,
    )
}
```

---

## Binary Size Estimates

| Component | Size |
|-----------|------|
| vget core | ~20MB |
| go-ffmpreg (WASM) | ~8MB |
| Pure Go decoders | ~1MB |
| **Total binary** | **~30MB** |

Runtime binaries (downloaded separately):
| Runtime | Size |
|---------|------|
| whisper.cpp | ~5-8MB |
| sherpa-onnx | ~10-15MB |
| piper | ~8MB |
| tesseract | ~5MB |

---

## Implementation Phases

### Phase 1: Infrastructure
- [ ] Runtime manager with download/verify/execute
- [ ] Model manager with download/verify
- [ ] Cloudflare R2 bucket setup
- [ ] Build scripts for runtimes

### Phase 2: Speech-to-Text
- [ ] whisper.cpp runtime integration
- [ ] sherpa-onnx runtime integration
- [ ] Audio decoders (pure Go + ffmpeg WASM)
- [ ] `vget ai transcribe` command
- [ ] `vget ai models` command

### Phase 3: Text-to-Speech
- [ ] Piper runtime integration
- [ ] `vget ai speak` command
- [ ] `vget ai voices` command

### Phase 4: OCR
- [ ] Tesseract runtime integration
- [ ] `vget ai ocr` command
- [ ] PDF OCR support

### Phase 5: Polish
- [ ] Progress TUI with Bubbletea
- [ ] Error handling and retries
- [ ] Documentation

---

## Testing

### Local Testing

```bash
# Build and test transcription
go build -o build/vget ./cmd/vget
./build/vget ai transcribe testdata/sample.mp3

# Test model download
./build/vget ai download whisper-small
./build/vget ai models --installed

# Test TTS
./build/vget ai speak "Hello world" -o hello.wav

# Test OCR
./build/vget ai ocr testdata/screenshot.png
```

### Integration Tests

```go
func TestTranscribe(t *testing.T) {
    // Download model if not present
    modelMgr, _ := models.NewManager()
    if !modelMgr.IsInstalled("whisper-small") {
        modelMgr.Download(context.Background(), "whisper-small", nil)
    }

    // Transcribe test file
    result, err := transcriber.Transcribe("testdata/sample.wav", "whisper-small")
    require.NoError(t, err)
    assert.Contains(t, result.Text, "expected text")
}
```

---

## References

- [whisper.cpp](https://github.com/ggerganov/whisper.cpp) - C++ Whisper implementation
- [sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx) - ONNX speech recognition
- [Piper](https://github.com/rhasspy/piper) - Neural TTS
- [Tesseract](https://github.com/tesseract-ocr/tesseract) - OCR engine
- [go-ffmpreg](https://codeberg.org/gruf/go-ffmpreg) - Embedded ffmpeg WASM
- [go-mp3](https://github.com/hajimehoshi/go-mp3) - Pure Go MP3 decoder
