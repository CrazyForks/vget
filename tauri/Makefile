.PHONY: dev build install clean

# Development
dev:
	bun tauri dev

# Build for release
build:
	bun tauri build

# Install dependencies
install:
	bun install

# Clean build artifacts
clean:
	rm -rf dist
	rm -rf src-tauri/target

# Generate icons (requires source icon)
icons:
	bun tauri icon public/app-icon.png

# Type check
check:
	bun tsc --noEmit
	cd src-tauri && cargo check
