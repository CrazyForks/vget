## Build Docker Image

### Build vget image (amd64 with CUDA)

```shell
docker build -f docker/vget/Dockerfile --build-arg ENABLE_CUDA=true -t ghcr.io/guiyumin/vget:latest .
```

### Build vget image (arm64, no CUDA)

```shell
docker build -f docker/vget/Dockerfile -t ghcr.io/guiyumin/vget:latest .
```

### Runtime Behavior

- **GPU detected** (via `nvidia-smi`): Local transcription mode, download models on demand
- **No GPU**: Cloud API mode (OpenAI Whisper API, Groq, etc.)

Models are not bundled in the image. They are downloaded on first use from HuggingFace or vmirror (China).
