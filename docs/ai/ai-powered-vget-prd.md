# PRD: AI-Powered vget

## Overview

vget integrates local-first AI capabilities for media processing. The design prioritizes:

- **Zero runtime dependencies** - Single binary, no system libraries required
- **Download on first use** - Runtime binaries and models fetched from Cloudflare R2
- **Offline capable** - Once downloaded, works without internet
- **Cross-platform** - macOS (arm64/amd64), Linux (arm64/amd64), Windows (amd64)

## AI Features

| Feature | Runtime | Use Case | Status |
|---------|---------|----------|--------|
| Speech-to-Text (STT) | whisper.cpp, sherpa-onnx | Transcription, subtitles | **Active** |
| Text-to-Speech (TTS) | Piper | Audiobook generation, accessibility | TODO |
| OCR | Tesseract | Image text extraction, scanned PDFs | TODO |
| PDF Processing | pdfcpu, poppler | Text extraction, manipulation | TODO |

---

## Architecture

### Runtime Binary Management

```
┌─────────────────────────────────────────────────────────────────┐
│                        vget CLI Binary                          │
│                       (CGO_ENABLED=0)                           │
├─────────────────────────────────────────────────────────────────┤
│  Runtime Manager                                                │
│  ├── Check ~/.config/vget/bin/ for required binary              │
│  ├── Download from Cloudflare R2 if missing                     │
│  ├── Verify checksum                                            │
│  └── Execute via exec.Command()                                 │
├─────────────────────────────────────────────────────────────────┤
│  Model Manager                                                  │
│  ├── Check ~/.config/vget/models/ for required model            │
│  ├── Download from Cloudflare R2 if missing                     │
│  ├── Verify checksum                                            │
│  └── Pass to runtime binary                                     │
└─────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
~/.config/vget/
├── config.yml
├── bin/                              # Runtime binaries (auto-downloaded)
│   ├── whisper-darwin-arm64          # whisper.cpp for macOS ARM
│   ├── whisper-linux-amd64           # whisper.cpp for Linux x64
│   ├── sherpa-onnx-darwin-arm64      # sherpa-onnx for macOS ARM
│   ├── piper-darwin-arm64            # Piper TTS
│   ├── tesseract-darwin-arm64        # Tesseract OCR
│   └── ...
└── models/                           # AI models (auto-downloaded)
    ├── whisper-large-v3-turbo.bin    # Whisper model (~1.6GB)
    ├── parakeet-v3-int8.onnx         # Parakeet model (~640MB)
    ├── piper-en-us.onnx              # Piper voice model
    ├── tesseract-eng.traineddata     # Tesseract language data
    └── ...
```

### Download Infrastructure

All binaries and models are hosted on Cloudflare R2:

```
https://dl.vget.dev/
├── bin/
│   ├── whisper/
│   │   ├── v1.8.2/
│   │   │   ├── whisper-darwin-arm64.tar.gz
│   │   │   ├── whisper-darwin-amd64.tar.gz
│   │   │   ├── whisper-linux-amd64.tar.gz
│   │   │   ├── whisper-linux-arm64.tar.gz
│   │   │   ├── whisper-windows-amd64.zip
│   │   │   └── checksums.txt
│   │   └── latest -> v1.8.2
│   ├── sherpa-onnx/
│   ├── piper/
│   └── tesseract/
└── models/
    ├── whisper/
    │   ├── large-v3-turbo.bin
    │   ├── medium.bin
    │   └── small.bin
    ├── parakeet/
    ├── piper/
    └── tesseract/
```

---

## Feature Details

### 1. Speech-to-Text (STT)

**Purpose:** Convert audio/video to text transcripts and subtitles.

#### Dual Engine Architecture

| Engine | Library | Languages | Best For |
|--------|---------|-----------|----------|
| Parakeet V3 | sherpa-onnx | 25 European | Fast, default for non-CJK |
| Whisper | whisper.cpp | 99 | CJK languages, accuracy |

#### Auto Model Selection

```
if language in [zh, ja, ko] or detected_language in [zh, ja, ko]:
    use Whisper model (whisper.cpp runtime)
else:
    use Parakeet model (sherpa-onnx runtime)
```

User can override via `--model` flag. The runtime is determined by the model.

#### Model Options

| Model | Engine | Size | Languages | Use Case |
|-------|--------|------|-----------|----------|
| Parakeet V3 INT8 | sherpa-onnx | ~640MB | 25 EU | Default, fastest |
| Whisper Small | whisper.cpp | ~466MB | 99 | Quick CJK |
| Whisper Medium | whisper.cpp | ~1.5GB | 99 | Balanced |
| Whisper Large V3 Turbo | whisper.cpp | ~1.6GB | 99 | Best accuracy |

#### Output Formats

- **Transcript** (`.transcript.md`) - Timestamped text in Markdown
- **Subtitles** (`.srt`, `.vtt`) - Standard subtitle formats
- **Raw text** (`.txt`) - Plain text without timestamps

### 2. Text-to-Speech (TTS)

**Purpose:** Generate natural speech from text.

#### Runtime: Piper

