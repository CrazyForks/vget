# vget Base Image - Go 1.25 with sherpa-onnx + whisper.cpp
# Pre-built shared libraries for CGO bindings
# Published to ghcr.io/guiyumin/vget-base:latest

FROM golang:1.25-bookworm

LABEL org.opencontainers.image.source=https://github.com/guiyumin/vget
LABEL org.opencontainers.image.description="vget Base - Go 1.25 with sherpa-onnx + whisper.cpp for local speech-to-text"
LABEL org.opencontainers.image.licenses=MIT

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    cmake \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# sherpa-onnx version (for Parakeet)
ARG SHERPA_VERSION=1.12.20

# Download and install pre-built sherpa-onnx shared libraries
# Platform is determined at build time (linux/amd64 or linux/arm64)
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        SHERPA_ARCH="aarch64"; \
    else \
        SHERPA_ARCH="x64"; \
    fi && \
    curl -L "https://github.com/k2-fsa/sherpa-onnx/releases/download/v${SHERPA_VERSION}/sherpa-onnx-v${SHERPA_VERSION}-linux-${SHERPA_ARCH}-shared.tar.bz2" | tar -xjf - && \
    cp sherpa-onnx-v${SHERPA_VERSION}-linux-${SHERPA_ARCH}-shared/lib/*.so* /usr/local/lib/ && \
    cp -r sherpa-onnx-v${SHERPA_VERSION}-linux-${SHERPA_ARCH}-shared/include/* /usr/local/include/ && \
    rm -rf sherpa-onnx-v${SHERPA_VERSION}-linux-${SHERPA_ARCH}-shared && \
    ldconfig

# whisper.cpp version (for Whisper)
ARG WHISPER_VERSION=1.7.3

# Build whisper.cpp from source with shared library
RUN git clone --depth 1 --branch v${WHISPER_VERSION} https://github.com/ggerganov/whisper.cpp /tmp/whisper.cpp && \
    cd /tmp/whisper.cpp && \
    cmake -B build \
        -DCMAKE_BUILD_TYPE=Release \
        -DBUILD_SHARED_LIBS=ON \
        -DGGML_NATIVE=OFF \
        -DWHISPER_BUILD_EXAMPLES=OFF \
        -DWHISPER_BUILD_TESTS=OFF && \
    cmake --build build --config Release -j$(nproc) && \
    cp build/src/libwhisper.so* /usr/local/lib/ && \
    cp build/ggml/src/libggml*.so* /usr/local/lib/ && \
    cp include/whisper.h /usr/local/include/ && \
    cp ggml/include/ggml*.h /usr/local/include/ && \
    rm -rf /tmp/whisper.cpp && \
    ldconfig

# Set environment variables for CGO
ENV CGO_ENABLED=1
ENV LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

# C_INCLUDE_PATH and LIBRARY_PATH for whisper.cpp Go bindings
ENV C_INCLUDE_PATH=/usr/local/include
ENV LIBRARY_PATH=/usr/local/lib

# Reset workdir
WORKDIR /app

# Metadata
ENV BASE_IMAGE_VERSION=3.0.0
ENV SHERPA_ONNX_VERSION=${SHERPA_VERSION}
ENV WHISPER_CPP_VERSION=${WHISPER_VERSION}
