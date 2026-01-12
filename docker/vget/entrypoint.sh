#!/bin/bash
set -e

# Select the appropriate binary based on GPU availability
# vget-server-cuda: full AI with CUDA (requires GPU)
# vget-server: no local AI (works without GPU)
BINARY="vget-server"

if nvidia-smi &>/dev/null; then
    echo "✓ NVIDIA GPU detected - local AI transcription enabled"
    nvidia-smi --query-gpu=name,memory.total --format=csv,noheader 2>/dev/null | head -1
    # Use CUDA binary if available
    if [ -f /usr/local/bin/vget-server-cuda ]; then
        BINARY="vget-server-cuda"
        export LD_LIBRARY_PATH=/usr/local/lib
    fi
else
    echo "─────────────────────────────────────────────────────────"
    echo "  No GPU detected - local AI disabled, cloud API available"
    echo ""
    echo "  Have an NVIDIA GPU? Run with GPU access:"
    echo "    docker run --gpus all -p 8080:8080 ghcr.io/guiyumin/vget:latest"
    echo ""
    echo "  Or in compose.yml:"
    echo "    deploy:"
    echo "      resources:"
    echo "        reservations:"
    echo "          devices:"
    echo "            - driver: nvidia"
    echo "              count: all"
    echo "              capabilities: [gpu]"
    echo ""
    echo "  See: docs/ai/docker-gpu-passthrough.md"
    echo "─────────────────────────────────────────────────────────"
    # Use non-CUDA binary (no whisper dependencies)
    BINARY="vget-server"
fi

echo "Starting $BINARY..."

# Fix ownership of mounted volumes if running as root
if [ "$(id -u)" = "0" ]; then
    chown -R 1000:1000 /home/vget/downloads /home/vget/.config/vget
    exec gosu 1000:1000 "$BINARY" "$@"
else
    exec "$BINARY" "$@"
fi
