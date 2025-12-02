# TUI File Browser for Remote Paths

## Overview

When user runs `vget <remote>:/path/to/directory/`, instead of showing an error, vget displays an interactive TUI browser to navigate and select files for download.

## Behavior

```bash
vget pikpak:/电影/          # Directory → Opens TUI browser
vget pikpak:/电影/file.mkv  # File → Direct download
```

## Key Features

### Navigation
- `↑/↓` or `k/j`: Move cursor up/down
- `Enter`: Enter directory / Download file
- `b` or `Backspace` or `h`: Go up one directory
- `q` or `Esc`: Quit without downloading

### Display
- Current path shown as header
- Directories listed first (with 📁 icon), then files
- File sizes displayed
- Scrollable list with position indicator
- Highlighted cursor row

### Selection Behavior
- Enter on directory: navigate into it (stay in TUI)
- Enter on file: exit TUI and start download
- User can navigate through multiple directory levels before selecting a file

## Implementation

### Files
- `internal/cli/browse.go` - TUI browser component using Bubbletea
- `internal/cli/root.go` - Integration point in `runWebDAVDownload()`

### Flow
1. User runs `vget pikpak:/电影/`
2. vget detects it's a directory
3. TUI browser opens showing directory contents
4. User navigates and selects a file
5. TUI closes and download begins
