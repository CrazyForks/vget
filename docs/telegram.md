# Telegram Support

Implementation plan for Telegram media download support in vget.

## Overview

vget aims to be an all-in-one media downloader. Telegram support is part of this vision, even though `tdl` (6k+ stars) exists as a dedicated tool.

**Approach**: Desktop session import only, using Telegram Desktop's API credentials.

## Technical Background

### How Telegram Auth Works

```
api_id + api_hash  =  identifies THE APP (vget)
user session       =  identifies THE USER's account
```

- Sessions are tied to the `api_id` they were created with
- Desktop session import reuses existing login from Telegram Desktop
- No phone/SMS verification needed if user has Desktop installed

### API Credentials

Two sets of credentials for different login methods:

```go
const (
    // vget's own registered credentials (for phone login)
    VgetAppID   = XXXXXXX  // TODO: fill in registered api_id
    VgetAppHash = "..."    // TODO: fill in registered api_hash

    // Telegram Desktop's credentials (for desktop session import)
    TelegramDesktopAppID   = 2040
    TelegramDesktopAppHash = "b18441a1ff607e10a989891a5462e627"
)
```

### Login Methods & Ban Risk

| Method | API Credentials | Ban Risk | Why |
|--------|-----------------|----------|-----|
| Phone + SMS | vget's own | **Zero** | Fresh session with registered app |
| Desktop import | Desktop's (2040) | Low | Reusing session, same app identity |

Recommended: Phone login for maximum safety.

## Dependencies

```go
github.com/gotd/td                    // Pure Go MTProto 2.0 implementation
github.com/gotd/td/session/tdesktop   // Desktop session import
```

## Implementation Plan

### Phase 1: MVP

#### 1. Session Management Commands

```bash
vget telegram login             # Interactive prompt (choose method)
vget telegram login --cli       # Phone + SMS directly (zero ban risk)
vget telegram login --desktop   # Import from Desktop directly (low ban risk)
vget telegram logout            # Clear stored session
vget telegram status            # Show login state
```

**Interactive prompt (default):**

```
? How would you like to login?

  > [1] Phone + SMS (recommended)
        Uses vget's registered API credentials
        Ban risk: None

    [2] Import from Telegram Desktop
        Requires Telegram Desktop installed and logged in
        Ban risk: Low (reuses existing session)
```

**Phone login flow (`--cli`):**
1. User enters phone number
2. Telegram sends verification code:
   - **Primary**: In-app message to existing Telegram sessions (Desktop/mobile)
   - **Fallback**: SMS (if no active sessions or user requests it)
3. User enters code
4. (Optional) Enters 2FA password if enabled
5. Session created with vget's API credentials

**Desktop import flow (`--desktop`):**
- Read Desktop's `tdata/` directory
  - macOS: `~/Library/Application Support/Telegram Desktop/tdata/`
  - Linux: `~/.local/share/TelegramDesktop/tdata/`
  - Windows: `%APPDATA%/Telegram Desktop/tdata/`
- Import session with Desktop's API credentials

**Session storage:** `~/.config/vget/telegram/`

#### 2. URL Parsing

Support these `t.me` formats:

| Format | Example | Type |
|--------|---------|------|
| Public channel | `https://t.me/channel/123` | Public |
| Private channel | `https://t.me/c/123456789/123` | Private |
| User/bot post | `https://t.me/username/123` | Public |
| Single from album | `https://t.me/channel/123?single` | Public |

#### 3. Single Message Download

```bash
vget https://t.me/somechannel/456
```

- Extract media (video/audio/document) from one message
- Download with progress bar (existing Bubbletea infrastructure)
- Save to current directory or `-o` path

#### 4. Media Type Detection

```go
MediaTypeVideo     // .mp4, .mov
MediaTypeAudio     // .mp3, .ogg voice messages
MediaTypeDocument  // .pdf, .zip, etc.
MediaTypePhoto     // .jpg (lower priority)
```

### Phase 2: Nice-to-Have

| Feature | Description |
|---------|-------------|
| Batch download | `vget https://t.me/channel/100-200` (range) |
| Resume | Continue interrupted downloads |
| Album support | Download all media from grouped messages |
| Channel dump | `vget https://t.me/channel --all` |

## File Structure

```
internal/extractor/
├── telegram.go          # Extractor implementation
├── telegram_auth.go     # Session import/management
├── telegram_parser.go   # URL parsing

internal/cli/
├── telegram.go          # login/logout/status commands
```

## vget vs tdl

| Aspect | tdl | vget |
|--------|-----|------|
| Scope | Telegram-only | Multi-platform |
| Features | Many advanced (batch, resume, takeout) | Simple: paste URL, get media |
| Philosophy | Power tool | All-in-one simplicity |

## Reference Implementation

The `tdl` project (github.com/iyear/tdl) was analyzed for patterns:

### Worth Borrowing

1. **URL Parsing** (`pkg/tmessage/parse.go`) - handles various t.me formats
2. **Media Extraction** (`core/tmedia/media.go`) - unified media type abstraction
3. **Middleware Pattern** - retry, recovery, flood-wait as composable layers

### Skip for MVP

- Iterator + Resume pattern (Phase 2)
- Data Center pooling (overkill for single downloads)
- Takeout mode (for bulk exports)

## References

- tdl source: https://github.com/iyear/tdl
- gotd/td (MTProto library): https://github.com/gotd/td
- Telegram Desktop session format: https://github.com/nickoala/tdesktop-session
