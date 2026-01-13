# vget Docker Image
# Always builds with CUDA support for GPU acceleration
# Runtime: entrypoint detects GPU and selects appropriate binary

# Build stage for UI
FROM node:22-slim AS ui-builder
WORKDIR /app/ui
COPY ui/package*.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

# Go builder stage - use pre-built CUDA base image with whisper.cpp
FROM ghcr.io/guiyumin/vget-base:latest AS go-builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui-builder /app/ui/dist ./internal/server/dist

# Build environment for whisper.cpp bindings
ENV LD_LIBRARY_PATH=/usr/local/lib:/usr/local/cuda/lib64:/usr/local/cuda/lib64/stubs
ENV C_INCLUDE_PATH=/usr/local/include
ENV LIBRARY_PATH=/usr/local/lib:/usr/local/cuda/lib64:/usr/local/cuda/lib64/stubs

# Setup CUDA stubs for linking
RUN ln -sf /usr/local/cuda/lib64/stubs/libcuda.so /usr/local/lib/libcuda.so && \
    ln -sf /usr/local/cuda/lib64/stubs/libcuda.so /usr/local/lib/libcuda.so.1 && \
    ln -sf /usr/local/cuda/lib64/stubs/libcuda.so /usr/local/cuda/lib64/stubs/libcuda.so.1 && \
    ldconfig

# Build TWO binaries:
# - vget-server: no AI, pure Go, no CUDA dependencies (used when no GPU detected)
# - vget-server-cuda: full AI with CUDA support (used when GPU is available)
RUN echo "Building vget-server (no AI, pure Go)..." && \
    CGO_ENABLED=0 go build -tags noai -ldflags="-s -w" -o /vget-server ./cmd/vget-server && \
    echo "Building vget-server-cuda (full AI with CUDA)..." && \
    CGO_ENABLED=1 go build -ldflags="-s -w" -o /vget-server-cuda ./cmd/vget-server

# Runtime stage - CUDA runtime for GPU support
FROM nvidia/cuda:12.6.3-runtime-ubuntu24.04 AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    chromium \
    fonts-noto-cjk \
    fonts-noto-color-emoji \
    python3 \
    python3-pip \
    python3-venv \
    nodejs \
    gosu \
    curl \
    bzip2 \
    xz-utils \
    && rm -rf /var/lib/apt/lists/*

# Install ffmpeg 7.x static build (more stable than Ubuntu's 6.1.1)
RUN curl -L https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz | \
    tar -xJ --strip-components=1 -C /usr/local && \
    ln -sf /usr/local/bin/ffmpeg /usr/bin/ffmpeg && \
    ln -sf /usr/local/bin/ffprobe /usr/bin/ffprobe

RUN pip3 install --no-cache-dir --break-system-packages \
    yt-dlp \
    youtube-dl

RUN (getent group 1000 >/dev/null || groupadd -g 1000 vget) && \
    (id -u 1000 >/dev/null 2>&1 || useradd -u 1000 -g 1000 -m -d /home/vget vget) && \
    mkdir -p /home/vget/downloads /home/vget/.config/vget /home/vget/models && \
    chown -R 1000:1000 /home/vget

# Copy AI libraries (whisper.cpp with CUDA)
COPY --from=go-builder /usr/local/lib/libwhisper*.so* /usr/local/lib/
COPY --from=go-builder /usr/local/lib/libggml*.so* /usr/local/lib/
RUN ldconfig

# Copy both binaries
COPY --from=go-builder /vget-server /usr/local/bin/vget-server
COPY --from=go-builder /vget-server-cuda /usr/local/bin/vget-server-cuda

ENV ROD_BROWSER=/usr/bin/chromium
ENV LD_LIBRARY_PATH=/usr/local/lib
ENV NVIDIA_VISIBLE_DEVICES=all
ENV NVIDIA_DRIVER_CAPABILITIES=compute,utility

COPY docker/vget/entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /home/vget
EXPOSE 8080
VOLUME ["/home/vget/downloads", "/home/vget/.config/vget"]

ENTRYPOINT ["entrypoint.sh"]
CMD []
