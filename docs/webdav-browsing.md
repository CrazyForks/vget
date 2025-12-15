# WebDAV File Browsing (Web UI)

## Overview

Add file browsing capability to the vget Web UI for WebDAV remotes. Currently, browsing only works via CLI (`vget ls`). Users should be able to browse, navigate, and download files from WebDAV servers directly in the web interface.

## Current State

### What Works (CLI)
```bash
vget ls pikpak:/              # List root
vget ls pikpak:/Movies        # List subdirectory
vget pikpak:/Movies/film.mp4  # Download file
```

### What's Missing (Web UI)
- No way to browse WebDAV files in the browser
- Users must use CLI to discover file paths
- No visual navigation of remote directories
- No click-to-download functionality

## Motivation

The web UI is designed for convenience - users shouldn't need to switch to CLI just to browse files. A file browser makes WebDAV support actually usable:

1. **Discovery** - See what's available without memorizing paths
2. **Navigation** - Click through directories naturally
3. **Download** - Select files and download with one click
4. **Mobile-friendly** - Browse from phone/tablet (no CLI)

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Web UI (Browser)                                               │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  File Browser Component                                  │   │
│  │  - Directory tree / breadcrumb navigation               │   │
│  │  - File list with size, type                            │   │
│  │  - Select & download actions                            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              │ fetch()                          │
│                              ▼                                  │
└─────────────────────────────────────────────────────────────────┘
                               │
                               │ HTTP API
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│  vget Backend (Go)                                              │
│                                                                 │
│  GET /api/webdav/list?remote=pikpak&path=/Movies                │
│  POST /api/webdav/download                                      │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  internal/core/webdav/client.go                         │   │
│  │  - List() ✓ (already exists)                            │   │
│  │  - Stat() ✓ (already exists)                            │   │
│  │  - Open() ✓ (already exists)                            │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│                              │ WebDAV (PROPFIND/GET)           │
│                              ▼                                  │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│  WebDAV Server (PikPak, Alist, Synology, etc.)                 │
└─────────────────────────────────────────────────────────────────┘
```

## UI Design

### WebDAV Page Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  WebDAV                                               [Settings]│
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Remote: [pikpak          ▼]                                    │
│                                                                 │
│  ┌─── File Browser ──────────────────────────────────────────┐ │
│  │                                                           │ │
│  │  📍 pikpak: / Movies / Action /                           │ │
│  │  ─────────────────────────────────────────────────────── │ │
│  │  [ ] Name                          Size        Modified   │ │
│  │  ─────────────────────────────────────────────────────── │ │
│  │      📁 ..                          -                     │ │
│  │  [ ] 📁 Subtitles/                  -          2024-01-15│ │
│  │  [x] 📄 movie-1080p.mkv            4.7 GB     2024-01-13│ │
│  │  [ ] 📄 movie-720p.mkv             2.1 GB     2024-01-13│ │
│  │  [x] 📄 movie.srt                  45 KB      2024-01-13│ │
│  │  ─────────────────────────────────────────────────────── │ │
│  │                                                           │ │
│  │  Selected: 2 files (4.7 GB)                              │ │
│  │                                                           │ │
│  │  [Download Selected]  [Download All]                      │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─── Active Downloads ──────────────────────────────────────┐ │
│  │                                                           │ │
│  │  movie-1080p.mkv                                         │ │
│  │  ████████████████████░░░░░░░░░░░░  45%  23.5 MB/s        │ │
│  │                                                           │ │
│  │  movie.srt                                               │ │
│  │  ████████████████████████████████  100%  Complete        │ │
│  │                                                           │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Navigation Features

1. **Breadcrumb** - Click any part to jump: `pikpak: / Movies / Action /`
2. **Parent directory** - `📁 ..` row to go up one level
3. **Click folder** - Navigate into subdirectory
4. **Click file** - Preview info or start download
5. **Checkbox select** - Multi-select for batch download

### Remote Selector

Dropdown shows all configured WebDAV servers from config:
```
[pikpak          ▼]
 ├─ pikpak
 ├─ alist
 └─ synology
```

## API Endpoints

### List Directory
```
GET /api/webdav/list?remote=pikpak&path=/Movies

Response:
{
  "remote": "pikpak",
  "path": "/Movies",
  "files": [
    {"name": "Action", "path": "/Movies/Action", "isDir": true, "size": 0, "modTime": "2024-01-15T10:30:00Z"},
    {"name": "movie.mkv", "path": "/Movies/movie.mkv", "isDir": false, "size": 5000000000, "modTime": "2024-01-13T08:00:00Z"}
  ]
}
```

### Get File Info
```
GET /api/webdav/info?remote=pikpak&path=/Movies/movie.mkv

Response:
{
  "name": "movie.mkv",
  "path": "/Movies/movie.mkv",
  "isDir": false,
  "size": 5000000000,
  "modTime": "2024-01-13T08:00:00Z",
  "downloadUrl": "pikpak:/Movies/movie.mkv"
}
```

### Download File(s)
```
POST /api/webdav/download
{
  "remote": "pikpak",
  "files": [
    "/Movies/movie.mkv",
    "/Movies/movie.srt"
  ],
  "outputDir": "/downloads"  // optional, uses default if not specified
}

