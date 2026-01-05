# Local-First Speech-to-Text

## Overview

vget uses a hybrid approach for local speech-to-text transcription:

- **Parakeet V3** via sherpa-onnx: Fast, CPU-optimized, 25 European languages
- **Whisper** via whisper.cpp: Highly optimized, 99 languages including Chinese

**Why two engines instead of one?**
- whisper.cpp is the most optimized Whisper implementation (Metal, CUDA, AVX2/AVX512)
- sherpa-onnx is required for Parakeet (only available option with Go bindings)
- Each engine excels at what it does best

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    vget transcriber                      │
│                                                         │
│  internal/core/ai/transcriber/                          │
│    ├── sherpa.go   → Parakeet V3 (sherpa-onnx)          │
│    └── whisper.go  → Whisper (whisper.cpp)              │
├─────────────────────────────────────────────────────────┤
│  CGO Bindings                                           │
│    ├── sherpa-onnx-go (Parakeet)                        │
│    └── go-whisper (whisper.cpp)                         │
├─────────────────────────────────────────────────────────┤
│  Native Libraries                                       │
│    ├── libsherpa-onnx-core.so + ONNX Runtime            │
│    └── libwhisper.so (with Metal/CUDA/AVX2)             │
└─────────────────────────────────────────────────────────┘
```

## Model Selection Logic

```
if language == "zh" || (language == "auto" && detected == "zh"):
    use Whisper
else:
    use Parakeet (default) or Whisper (user choice)
```

**Rationale:**
- Parakeet V3 is faster and more accurate for European languages (25 supported)
- Whisper is required for Chinese and other non-European languages
- Users can override the default via settings

## Docker Image

Single image for all users - runtime GPU detection determines behavior:

| Condition | Mode | Model Source |
|-----------|------|--------------|
| `--gpus all` + NVIDIA GPU | Local transcription | Download on demand from HuggingFace/vmirror |
| No GPU flag or no NVIDIA | Cloud API | OpenAI Whisper API, Groq, etc. |

Models are not bundled in the image (~300MB base). They are downloaded on first use.

## Model Details

### Parakeet V3 (INT8 Quantized)
- **Source:** nvidia/parakeet-tdt-0.6b-v3
- **Files:** encoder.int8.onnx (~622MB), decoder.int8.onnx (~12MB), joiner.int8.onnx (~6MB), tokens.txt
- **Languages:** 25 European languages with auto-detection
  - Bulgarian, Croatian, Czech, Danish, Dutch, English, Estonian, Finnish, French, German, Greek, Hungarian, Italian, Latvian, Lithuanian, Maltese, Polish, Portuguese, Romanian, Slovak, Slovenian, Spanish, Swedish, Russian, Ukrainian
- **Download:** https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8.tar.bz2

### Whisper Models (via whisper.cpp)

| Model | Size | Use Case |
|-------|------|----------|
| ggml-small.bin | ~466MB | Fast, good accuracy |
| ggml-medium.bin | ~1.5GB | Balanced |
| ggml-large-v3-turbo.bin | ~1.6GB | Best accuracy, Chinese |

**Download URLs:**
- Small: https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin
- Medium: https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin
- Large V3 Turbo: https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin

**whisper.cpp optimizations:**
- Apple Silicon: Metal acceleration (GPU)
- NVIDIA: CUDA acceleration
- x86: AVX2/AVX512 SIMD
- ARM: NEON SIMD

## Configuration

In `~/.config/vget/config.yml`:

```yaml
ai:
  local_asr:
    enabled: true
    engine: "parakeet"    # or "whisper"
    model: "parakeet-v3"  # or "whisper-small", "whisper-medium", "whisper-large-turbo"
    language: "auto"      # or specific language code (en, zh, de, fr, etc.)
    models_dir: ""        # empty = default ~/.config/vget/models/
```

## Go API Usage

```go
// For Parakeet V3 (sherpa-onnx)
import sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

config := sherpa.OfflineRecognizerConfig{}
config.FeatConfig.SampleRate = 16000
config.FeatConfig.FeatureDim = 80
config.ModelConfig.Transducer.Encoder = "encoder.int8.onnx"
config.ModelConfig.Transducer.Decoder = "decoder.int8.onnx"
config.ModelConfig.Transducer.Joiner = "joiner.int8.onnx"
config.ModelConfig.Tokens = "tokens.txt"
config.ModelConfig.NumThreads = 4

recognizer := sherpa.NewOfflineRecognizer(&config)
stream := sherpa.NewOfflineStream(recognizer)
stream.AcceptWaveform(16000, audioSamples)
recognizer.Decode(stream)
result := stream.GetResult()
// result.Text, result.Lang (auto-detected), result.Timestamps

// For Whisper (whisper.cpp via go-whisper)
import "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"

