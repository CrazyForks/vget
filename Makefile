.PHONY: build build-ui build-metal build-cuda build-nocgo build-libs build-whisper setup-sherpa push version patch minor major

BUILD_DIR := ./build
VERSION_FILE := internal/core/version/version.go
UI_DIR := ./ui
SERVER_DIST := ./internal/server/dist

# Get current version from latest git tag (strips 'v' prefix)
CURRENT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")

# Get whisper.cpp module path
WHISPER_PATH := $(shell go list -m -f '{{.Dir}}' github.com/ggerganov/whisper.cpp/bindings/go 2>/dev/null)

# sherpa-onnx version
SHERPA_VERSION := 1.12.20

# Detect platform for sherpa-onnx download
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),Darwin)
	ifeq ($(UNAME_M),arm64)
		SHERPA_PLATFORM := osx-arm64
	else
		SHERPA_PLATFORM := osx-x86_64
	endif
else
	ifeq ($(UNAME_M),aarch64)
		SHERPA_PLATFORM := linux-aarch64
	else
		SHERPA_PLATFORM := linux-x64
	endif
endif

build-ui:
	cd $(UI_DIR) && npm install && npm run build
	rm -rf $(SERVER_DIST)/*
	cp -r $(UI_DIR)/dist/* $(SERVER_DIST)/

# Download and install sherpa-onnx (for Parakeet)
setup-sherpa:
	@echo "Downloading sherpa-onnx v$(SHERPA_VERSION) for $(SHERPA_PLATFORM)..."
	@mkdir -p $(BUILD_DIR)/sherpa
	curl -L "https://github.com/k2-fsa/sherpa-onnx/releases/download/v$(SHERPA_VERSION)/sherpa-onnx-v$(SHERPA_VERSION)-$(SHERPA_PLATFORM)-shared.tar.bz2" | \
		tar -xjf - -C $(BUILD_DIR)/sherpa --strip-components=1
	@echo "sherpa-onnx installed at $(BUILD_DIR)/sherpa"

# Build whisper.cpp static library (for Whisper)
build-whisper:
	@if [ -z "$(WHISPER_PATH)" ]; then \
		echo "Error: whisper.cpp module not found. Run 'go mod download' first."; \
		exit 1; \
	fi
	cd "$(WHISPER_PATH)" && make whisper
	@echo "whisper.cpp library built at $(WHISPER_PATH)/libwhisper.a"

# Build both native libraries (run once before build)
build-libs: setup-sherpa build-whisper
	@echo "All native libraries ready"

# Standard build with CGO for sherpa-onnx + whisper.cpp (CPU only)
# Requires: make build-libs (run once)
build: build-ui
	CGO_ENABLED=1 \
	C_INCLUDE_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/include" \
	LIBRARY_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/lib" \
	go build -o $(BUILD_DIR)/vget ./cmd/vget
	CGO_ENABLED=1 \
	C_INCLUDE_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/include" \
	LIBRARY_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/lib" \
	go build -o $(BUILD_DIR)/vget-server ./cmd/vget-server

# macOS with Metal acceleration (Apple Silicon)
# Requires: WHISPER_METAL=1 make build-whisper && make setup-sherpa (run once)
build-metal: build-ui
	CGO_ENABLED=1 \
	C_INCLUDE_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/include" \
	LIBRARY_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/lib" \
	go build -tags metal -o $(BUILD_DIR)/vget ./cmd/vget
	CGO_ENABLED=1 \
	C_INCLUDE_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/include" \
	LIBRARY_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/lib" \
	go build -tags metal -o $(BUILD_DIR)/vget-server ./cmd/vget-server

# Linux with CUDA acceleration (NVIDIA GPU)
# Requires: GGML_CUDA=1 make build-whisper && make setup-sherpa (run once)
build-cuda: build-ui
	CGO_ENABLED=1 \
	C_INCLUDE_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/include" \
	LIBRARY_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/lib" \
	CGO_CFLAGS="-I/usr/local/cuda/include" \
	CGO_LDFLAGS="-L/usr/local/cuda/lib64" \
	go build -tags cuda -o $(BUILD_DIR)/vget ./cmd/vget
	CGO_ENABLED=1 \
	C_INCLUDE_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/include" \
	LIBRARY_PATH="$(WHISPER_PATH):$(BUILD_DIR)/sherpa/lib" \
	CGO_CFLAGS="-I/usr/local/cuda/include" \
	CGO_LDFLAGS="-L/usr/local/cuda/lib64" \
	go build -tags cuda -o $(BUILD_DIR)/vget-server ./cmd/vget-server

# Build without CGO (disables local transcription)
build-nocgo: build-ui
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/vget ./cmd/vget
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/vget-server ./cmd/vget-server

push:
	git push origin main --tags

# Version bump: make version <patch|minor|major>
version:
	@if [ -z "$(filter patch minor major,$(MAKECMDGOALS))" ]; then \
		echo "Usage: make version <patch|minor|major>"; \
		echo "Current version: $(CURRENT_VERSION)"; \
		exit 1; \
	fi

patch minor major: version
	@TYPE=$@ && \
	echo "Current version: $(CURRENT_VERSION)" && \
	NEW_VERSION=$$(echo "$(CURRENT_VERSION)" | awk -F. -v type="$$TYPE" '{ \
		if (type == "major") { print $$1+1".0.0" } \
		else if (type == "minor") { print $$1"."$$2+1".0" } \
		else { print $$1"."$$2"."$$3+1 } \
	}') && \
	BUILD_DATE=$$(date -u +"%Y-%m-%d") && \
	echo "New version: $$NEW_VERSION" && \
	echo "Build date: $$BUILD_DATE" && \
	sed -i '' 's/Version = ".*"/Version = "'$$NEW_VERSION'"/' $(VERSION_FILE) && \
	sed -i '' 's/Date    = ".*"/Date    = "'$$BUILD_DATE'"/' $(VERSION_FILE) && \
	git add $(VERSION_FILE) && \
	git commit -m "chore: bump version to v$$NEW_VERSION" && \
	git tag "v$$NEW_VERSION" && \
	echo "Created tag v$$NEW_VERSION" && \
	echo "Run 'make push' to push changes and trigger release"