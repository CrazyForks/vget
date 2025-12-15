# Seedbox Support

## Overview

Extend vget to support seedboxes as remote torrent clients. Unlike NAS mode (dispatch only), seedbox mode includes:
1. **Dispatch** - Send magnet/torrent to seedbox
2. **Browse** - List files on seedbox
3. **Download** - Fetch completed files to NAS/local via HTTP/HTTPS or SFTP

## Motivation

Seedboxes are remote servers with high-bandwidth connections, commonly used for:
- Fast torrent downloads (datacenter speeds)
- Maintaining seed ratios on private trackers
- Avoiding ISP throttling/detection of P2P traffic

Users want to:
1. Send torrents to seedbox from vget
2. Browse what's on the seedbox
3. Download completed files via HTTP/HTTPS (looks like normal web traffic to ISP)

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  vget (Docker or CLI)                                               │
│                                                                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐             │
│  │   Dispatch  │    │   Browse    │    │  Download   │             │
│  │   Torrent   │    │   Files     │    │   Files     │             │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘             │
└─────────┼──────────────────┼──────────────────┼─────────────────────┘
          │                  │                  │
          │ RPC/API          │ SFTP/HTTP        │ HTTP/SFTP
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Seedbox                                                            │
│                                                                     │
│  ┌─────────────────┐    ┌─────────────────┐                        │
│  │ Torrent Client  │    │   File Server   │                        │
│  │ - rTorrent      │    │ - nginx/HTTP    │                        │
│  │ - Deluge        │    │ - SFTP (SSH)    │                        │
│  │ - qBittorrent   │    │                 │                        │
│  │ - Transmission  │    │                 │                        │
│  └─────────────────┘    └─────────────────┘                        │
│           │                     │                                   │
│           ▼                     ▼                                   │
│  ┌─────────────────────────────────────────┐                       │
│  │           /downloads/                    │                       │
│  │  ├── movie.mkv                          │                       │
│  │  ├── album/                             │                       │
│  │  │   ├── 01-track.flac                  │                       │
│  │  │   └── 02-track.flac                  │                       │
│  │  └── series/                            │                       │
│  └─────────────────────────────────────────┘                       │
└─────────────────────────────────────────────────────────────────────┘
          │
          │ HTTP/HTTPS or SFTP
          ▼
┌─────────────────────────────────────────────────────────────────────┐
│  Destination (NAS or Local)                                         │
│  /volume1/downloads/ or ~/Downloads/                                │
└─────────────────────────────────────────────────────────────────────┘
```

## Supported Torrent Clients

### Existing (from NAS mode)
| Client | Protocol | Default Port | Status |
|--------|----------|--------------|--------|
| Transmission | JSON-RPC | 9091 | Done |
| qBittorrent | REST API | 8080 | Done |
| Synology DS | REST API | 5000/5001 | Done |

### New (for seedbox)
| Client | Protocol | Default Port | Status |
|--------|----------|--------------|--------|
| rTorrent | XML-RPC | 8080 (via ruTorrent) | TODO |
| Deluge | JSON-RPC | 8112 | TODO |

### rTorrent (XML-RPC)

Most common on seedboxes. Usually accessed via:
- ruTorrent web UI (PHP frontend)
- Direct XML-RPC endpoint (often `/RPC2` or `/rutorrent/plugins/httprpc/action.php`)

```go
// internal/torrent/rtorrent.go
type RTorrentClient struct {
    endpoint string  // e.g., "https://seedbox.example.com/RPC2"
    username string
    password string
}

// XML-RPC methods:
// - load.raw_start (add torrent from base64 data)
// - load.start (add torrent from URL/magnet)
// - d.multicall2 (list torrents)
// - d.name, d.size_bytes, d.completed_bytes, d.ratio, etc.
```

### Deluge (JSON-RPC)

Popular alternative with web UI.

```go
// internal/torrent/deluge.go
type DelugeClient struct {
    host     string  // e.g., "seedbox.example.com:8112"
    password string  // Deluge uses single password, no username
    useTLS   bool
}

