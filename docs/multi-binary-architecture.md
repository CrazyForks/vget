# Multi-Binary Architecture Plan

## Overview

Split vget into three separate binaries with a shared core module:

| Binary | Purpose | Distribution |
|--------|---------|--------------|
| `vget` | CLI tool | GitHub Releases |
| `vget-server` | HTTP server + Web UI | Docker Image |
| `vget-desktop` | Desktop GUI (Fyne) | GitHub Releases |

## Target Structure

```
cmd/
  vget/main.go              # CLI entry point
  vget-server/main.go       # Server entry point
  vget-desktop/main.go      # Desktop entry point

internal/
  core/                     # Shared by all three binaries
    extractor/              # URL matching, media extraction
    downloader/             # Download logic, progress callbacks
    config/                 # Config file management
    i18n/                   # Translations
    version/                # Version info
    webdav/                 # WebDAV client

  cli/                      # CLI-specific (Cobra + Bubbletea)
  server/                   # Server-specific (HTTP + job queue)
  desktop/                  # Desktop-specific (Fyne UI) [new]
  updater/                  # Self-update (CLI + Desktop only)
```

## Phase 1: Create Core Module

Move shared packages into `internal/core/`:

```bash
internal/core/
  extractor/    ← move from internal/extractor/
  downloader/   ← move from internal/downloader/
  config/       ← move from internal/config/
  i18n/         ← move from internal/i18n/
  version/      ← move from internal/version/
  webdav/       ← move from internal/webdav/
```

Update all imports:
```go
// Before
import "github.com/guiyumin/vget/internal/extractor"

// After
import "github.com/guiyumin/vget/internal/core/extractor"
```

## Phase 2: Decouple Downloader from TUI

Current state: `internal/downloader/` contains Bubbletea TUI code.

Target: Pure download logic with progress callbacks.

### 2.1 Create callback-based API

```go
// internal/core/downloader/download.go
package downloader

type ProgressFunc func(downloaded, total int64)
type StatusFunc func(status string)

type DownloadOptions struct {
    URL       string
    Output    string
    Headers   map[string]string
    OnProgress ProgressFunc
    OnStatus   StatusFunc
}

func Download(opts DownloadOptions) error {
    // Pure download logic, no TUI
}
```

### 2.2 Move TUI to CLI package

```go
// internal/cli/download_tui.go
package cli

import (
    "github.com/guiyumin/vget/internal/core/downloader"
    tea "github.com/charmbracelet/bubbletea"
)

func RunDownloadWithTUI(url, output string) error {
    // Bubbletea model wraps core downloader
}
```

## Phase 3: Split Server from CLI

Current state: Server commands in `internal/cli/server.go`

### 3.1 Create `cmd/vget-server/main.go`

```go
package main

import (
    "github.com/guiyumin/vget/internal/server"
    "github.com/guiyumin/vget/internal/core/config"
)

func main() {
    cfg := config.LoadOrDefault()
    srv := server.NewServer(cfg)
    srv.Start()
}
```

### 3.2 Remove server commands from CLI

Delete or move:
- `internal/cli/server.go` → `cmd/vget-server/` or `internal/server/cmd.go`

Keep in `cmd/vget/`:
- Download command
- `init` wizard
- `config` commands
- `search` command
- `update` command
- `ls` command (WebDAV)

## Phase 4: Update Dockerfile

```dockerfile
# Build vget-server instead of vget
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /vget-server ./cmd/vget-server

# ...

COPY --from=go-builder /vget-server /usr/local/bin/vget-server
ENTRYPOINT ["vget-server"]
```

## Phase 5: Desktop App (Future)

Create `cmd/vget-desktop/main.go` using Fyne:

```go
package main

import (
    "fyne.io/fyne/v2/app"
    "github.com/guiyumin/vget/internal/desktop"
)

func main() {
    a := app.New()
    w := a.NewWindow("VGet")
    desktop.SetupUI(w)
    w.ShowAndRun()
}
```

Desktop-specific UI in `internal/desktop/`:
- URL input
- Format selection
- Download progress
- Settings

## Implementation Order

### Step 1: Core module extraction
- [ ] Create `internal/core/` directory
- [ ] Move extractor, downloader, config, i18n, version, webdav
- [ ] Update all imports (use IDE refactor or `sed`)
- [ ] Verify build: `go build ./...`

### Step 2: Downloader decoupling
- [ ] Identify Bubbletea dependencies in downloader
- [ ] Create callback-based download API
- [ ] Move TUI wrappers to `internal/cli/`
- [ ] Verify CLI still works

### Step 3: Server separation
- [ ] Create `cmd/vget-server/main.go`
- [ ] Move server startup logic from `internal/cli/server.go`
- [ ] Remove server commands from `internal/cli/`
- [ ] Update Dockerfile
- [ ] Verify Docker image works

### Step 4: Desktop app
- [ ] Add Fyne dependency
- [ ] Create `cmd/vget-desktop/main.go`
- [ ] Create `internal/desktop/` UI components
- [ ] Build and test on macOS, Windows, Linux

## Build Commands

```bash
# CLI only
go build -o build/vget ./cmd/vget

# Server (for Docker)
go build -o build/vget-server ./cmd/vget-server

# Desktop
go build -o build/vget-desktop ./cmd/vget-desktop

# All
go build ./cmd/...
```

## Release Artifacts

| Platform | CLI | Server | Desktop |
|----------|-----|--------|---------|
| Linux amd64 | vget-linux-amd64 | Docker image | vget-desktop-linux-amd64 |
| Linux arm64 | vget-linux-arm64 | Docker image | vget-desktop-linux-arm64 |
| macOS amd64 | vget-darwin-amd64 | - | vget-desktop-darwin-amd64 |
| macOS arm64 | vget-darwin-arm64 | - | vget-desktop-darwin-arm64 |
| Windows | vget-windows-amd64.exe | - | vget-desktop-windows-amd64.exe |

## Notes

- Server binary does NOT need self-update (Docker handles this)
- Desktop app needs Fyne dependencies (~15MB binary size increase)
- Core module has zero UI dependencies (no Bubbletea, no Fyne)
- CLI and Desktop can both use `internal/updater/` for self-update
