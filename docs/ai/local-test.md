## Build Docker Image

```shell
docker build -f docker/vget/Dockerfile -t ghcr.io/guiyumin/vget:latest .
```

### Runtime Behavior

The entrypoint automatically detects GPU availability:

- **GPU detected** (via `nvidia-smi`): Uses `vget-server-cuda` with local AI transcription
- **No GPU**: Uses `vget-server` with cloud API mode (OpenAI Whisper API, etc.)

Models are not bundled in the image. They are downloaded on first use from HuggingFace or vmirror (China).