[Piper](https://github.com/rhasspy/piper) is a fast, local neural TTS system.

| Voice Model | Language | Size | Quality |
|-------------|----------|------|---------|
| en_US-lessac-medium | English | ~60MB | Good |
| en_US-libritts-high | English | ~100MB | High |
| zh_CN-huayan-medium | Chinese | ~60MB | Good |

#### Use Cases

- Generate audiobooks from text/EPUB
- Accessibility (read articles aloud)
- Podcast generation from transcripts

### 3. OCR (Optical Character Recognition)

**Purpose:** Extract text from images and scanned documents.

#### Runtime: Tesseract

[Tesseract](https://github.com/tesseract-ocr/tesseract) with pre-trained language models.

| Language | Data File | Size |
|----------|-----------|------|
| English | eng.traineddata | ~4MB |
| Chinese (Simplified) | chi_sim.traineddata | ~50MB |
| Chinese (Traditional) | chi_tra.traineddata | ~50MB |
| Japanese | jpn.traineddata | ~15MB |

#### Use Cases

- Extract text from screenshots
- Process scanned PDFs
- Digitize printed documents

### 4. PDF Processing

**Purpose:** Extract, manipulate, and convert PDF documents.

#### Capabilities

- Text extraction (native and OCR fallback)
- Page manipulation (split, merge, rotate)
- Convert to images
- Metadata extraction

---

## Core Interfaces

### Runtime Interface

```go
// Runtime represents an external AI runtime binary
type Runtime interface {
    Name() string
    Version() string
    BinaryName() string
    DownloadURL() string
    Checksum() string

    // EnsureInstalled downloads the binary if not present
    EnsureInstalled(ctx context.Context) error

    // Execute runs the binary with given arguments
    Execute(ctx context.Context, args ...string) ([]byte, error)
}
```

### Model Interface

```go
// Model represents an AI model file
type Model interface {
    Name() string
    Runtime() string  // which runtime uses this model
    Size() int64
    DownloadURL() string
    Checksum() string

    // EnsureDownloaded downloads the model if not present
    EnsureDownloaded(ctx context.Context) error

    // Path returns the local file path
    Path() string
}
```

### Transcriber Interface

```go
type Transcriber interface {
    Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscribeResult, error)
}

type TranscribeOptions struct {
    Language    string   // ISO 639-1 code or "auto"
    Engine      string   // "whisper", "parakeet", or "auto"
    Model       string   // Model name
    OutputFormat string  // "transcript", "srt", "vtt", "txt"
}

type TranscribeResult struct {
    Text      string
    Segments  []Segment
    Language  string
    Duration  time.Duration
}

type Segment struct {
    Start time.Duration
    End   time.Duration
    Text  string
}
```

### Synthesizer Interface (TTS)

```go
type Synthesizer interface {
    Synthesize(ctx context.Context, text string, opts SynthesizeOptions) (*SynthesizeResult, error)
}

type SynthesizeOptions struct {
    Voice       string  // Voice model name
    Speed       float64 // Speech rate (0.5 - 2.0)
    OutputFormat string // "wav", "mp3"
}

type SynthesizeResult struct {
    AudioPath string
    Duration  time.Duration
}
```

### OCR Interface

```go
type OCREngine interface {
    ExtractText(ctx context.Context, imagePath string, opts OCROptions) (*OCRResult, error)
}

type OCROptions struct {
    Languages []string // ISO 639-3 codes
    DPI       int      // Image DPI hint
}

type OCRResult struct {
    Text       string
    Confidence float64
    Blocks     []TextBlock
}
```

---

## Processing Pipeline

```
┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐
│  Input  │ → │ Prepare │ → │ Process │ → │ Convert │ → │ Output  │
└─────────┘   └─────────┘   └─────────┘   └─────────┘   └─────────┘
     │             │             │             │             │
     ▼             ▼             ▼             ▼             ▼
  audio.mp3   extract WAV   transcribe    translate    .transcript.md
  video.mp4   chunk audio   OCR image     summarize    .srt
  image.png   convert fmt   synthesize    format       .wav
  doc.pdf     split pages   extract       merge        .txt
```

### Step Types

| Step | Description | Input | Output |
|------|-------------|-------|--------|
| `extract_audio` | Extract audio from video | video file | WAV/MP3 |
| `chunk_audio` | Split large audio | audio file | chunk files |
| `transcribe` | Speech to text | audio file | segments |
| `merge_chunks` | Combine transcripts | segments | full transcript |
| `translate` | Translate text | text | translated text |
| `generate_srt` | Create subtitles | segments | SRT file |
| `summarize` | Summarize content | text | summary |
| `synthesize` | Text to speech | text | audio file |
| `ocr` | Extract text from image | image | text |

---

## Error Handling

| Error | Behavior |
|-------|----------|
| Binary not found | Auto-download from R2, retry |
| Model not found | Auto-download from R2, retry |
| Download failed | Retry 3x with backoff, then error |
| Checksum mismatch | Delete and re-download |
| Unsupported platform | Clear error message with supported platforms |
| Insufficient disk space | Check before download, warn user |

---

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] Runtime manager (download, verify, execute)
- [ ] Model manager (download, verify, path)
- [ ] Cloudflare R2 hosting setup

### Phase 2: Speech-to-Text
- [ ] whisper.cpp integration
- [ ] sherpa-onnx integration
- [ ] Transcript output formats
- [ ] SRT/VTT generation

### Phase 3: Text-to-Speech
- [ ] Piper integration
- [ ] Voice model management
- [ ] Audio output formats

### Phase 4: OCR
- [ ] Tesseract integration
- [ ] Language data management
- [ ] PDF OCR fallback

### Phase 5: Advanced Features
- [ ] Translation (LLM-based)
- [ ] Summarization (LLM-based)
- [ ] Batch processing

---

## Success Criteria

1. `vget ai transcribe` works on first run (auto-downloads runtime + model)
2. All features work offline after initial download
3. Cross-platform support (macOS, Linux, Windows)
4. Reasonable download sizes (runtime < 20MB, models vary)
5. Clear progress indication during downloads
6. Graceful error handling with actionable messages