Response:
{
  "taskId": "download-456",
  "status": "started",
  "files": [
    {"path": "/Movies/movie.mkv", "status": "queued"},
    {"path": "/Movies/movie.srt", "status": "queued"}
  ]
}
```

### Download Progress
```
GET /api/webdav/download/download-456

Response:
{
  "taskId": "download-456",
  "status": "in_progress",
  "files": [
    {"path": "/Movies/movie.mkv", "progress": 45, "speed": 24600000, "status": "downloading"},
    {"path": "/Movies/movie.srt", "progress": 100, "status": "completed"}
  ],
  "totalSize": 5000045000,
  "downloaded": 2250045000
}
```

### List Configured Remotes
```
GET /api/webdav/remotes

Response:
{
  "remotes": [
    {"name": "pikpak", "url": "https://dav.pikpak.com", "hasAuth": true},
    {"name": "alist", "url": "http://192.168.1.100:5244/dav", "hasAuth": true}
  ]
}
```

## Backend Implementation

### New Files
```
internal/server/webdav_browse.go    # API handlers for browsing
```

### Handler Code (webdav_browse.go)

```go
package server

import (
    "encoding/json"
    "net/http"

    "github.com/guiyumin/vget/internal/core/config"
    "github.com/guiyumin/vget/internal/core/webdav"
)

// GET /api/webdav/remotes
func (s *Server) handleWebDAVRemotes(w http.ResponseWriter, r *http.Request) {
    cfg := config.LoadOrDefault()

    remotes := make([]map[string]interface{}, 0)
    for name, server := range cfg.WebDAVServers {
        remotes = append(remotes, map[string]interface{}{
            "name":    name,
            "url":     server.URL,
            "hasAuth": server.Username != "",
        })
    }

    json.NewEncoder(w).Encode(map[string]interface{}{
        "remotes": remotes,
    })
}

// GET /api/webdav/list?remote=xxx&path=/xxx
func (s *Server) handleWebDAVList(w http.ResponseWriter, r *http.Request) {
    remoteName := r.URL.Query().Get("remote")
    path := r.URL.Query().Get("path")
    if path == "" {
        path = "/"
    }

    cfg := config.LoadOrDefault()
    server := cfg.GetWebDAVServer(remoteName)
    if server == nil {
        http.Error(w, "Remote not found", http.StatusNotFound)
        return
    }

    client, err := webdav.NewClientFromConfig(server)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    files, err := client.List(r.Context(), path)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Convert to JSON response format
    // ... (format files array)

    json.NewEncoder(w).Encode(map[string]interface{}{
        "remote": remoteName,
        "path":   path,
        "files":  files,
    })
}

// POST /api/webdav/download
func (s *Server) handleWebDAVDownload(w http.ResponseWriter, r *http.Request) {
    // Parse request body
    // Queue download tasks
    // Return task ID for progress tracking
}
```

### Route Registration

```go
// internal/server/server.go

func (s *Server) setupRoutes() {
    // ... existing routes ...

    // WebDAV browsing
    s.router.HandleFunc("/api/webdav/remotes", s.handleWebDAVRemotes).Methods("GET")
    s.router.HandleFunc("/api/webdav/list", s.handleWebDAVList).Methods("GET")
    s.router.HandleFunc("/api/webdav/info", s.handleWebDAVInfo).Methods("GET")
    s.router.HandleFunc("/api/webdav/download", s.handleWebDAVDownload).Methods("POST")
    s.router.HandleFunc("/api/webdav/download/{taskId}", s.handleWebDAVDownloadProgress).Methods("GET")
}
```

## Frontend Implementation

### New Files
```
ui/src/pages/WebDAVPage.tsx           # Main page
ui/src/components/WebDAVBrowser.tsx   # File browser component
ui/src/components/WebDAVDownloads.tsx # Download progress component
ui/src/routes/webdav.tsx              # Route definition
```

### Component Structure

```tsx
// ui/src/pages/WebDAVPage.tsx

export function WebDAVPage() {
  const [selectedRemote, setSelectedRemote] = useState<string>("");
  const [currentPath, setCurrentPath] = useState<string>("/");
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [downloads, setDownloads] = useState<DownloadTask[]>([]);

  return (
    <div>
      <RemoteSelector
        value={selectedRemote}
        onChange={setSelectedRemote}
      />

      <WebDAVBrowser
        remote={selectedRemote}
        path={currentPath}
        files={files}
        selectedFiles={selectedFiles}
        onNavigate={setCurrentPath}
        onSelect={handleSelect}
        onDownload={handleDownload}
      />

      <WebDAVDownloads tasks={downloads} />
    </div>
  );
}
```

### File Browser Component

```tsx
// ui/src/components/WebDAVBrowser.tsx

