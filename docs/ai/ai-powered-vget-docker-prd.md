# PRD: AI-Powered vget Docker

## Overview

Docker deployment for vget with AI capabilities, featuring a web UI for media processing.

See [ai-powered-vget-prd.md](./ai-powered-vget-prd.md) for shared concepts.

---

## User Flow (Web UI)

1. User opens web UI at `http://localhost:8080`
2. Selects audio/video file from downloads or uploads new file
3. Configures AI options (transcribe, translate, summarize)
4. Selects processing options (engine, model, language)
5. Monitors progress via real-time stepper UI
6. Downloads or views outputs (transcript, translation, SRT, summary)

---

## Multi-Image Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  vget-base (build base)                                         │
│  ├── :latest  - CPU (golang:1.23-bookworm)                      │
│  └── :cuda    - CUDA 12.6 (nvidia/cuda:12.6.3-devel)            │
│                                                                 │
│  Contains: Go 1.23, sherpa-onnx libs, whisper.cpp libs          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  vget (application)                                             │
│                                                                 │
│  CPU variants:                                                  │
│  ├── :latest       - No models (~500MB)                         │
│  ├── :small        - Parakeet + Whisper Small (~1.2GB)          │
│  ├── :medium       - Parakeet + Whisper Medium (~2.0GB)         │
│  └── :large        - Parakeet + Whisper Large Turbo (~2.3GB)    │
│                                                                 │
│  CUDA variants:                                                 │
│  ├── :cuda         - No models + CUDA runtime                   │
│  ├── :cuda-small   - Parakeet + Whisper Small + CUDA            │
│  ├── :cuda-medium  - Parakeet + Whisper Medium + CUDA           │
│  └── :cuda-large   - Parakeet + Whisper Large Turbo + CUDA      │
└─────────────────────────────────────────────────────────────────┘
```

### Image Variants

| Tag | Models | Size | Best For |
|-----|--------|------|----------|
| `:latest` | None | ~500MB | Download models on first use |
| `:small` | Parakeet V3 + Whisper Small | ~1.2GB | NAS <8GB RAM |
| `:medium` | Parakeet V3 + Whisper Medium | ~2.0GB | 8-16GB RAM |
| `:large` | Parakeet V3 + Whisper Large Turbo | ~2.3GB | Best accuracy |
| `:cuda-*` | Same as above + CUDA | +2GB | NVIDIA GPU |

---

## Docker Usage

### Basic Usage (CPU)

```yaml
# compose.yml
services:
  vget:
    image: ghcr.io/guiyumin/vget:medium
    ports:
      - "8080:8080"
    volumes:
      - ./config:/home/vget/.config/vget
      - ./downloads:/home/vget/downloads
```

### GPU Usage (NVIDIA CUDA)

```yaml
services:
  vget:
    image: ghcr.io/guiyumin/vget:cuda-large
    ports:
      - "8080:8080"
    volumes:
      - ./config:/home/vget/.config/vget
      - ./downloads:/home/vget/downloads
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VGET_PORT` | 8080 | Web UI port |
| `VGET_HOST` | 0.0.0.0 | Listen address |
| `VGET_DATA_DIR` | /home/vget/downloads | Download directory |
| `VGET_MODEL_DIR` | /home/vget/.config/vget/models | Model storage |

---

## API Endpoints

### AI Job Management

```
POST   /api/ai/jobs              # Start new AI job
GET    /api/ai/jobs              # List all jobs
GET    /api/ai/jobs/:id          # Get job status and progress
DELETE /api/ai/jobs/:id          # Cancel job
GET    /api/ai/jobs/:id/result   # Get job outputs
```

### Request/Response

```typescript
// POST /api/ai/jobs
interface StartJobRequest {
  file_path: string;
  options: {
    transcribe: boolean;
    summarize: boolean;
    translate_to?: string[];  // ["en", "zh"]
    model?: string;  // e.g., "whisper-large-v3-turbo", "parakeet-v3"
    generate_srt?: boolean;   // also generate SRT after transcription
    generate_vtt?: boolean;   // also generate VTT after transcription
    language?: string;
  };
}