// JSON-RPC methods (via /json endpoint):
// - auth.login
// - core.add_torrent_magnet
// - core.add_torrent_url
// - core.get_torrents_status
```

## File Access Methods

### HTTP/HTTPS (Primary)

Most seedboxes run nginx/apache serving the downloads directory. This is ideal because:
- Looks like normal web traffic to ISP
- No P2P protocol detection
- Often faster than SFTP for large files
- Resume support via Range headers

```
Seedbox URL: https://user.seedbox.io/downloads/
            https://user.seedbox.io/downloads/movie.mkv
            https://user.seedbox.io/downloads/album/01-track.flac
```

**Implementation:**
```go
// internal/seedbox/http.go
type HTTPFileServer struct {
    baseURL  string  // e.g., "https://user.seedbox.io/downloads/"
    username string  // HTTP Basic Auth
    password string
}

func (h *HTTPFileServer) List(path string) ([]FileInfo, error)
func (h *HTTPFileServer) Download(remotePath, localPath string, progress func(int64, int64)) error
```

**Directory listing:** Parse HTML index page or use JSON index if available.

### SFTP (Alternative)

Universal fallback - every seedbox has SSH access.

```go
// internal/seedbox/sftp.go
type SFTPFileServer struct {
    host       string  // e.g., "seedbox.example.com:22"
    username   string
    password   string  // or privateKey
    privateKey string  // path to SSH key
    basePath   string  // e.g., "/home/user/downloads"
}

func (s *SFTPFileServer) List(path string) ([]FileInfo, error)
func (s *SFTPFileServer) Download(remotePath, localPath string, progress func(int64, int64)) error
```

**Library:** Use `github.com/pkg/sftp` with `golang.org/x/crypto/ssh`

### FileInfo Structure

```go
// internal/seedbox/types.go
type FileInfo struct {
    Name    string    `json:"name"`
    Path    string    `json:"path"`     // relative path from base
    Size    int64     `json:"size"`
    IsDir   bool      `json:"isDir"`
    ModTime time.Time `json:"modTime"`
}
```

## UI Design

### Seedbox Page (Web UI)

```
┌─────────────────────────────────────────────────────────────────┐
│  Seedbox                                              [Settings]│
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─── Add Torrent ───────────────────────────────────────────┐ │
│  │                                                           │ │
│  │  [Magnet link or .torrent URL                        ]   │ │
│  │                                                           │ │
│  │  [ ] Start paused                      [Send to Seedbox] │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─── Browse Files ──────────────────────────────────────────┐ │
│  │                                                           │ │
│  │  Path: /downloads/                            [Refresh]   │ │
│  │  ─────────────────────────────────────────────────────── │ │
│  │  [ ] Name                          Size        Modified   │ │
│  │  ─────────────────────────────────────────────────────── │ │
│  │  [ ] 📁 movies/                     -          2024-01-15│ │
│  │  [ ] 📁 music/                      -          2024-01-14│ │
│  │  [x] 📄 ubuntu-24.04.iso           4.7 GB     2024-01-13│ │
│  │  [ ] 📄 document.pdf               2.3 MB     2024-01-12│ │
│  │  ─────────────────────────────────────────────────────── │ │
│  │                                                           │ │
│  │  Selected: 1 file (4.7 GB)                               │ │
│  │                                                           │ │
│  │  Download to: [/volume1/downloads     ▼]  [Download]     │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─── Active Torrents ───────────────────────────────────────┐ │
│  │                                                           │ │
│  │  ubuntu-24.04.iso                                        │ │
│  │  ████████████████████████████░░░░░░  85%  12.3 MB/s      │ │
│  │                                                           │ │
│  │  archlinux-2024.01.01.iso                                │ │
│  │  ████████████████████████████████████  100%  Seeding     │ │
│  │                                                           │ │
│  └───────────────────────────────────────────────────────────┘ │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Settings Modal