interface WebDAVBrowserProps {
  remote: string;
  path: string;
  files: FileInfo[];
  selectedFiles: Set<string>;
  onNavigate: (path: string) => void;
  onSelect: (path: string, selected: boolean) => void;
  onDownload: (files: string[]) => void;
}

export function WebDAVBrowser(props: WebDAVBrowserProps) {
  // Breadcrumb navigation
  const pathParts = props.path.split('/').filter(Boolean);

  return (
    <div className="webdav-browser">
      {/* Breadcrumb */}
      <nav className="breadcrumb">
        <span onClick={() => props.onNavigate('/')}>{props.remote}:</span>
        <span> / </span>
        {pathParts.map((part, i) => (
          <span key={i}>
            <span onClick={() => props.onNavigate('/' + pathParts.slice(0, i + 1).join('/'))}>
              {part}
            </span>
            <span> / </span>
          </span>
        ))}
      </nav>

      {/* File list */}
      <table className="file-list">
        <thead>
          <tr>
            <th><input type="checkbox" /></th>
            <th>Name</th>
            <th>Size</th>
            <th>Modified</th>
          </tr>
        </thead>
        <tbody>
          {/* Parent directory */}
          {props.path !== '/' && (
            <tr onClick={() => props.onNavigate(getParentPath(props.path))}>
              <td></td>
              <td>📁 ..</td>
              <td>-</td>
              <td></td>
            </tr>
          )}

          {/* Files and folders */}
          {props.files.map(file => (
            <tr key={file.path}>
              <td>
                {!file.isDir && (
                  <input
                    type="checkbox"
                    checked={props.selectedFiles.has(file.path)}
                    onChange={(e) => props.onSelect(file.path, e.target.checked)}
                  />
                )}
              </td>
              <td onClick={() => file.isDir && props.onNavigate(file.path)}>
                {file.isDir ? '📁' : '📄'} {file.name}
              </td>
              <td>{file.isDir ? '-' : formatSize(file.size)}</td>
              <td>{formatDate(file.modTime)}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {/* Actions */}
      <div className="actions">
        <span>Selected: {props.selectedFiles.size} files</span>
        <button onClick={() => props.onDownload(Array.from(props.selectedFiles))}>
          Download Selected
        </button>
      </div>
    </div>
  );
}
```

## Translations

Add to all locale files:

```yaml
# internal/core/i18n/locales/en.yml
webdav_browser:
  title: "File Browser"
  select_remote: "Select Remote"
  no_remotes: "No WebDAV servers configured"
  add_remote: "Add Server"
  empty_directory: "Empty directory"
  parent_directory: "Parent directory"
  download_selected: "Download Selected"
  download_all: "Download All"
  selected_count: "Selected: %d files (%s)"
  downloading: "Downloading"
  completed: "Completed"
  failed: "Failed"
```

## Implementation Plan

### Phase 1: Backend API
- [ ] `internal/server/webdav_browse.go` - API handlers
- [ ] Register routes in `internal/server/server.go`
- [ ] Download task queue with progress tracking

### Phase 2: Frontend Components
- [ ] `ui/src/pages/WebDAVPage.tsx` - Main page
- [ ] `ui/src/components/WebDAVBrowser.tsx` - File browser
- [ ] `ui/src/components/WebDAVDownloads.tsx` - Download progress
- [ ] Add to sidebar navigation
- [ ] Add routes

### Phase 3: Integration
- [ ] Connect browser to download API
- [ ] Progress polling / WebSocket for updates
- [ ] Error handling (auth failures, network errors)

### Phase 4: Polish
- [ ] i18n translations (all locales)
- [ ] Loading states and skeletons
- [ ] Empty states
- [ ] Mobile responsive design

## Relationship to Seedbox Feature

The WebDAV browser and Seedbox browser share similar UI patterns:

| Feature | WebDAV Browser | Seedbox Browser |
|---------|---------------|-----------------|
| Browse files | WebDAV PROPFIND | HTTP index / SFTP |
| Download | WebDAV GET | HTTP GET / SFTP |
| Backend | `webdav.Client` | `seedbox.HTTPFileServer` / `seedbox.SFTPFileServer` |
| UI Component | Can share `FileBrowser` base component |

Consider extracting a shared `FileBrowser` component that both features can use:

```tsx
// ui/src/components/FileBrowser.tsx (shared)
interface FileBrowserProps {
  files: FileInfo[];
  currentPath: string;
  onNavigate: (path: string) => void;
  onSelect: (paths: string[]) => void;
  onDownload: (paths: string[]) => void;
}
```

## Security Considerations

- WebDAV credentials already stored in config (same security model)
- API endpoints require same auth as other vget endpoints
- No cross-remote access (can only browse configured remotes)
- Path traversal protection (validate paths stay within remote)

## Future Enhancements (Not Planned)

- Upload files to WebDAV
- Rename/delete files
- Create directories
- Search within remote
- Favorites/bookmarks
- Recent files history