model, _ := whisper.New(modelPath)
defer model.Close()

ctx, _ := model.NewContext()
ctx.SetLanguage("zh")  // or "auto" for auto-detect

// Process audio samples
ctx.Process(audioSamples, nil, nil)

// Get segments with timestamps
for {
    segment, err := ctx.NextSegment()
    if err != nil { break }
    fmt.Printf("[%v -> %v] %s\n", segment.Start, segment.End, segment.Text)
}
```

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/core/ai/transcriber/sherpa.go` | Parakeet V3 transcriber (sherpa-onnx) |
| `internal/core/ai/transcriber/whisper.go` | Whisper transcriber (whisper.cpp) |
| `internal/core/ai/transcriber/models.go` | Model definitions and management |
| `internal/core/ai/transcriber/transcriber.go` | Interface and factory |
| `docker/vget/Dockerfile` | Single Docker image with sherpa-onnx + whisper.cpp |

## Docker Usage

```bash
# Pull and run
docker compose up -d

# With NVIDIA GPU for local transcription
docker run --gpus all -p 8080:8080 ghcr.io/guiyumin/vget:latest

# Without GPU - uses cloud API (OpenAI Whisper, Groq, etc.)
docker run -p 8080:8080 ghcr.io/guiyumin/vget:latest
```

Models are downloaded on first use from HuggingFace or vmirror (China).

## Comparison: Parakeet vs Whisper

| Feature | Parakeet V3 (sherpa-onnx) | Whisper (whisper.cpp) |
|---------|---------------------------|------------------------|
| Speed | Faster | Slower (but GPU accelerated) |
| Accuracy (EU) | Better | Good |
| Chinese | No | Yes |
| Japanese | No | Yes |
| Korean | No | Yes |
| Languages | 25 | 99 |
| Model Size | ~640MB | ~466MB to ~1.6GB |
| GPU Accel | ONNX Runtime | Metal, CUDA, Vulkan |
| CPU Optim | Good | Excellent (AVX2/AVX512/NEON) |

**Recommendation:** Use Parakeet V3 as default for European languages, Whisper for Chinese/Japanese/Korean and when GPU is available.

## Troubleshooting

### "libsherpa-onnx-core.so not found"

The sherpa-onnx library is not installed or not in the library path.

```bash
# Check if installed
ldconfig -p | grep sherpa

# If missing, download and install
curl -L https://github.com/k2-fsa/sherpa-onnx/releases/download/v1.12.20/sherpa-onnx-v1.12.20-linux-x64-shared.tar.bz2 | tar -xjf -
sudo cp sherpa-onnx-*/lib/*.so* /usr/local/lib/
sudo ldconfig
```

### "libwhisper.so not found"

The whisper.cpp library is not installed.

```bash
# Check if installed
ldconfig -p | grep whisper

# If missing, build from source
git clone --depth 1 https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
cmake -B build -DBUILD_SHARED_LIBS=ON
cmake --build build --config Release
sudo cp build/src/libwhisper.so* /usr/local/lib/
sudo ldconfig
```

### "ffmpeg not found"

Install ffmpeg for audio conversion:

```bash
# macOS
brew install ffmpeg

# Linux
sudo apt-get install ffmpeg
```

### Model download fails

Models are downloaded from GitHub/Hugging Face. Check your internet connection and try again.

```bash
# Manual download (Parakeet V3)
curl -L https://github.com/k2-fsa/sherpa-onnx/releases/download/asr-models/sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8.tar.bz2 | tar -xjf -
mv sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8 ~/.config/vget/models/

# Manual download (Whisper models - ggml format for whisper.cpp)
curl -L -o ~/.config/vget/models/ggml-small.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin
curl -L -o ~/.config/vget/models/ggml-medium.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin
curl -L -o ~/.config/vget/models/ggml-large-v3-turbo.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin
```

## References

### sherpa-onnx (Parakeet)
- [sherpa-onnx GitHub](https://github.com/k2-fsa/sherpa-onnx)
- [sherpa-onnx Go API](https://k2-fsa.github.io/sherpa/onnx/go-api/index.html)
- [Parakeet V3 Model](https://huggingface.co/nvidia/parakeet-tdt-0.6b-v3)
- [Pre-trained Models](https://k2-fsa.github.io/sherpa/onnx/pretrained_models/index.html)

### whisper.cpp (Whisper)
- [whisper.cpp GitHub](https://github.com/ggerganov/whisper.cpp)
- [whisper.cpp Go bindings](https://github.com/ggerganov/whisper.cpp/tree/master/bindings/go)
- [GGML Whisper Models](https://huggingface.co/ggerganov/whisper.cpp)
- [Whisper Model Card](https://huggingface.co/openai/whisper-large-v3-turbo)
