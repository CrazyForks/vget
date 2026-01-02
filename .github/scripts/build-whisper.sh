#!/bin/bash
set -e

WHISPER_VERSION="${WHISPER_VERSION:-v1.8.2}"
WHISPER_REPO="https://github.com/ggerganov/whisper.cpp"
OUTPUT_DIR="${OUTPUT_DIR:-internal/core/ai/transcriber/bin}"

echo "Building whisper.cpp ${WHISPER_VERSION}..."
echo "Platform: $(uname -s)-$(uname -m)"

# Determine output filename
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

case "${GOOS}-${GOARCH}" in
    darwin-arm64)
        OUTPUT_NAME="whisper-darwin-arm64"
        CMAKE_FLAGS="-DWHISPER_METAL=ON"
        ;;
    darwin-amd64)
        OUTPUT_NAME="whisper-darwin-amd64"
        CMAKE_FLAGS="-DWHISPER_ACCELERATE=ON"
        ;;
    linux-amd64)
        OUTPUT_NAME="whisper-linux-amd64"
        CMAKE_FLAGS="-DWHISPER_OPENBLAS=ON"
        ;;
    linux-arm64)
        OUTPUT_NAME="whisper-linux-arm64"
        CMAKE_FLAGS="-DWHISPER_OPENBLAS=ON"
        ;;
    windows-amd64)
        OUTPUT_NAME="whisper-windows-amd64.exe"
        CMAKE_FLAGS="-DWHISPER_OPENBLAS=ON"
        ;;
    *)
        echo "Unsupported platform: ${GOOS}-${GOARCH}"
        exit 1
        ;;
esac

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Clone whisper.cpp
if [ -d "whisper.cpp" ]; then
    rm -rf whisper.cpp
fi
git clone --depth 1 --branch "${WHISPER_VERSION}" "${WHISPER_REPO}" whisper.cpp

# Build
cd whisper.cpp
cmake -B build -DCMAKE_BUILD_TYPE=Release ${CMAKE_FLAGS}
cmake --build build --config Release -j$(nproc 2>/dev/null || sysctl -n hw.ncpu)

# Copy binary
if [ -f "build/bin/whisper-cli" ]; then
    cp "build/bin/whisper-cli" "../${OUTPUT_DIR}/${OUTPUT_NAME}"
elif [ -f "build/bin/whisper" ]; then
    cp "build/bin/whisper" "../${OUTPUT_DIR}/${OUTPUT_NAME}"
else
    echo "Error: whisper binary not found in build/bin/"
    ls -la build/bin/
    exit 1
fi

# Cleanup
cd ..
rm -rf whisper.cpp

echo "Built: ${OUTPUT_DIR}/${OUTPUT_NAME}"
ls -la "${OUTPUT_DIR}/${OUTPUT_NAME}"