```
┌─────────────────────────────────────────────────────────────────┐
│  Seedbox Settings                                          [X] │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Torrent Client                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Client:    [rTorrent (ruTorrent)  ▼]                   │   │
│  │  RPC URL:   [https://my.seedbox.io/rutorrent/plugins/ht]│   │
│  │  Username:  [myuser                  ]                   │   │
│  │  Password:  [••••••••                ]                   │   │
│  │                                    [Test Connection]     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  File Access                                                    │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Method:    (•) HTTP/HTTPS    ( ) SFTP                  │   │
│  │                                                         │   │
│  │  ── HTTP/HTTPS Settings ──                              │   │
│  │  Base URL:  [https://my.seedbox.io/downloads/       ]   │   │
│  │  Username:  [myuser                  ]                   │   │
│  │  Password:  [••••••••                ]                   │   │
│  │                                                         │   │
│  │  ── SFTP Settings (if selected) ──                      │   │
│  │  Host:      [my.seedbox.io:22        ]                   │   │
│  │  Username:  [myuser                  ]                   │   │
│  │  Auth:      (•) Password  ( ) SSH Key                   │   │
│  │  Password:  [••••••••                ]                   │   │
│  │  Base Path: [/home/myuser/downloads  ]                   │   │
│  │                                    [Test Connection]     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Default Download Location                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Path:      [/volume1/downloads      ]                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│                              [Cancel]  [Save]                   │
└─────────────────────────────────────────────────────────────────┘
```

## API Endpoints

### Torrent Dispatch
```
POST /api/seedbox/torrent
{
  "magnet": "magnet:?xt=urn:btih:...",
  // or
  "torrentUrl": "https://example.com/file.torrent",
  "startPaused": false
}
```

### List Torrents
```
GET /api/seedbox/torrents

Response:
{
  "torrents": [
    {
      "id": "abc123",
      "name": "ubuntu-24.04.iso",
      "size": 5000000000,
      "downloaded": 4250000000,
      "progress": 85,
      "speed": 12900000,
      "status": "downloading",  // downloading, seeding, paused, error
      "ratio": 0.5
    }
  ]
}
```

### Browse Files
```
GET /api/seedbox/files?path=/downloads/

Response:
{
  "path": "/downloads/",
  "files": [
    {"name": "movies", "path": "/downloads/movies/", "isDir": true, "size": 0, "modTime": "..."},
    {"name": "ubuntu.iso", "path": "/downloads/ubuntu.iso", "isDir": false, "size": 5000000000, "modTime": "..."}
  ]
}
```

### Download Files
```
POST /api/seedbox/download
{
  "files": [
    "/downloads/ubuntu.iso",
    "/downloads/movies/"
  ],
  "destination": "/volume1/downloads"
}

Response:
{
  "taskId": "download-123",
  "status": "started"
}
```

### Download Progress
```
GET /api/seedbox/download/download-123

Response:
{
  "taskId": "download-123",
  "status": "in_progress",  // in_progress, completed, failed
  "files": [
    {"path": "/downloads/ubuntu.iso", "progress": 45, "speed": 50000000}
  ],
  "totalSize": 5000000000,
  "downloaded": 2250000000
}
```

## Configuration

### Config File Structure

```yaml
# ~/.config/vget/config.yml

seedbox:
  enabled: true

  # Torrent client settings
  client: rtorrent  # rtorrent, deluge, qbittorrent, transmission
  clientHost: "https://my.seedbox.io/rutorrent/plugins/httprpc/action.php"
  clientUsername: "myuser"
  clientPassword: "secret"

  # File access settings
  fileAccess: http  # http or sftp

  # HTTP settings (when fileAccess: http)
  httpBaseURL: "https://my.seedbox.io/downloads/"
  httpUsername: "myuser"
  httpPassword: "secret"

  # SFTP settings (when fileAccess: sftp)
  sftpHost: "my.seedbox.io:22"
  sftpUsername: "myuser"
  sftpPassword: "secret"
  sftpPrivateKey: ""  # path to SSH key, alternative to password
  sftpBasePath: "/home/myuser/downloads"

  # Download settings
  defaultDownloadPath: "/volume1/downloads"
```

### Go Config Struct