// GET /api/ai/jobs/:id
interface JobStatus {
  id: string;
  file_path: string;
  file_name: string;
  status: "queued" | "processing" | "completed" | "failed" | "cancelled";
  current_step: StepKey;
  steps: ProcessingStep[];
  overall_progress: number;
  result?: JobResult;
  error?: string;
  created_at: string;
  updated_at: string;
}

interface ProcessingStep {
  key: StepKey;
  name: string;
  status: "pending" | "in_progress" | "completed" | "skipped" | "failed";
  progress: number;
  detail?: string;
}

type StepKey =
  | "extract_audio"
  | "compress_audio"
  | "chunk_audio"
  | "transcribe"
  | "merge"
  | "translate"
  | "generate_srt"
  | "summarize";

interface JobResult {
  transcript_path?: string;
  translated_paths?: Record<string, string>;  // { "en": "...", "zh": "..." }
  srt_paths?: Record<string, string>;
  summary_path?: string;
}
```

### Model Management API

```
GET    /api/ai/models             # List available models
GET    /api/ai/models/installed   # List installed models
POST   /api/ai/models/:id/install # Download and install model
DELETE /api/ai/models/:id         # Remove installed model
```

---

## Web UI Components

### Processing Configuration

```typescript
interface ProcessingConfig {
  transcribe: boolean;
  summarize: boolean;
  translateTo: string[];      // ["en", "zh", "jp"]
  model: string;              // "auto", "whisper-large-v3-turbo", "parakeet-v3", etc.
  language: string;
  generateSrt: boolean;       // also generate SRT after transcription
  generateVtt: boolean;       // also generate VTT after transcription
}
```

### Step Display (ProcessingStepper)

Real-time progress visualization with step-by-step status.

### UI Translations

```typescript
// AI step names
ai_step_extract_audio: "Extract Audio",
ai_step_compress_audio: "Compress Audio",
ai_step_chunk_audio: "Chunk Audio",
ai_step_transcribe: "Transcribe",
ai_step_merge: "Merge Chunks",
ai_step_translate: "Translate",
ai_step_generate_srt: "Generate Subtitles",
ai_step_summarize: "Generate Summary",

// Options
ai_model_auto: "Auto (recommended)",
ai_model_whisper: "Whisper Large V3 Turbo (99 languages)",
ai_model_parakeet: "Parakeet V3 (fast, European)",
ai_translate_to: "Translate to",
ai_generate_srt: "Also generate SRT subtitles",
ai_generate_vtt: "Also generate VTT subtitles",
```

---

## Processing Pipeline

```
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│   Upload    │ → │   Chunk     │ → │ Transcribe  │ → │  Translate  │ → │  Summarize  │
│  (Web UI)   │   │  (ffmpeg)   │   │  (Whisper)  │   │   (LLM)     │   │   (LLM)     │
└─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
                         │                 │                 │                 │
                         ▼                 ▼                 ▼                 ▼
                  .chunks/ dir      .transcript.md    .{lang}.transcript.md  .summary.md
                                                      .{lang}.srt
```

### Chunking Strategy

- Fixed 10-minute chunks with 10-second overlap
- Deduplication during merge phase
- Handles files up to several hours

---

## Output Files

```
podcast.mp3
  → podcast.transcript.md       (original language transcript)
  → podcast.en.transcript.md    (translated to English)
  → podcast.en.srt              (English subtitles)
  → podcast.summary.md          (summary in original language)
  → podcast.en.summary.md       (summary in English)
```

### Transcript Format (podcast.transcript.md)

```markdown
# Transcript: podcast.mp3

**Duration:** 1h 23m 45s
**Transcribed:** 2024-01-15 10:30:00
**Engine:** whisper/large-v3-turbo

---

[00:00:00] Welcome to today's episode. We're going to discuss...

[00:00:15] The main topic is about building reliable systems...

[00:05:30] Let me give you an example of how this works in practice...
```

### SRT Format (podcast.en.srt)

```srt
1
00:00:00,000 --> 00:00:15,000
Welcome to today's episode. We're going to discuss...

2
00:00:15,000 --> 00:05:30,000
The main topic is about building reliable systems...

3
00:05:30,000 --> 00:10:45,000
Let me give you an example of how this works in practice...
```

---

## Dockerfile

### Base Image

```dockerfile
# Dockerfile.base
FROM golang:1.23-bookworm AS base

# Install build dependencies
RUN apt-get update && apt-get install -y \
    cmake \
    libopenblas-dev \
    && rm -rf /var/lib/apt/lists/*

# Build sherpa-onnx
RUN git clone --depth 1 https://github.com/k2-fsa/sherpa-onnx && \
    cd sherpa-onnx && \
    cmake -B build -DCMAKE_BUILD_TYPE=Release && \
    cmake --build build && \
    cmake --install build

# Build whisper.cpp
RUN git clone --depth 1 https://github.com/ggerganov/whisper.cpp && \
    cd whisper.cpp && \
    cmake -B build -DCMAKE_BUILD_TYPE=Release && \
    cmake --build build && \
    cmake --install build
```

### Application Image

```dockerfile
# Dockerfile
ARG BASE_IMAGE=ghcr.io/guiyumin/vget-base:latest
ARG MODEL_VARIANT=none

FROM ${BASE_IMAGE} AS builder

WORKDIR /app
COPY . .

RUN go build -o vget ./cmd/vget

# Final image
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    ffmpeg \
    libopenblas0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/vget /usr/local/bin/vget
COPY --from=builder /usr/local/lib/lib*.so* /usr/local/lib/

# Download models based on variant
ARG MODEL_VARIANT
RUN if [ "$MODEL_VARIANT" != "none" ]; then \
      vget ai download parakeet-v3-int8; \
      case "$MODEL_VARIANT" in \
        small) vget ai download whisper-small ;; \
        medium) vget ai download whisper-medium ;; \
        large) vget ai download whisper-large-v3-turbo ;; \
      esac; \
    fi

# Create non-root user
RUN useradd -m -u 1000 vget
USER vget
WORKDIR /home/vget

EXPOSE 8080
CMD ["vget", "server"]
```

### Build Args

| Arg | Values | Description |
|-----|--------|-------------|
| `BASE_IMAGE` | vget-base:latest, vget-base:cuda | Base image |
| `MODEL_VARIANT` | none, small, medium, large | Bundle models |

---

## Error Handling

| Error | Behavior |
|-------|----------|
| No models installed | Show install prompt in UI |
| Transcription fails | Mark step failed, show error in UI |
| Chunk fails | Skip chunk, warn user, continue |
| Translation API error | Retry with backoff, fallback to partial |
| Disk space low | Check before job, warn user |
| GPU not available | Fallback to CPU, warn user |

---

## Implementation Phases

### Phase 1: Core Docker Setup
- [ ] Base image with runtime libraries
- [ ] Application image with model variants
- [ ] GitHub Actions for image builds

### Phase 2: Web UI
- [ ] File browser integration
- [ ] Processing configuration UI
- [ ] Real-time progress stepper
- [ ] Result viewer

### Phase 3: API
- [ ] Job management endpoints
- [ ] Model management endpoints
- [ ] WebSocket for progress updates

### Phase 4: GPU Support
- [ ] CUDA base image
- [ ] GPU detection and fallback
- [ ] Performance optimization

---

## Success Criteria

1. `docker compose up` starts working web UI
2. File selection and processing works
3. Progress is visible in real-time
4. Transcripts are accurate with timestamps
5. GPU acceleration works when available
6. Errors are clear and actionable
7. Image sizes are reasonable (<2.5GB with models)