```go
// internal/core/config/config.go

type SeedboxConfig struct {
    Enabled bool `yaml:"enabled"`

    // Torrent client
    Client         string `yaml:"client"`          // rtorrent, deluge, qbittorrent, transmission
    ClientHost     string `yaml:"clientHost"`
    ClientUsername string `yaml:"clientUsername"`
    ClientPassword string `yaml:"clientPassword"`

    // File access
    FileAccess string `yaml:"fileAccess"`  // http or sftp

    // HTTP settings
    HTTPBaseURL  string `yaml:"httpBaseURL"`
    HTTPUsername string `yaml:"httpUsername"`
    HTTPPassword string `yaml:"httpPassword"`

    // SFTP settings
    SFTPHost       string `yaml:"sftpHost"`
    SFTPUsername   string `yaml:"sftpUsername"`
    SFTPPassword   string `yaml:"sftpPassword"`
    SFTPPrivateKey string `yaml:"sftpPrivateKey"`
    SFTPBasePath   string `yaml:"sftpBasePath"`

    // Download settings
    DefaultDownloadPath string `yaml:"defaultDownloadPath"`
}
```

## Implementation Plan

### Phase 1: New Torrent Clients
- [ ] `internal/torrent/rtorrent.go` - rTorrent XML-RPC client
- [ ] `internal/torrent/deluge.go` - Deluge JSON-RPC client
- [ ] Add to client factory in `internal/torrent/client.go`
- [ ] Test with Docker containers

### Phase 2: File Access Layer
- [ ] `internal/seedbox/types.go` - FileInfo, interfaces
- [ ] `internal/seedbox/http.go` - HTTP/HTTPS file browser/downloader
- [ ] `internal/seedbox/sftp.go` - SFTP file browser/downloader
- [ ] `internal/seedbox/manager.go` - Factory and download task management

### Phase 3: Backend API
- [ ] Add SeedboxConfig to `internal/core/config/config.go`
- [ ] `internal/server/seedbox.go` - API handlers
- [ ] Register routes in `internal/server/server.go`
- [ ] Download task queue with progress tracking

### Phase 4: Frontend UI
- [ ] `ui/src/pages/SeedboxPage.tsx` - Main page component
- [ ] `ui/src/components/SeedboxTorrent.tsx` - Add torrent form
- [ ] `ui/src/components/SeedboxBrowser.tsx` - File browser
- [ ] `ui/src/components/SeedboxDownloads.tsx` - Download progress
- [ ] `ui/src/components/SeedboxSettings.tsx` - Settings modal
- [ ] Add to sidebar, routes, translations

### Phase 5: Polish
- [ ] i18n translations (all locales)
- [ ] Error handling and retry logic
- [ ] Connection testing in settings
- [ ] Documentation

## Testing

### Local Testing with Docker

```bash
# rTorrent with ruTorrent
docker run -d --name rutorrent \
  -p 8080:8080 \
  -p 45000:45000 \
  crazymax/rtorrent-rutorrent

# Deluge
docker run -d --name deluge \
  -p 8112:8112 \
  linuxserver/deluge

# nginx for HTTP file serving (simulate seedbox HTTP)
docker run -d --name nginx-files \
  -p 8081:80 \
  -v /tmp/downloads:/usr/share/nginx/html:ro \
  nginx
```

### Test Files
Use legal test torrents (Ubuntu, Arch Linux ISOs, etc.)

## Security Considerations

- Credentials stored in config file (same as other sensitive data)
- HTTPS strongly recommended for HTTP file access
- SSH key authentication preferred over password for SFTP
- No auto-discovery to avoid network scanning

## Differences from NAS Mode

| Aspect | NAS Mode | Seedbox Mode |
|--------|----------|--------------|
| Location | Local network | Remote server |
| Dispatch | Yes | Yes |
| Browse files | No (use NAS UI) | Yes |
| Download back | No (already local) | Yes (HTTP/SFTP) |
| ISP visibility | N/A | Hidden (HTTP looks normal) |
| Speed | LAN speed | Internet speed |
| Use case | Home NAS | Remote seedbox |

## Future Enhancements (Not Planned)

- Automatic sync (watch folder)
- Webhook notifications on completion
- Multiple seedbox profiles
- Bandwidth scheduling
- Integration with Plex/Jellyfin for auto-scan
