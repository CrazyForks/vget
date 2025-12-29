# Hyprnote Codebase Analysis

## Executive Summary

**Hyprnote** is an AI-powered meeting notetaking desktop application designed for private, offline-capable transcription and note summarization. Built as a sophisticated monorepo combining Rust backends, TypeScript/React frontends, and specialized ML services, it emphasizes privacy-first design with no bots joining meetings.

---

## Table of Contents

1. [Technology Stack](#technology-stack)
2. [Project Structure](#project-structure)
3. [Applications](#applications)
4. [Rust Crates (79 Total)](#rust-crates)
5. [TypeScript Packages](#typescript-packages)
6. [Tauri Plugins (35+)](#tauri-plugins)
7. [Database Architecture](#database-architecture)
8. [Audio Processing Pipeline](#audio-processing-pipeline)
9. [Transcription System](#transcription-system)
10. [LLM Integration](#llm-integration)
11. [Third-Party Integrations](#third-party-integrations)
12. [Build System & Tooling](#build-system--tooling)
13. [Security & Privacy](#security--privacy)
14. [Deployment Architecture](#deployment-architecture)
15. [Key Architectural Patterns](#key-architectural-patterns)

---

## Technology Stack

### Languages
| Language | Usage |
|----------|-------|
| **Rust** | Backend services, audio processing, ML inference, system integration |
| **TypeScript/JavaScript** | Frontend, APIs, tooling |
| **Python** | ML evaluation, data processing |
| **Go** | CLI tools and utilities |
| **Swift** | macOS native integration (audio interception) |

### Core Frameworks & Runtimes
| Technology | Purpose |
|------------|---------|
| **Tauri 2.9** | Desktop application framework (Rust + Web) |
| **React 19** | Frontend UI |
| **Vite** | Frontend bundler |
| **Axum** | Rust web framework |
| **TanStack Router** | Frontend routing |
| **TanStack Query/Form** | Data fetching and forms |
| **Hono** | Lightweight web framework (API, Pro) |
| **Bun** | JavaScript runtime (API server) |

### Databases & Storage
| Technology | Purpose |
|------------|---------|
| **Turso/LibSQL** | Local SQLite with encryption |
| **PostgreSQL** | Cloud database |
| **Drizzle ORM** | TypeScript database toolkit |
| **Supabase** | Backend as a service (auth, realtime, storage) |

### ML/AI Runtimes
| Technology | Purpose |
|------------|---------|
| **ONNX Runtime** | Cross-platform ML inference |
| **Llama.cpp** | Local LLM inference |
| **Whisper.cpp** | Local speech-to-text |
| **Silero VAD** | Voice activity detection |
| **Moonshine** | Lightweight STT model |

---

## Project Structure

```
hyprnote/
├── apps/                    # Main applications (7 total)
│   ├── desktop/            # Main Tauri desktop app
│   ├── web/                # TanStack Start web app
│   ├── api/                # Bun/Hono REST API
│   ├── ai/                 # Rust transcription/LLM service
│   ├── bot/                # GitHub Probot automation
│   ├── control/            # System tray Tauri app
│   └── pro/                # Cloudflare Workers MCP server
│
├── crates/                  # Rust libraries (79 total)
│   ├── audio/              # Audio capture & processing
│   ├── transcribe-*/       # Speech-to-text providers
│   ├── llm-*/              # LLM providers
│   ├── db-*/               # Database layers
│   ├── vad*/               # Voice activity detection
│   └── ...                 # Utilities, integrations
│
├── plugins/                 # Tauri plugins (35+ total)
│   ├── tauri-plugin-listener/
│   ├── tauri-plugin-local-stt/
│   ├── tauri-plugin-local-llm/
│   └── ...
│
├── packages/                # TypeScript packages (11 total)
│   ├── ui/                 # Shared UI components
│   ├── db/                 # Drizzle schemas
│   ├── api-client/         # Generated API client
│   ├── utils/              # Shared utilities
│   └── ...
│
├── extensions/              # User-facing extensions
│   ├── calendar/           # Apple Calendar integration
│   └── shared/             # Shared extension utilities
│
├── eval/                    # ML evaluation suite
├── data/                    # Data processing scripts
└── docs/                    # Documentation
```

---

## Applications

### 1. Desktop App (`apps/desktop`)

The primary application - a full-featured Tauri desktop app.

**Technology Stack:**
- Frontend: React 19 + TanStack Router + TypeScript
- Backend: Tauri 2 with Rust
- Bundler: Vite

**Key Features:**
- Real-time meeting transcription
- AI-powered note summarization
- Local-first with optional cloud sync
- Audio player with timeline visualization
- Search engine integration
- Tool registry system

**Source Structure:**
```
apps/desktop/
├── src/                    # React frontend
│   ├── components/        # UI components
│   ├── routes/            # TanStack Router routes
│   ├── contexts/          # React contexts
│   └── lib/               # Utilities
└── src-tauri/             # Rust backend
    ├── src/               # Main Tauri code
    └── Cargo.toml         # Dependencies
```

**Key Frontend Routes:**
- `/app/note/$noteId` - Note editor view
- `/app/new` - Create new session
- `/app/organization` - Organization management
- `/app/settings` - User settings
- `/app/extensions` - Extensions management

### 2. Web App (`apps/web`)

Cloud-based dashboard and management interface.

**Technology Stack:**
- Framework: TanStack Start (React full-stack SSR)
- Styling: Tailwind CSS
- Deployment: Netlify

**Key Features:**
- Dashboard for meetings management
- Integration management (OAuth flows)
- Template library
- AI chat interface
- Analytics integration (PostHog)
- Search (Orama)

### 3. API Server (`apps/api`)

RESTful API for cloud services.

**Technology Stack:**
- Runtime: Bun
- Framework: Hono
- Database: Supabase (PostgreSQL)

**Key Features:**
- Webhook handling
- Stripe billing integration
- OpenAI integration
- User/organization management

**Endpoints Structure:**
```typescript
// Key routes from apps/api/src/routes/
├── integrations/          # Third-party integrations
├── organizations/         # Organization management
├── stripe/               # Billing webhooks
├── templates/            # Note templates
└── users/                # User management
```

### 4. AI Service (`apps/ai`)

Real-time transcription and LLM proxy service.

**Technology Stack:**
- Language: Rust
- Framework: Axum + Tokio
- Protocol: HTTP + WebSocket

**Key Features:**
- Multi-provider transcription routing
- LLM request proxying
- Real-time streaming support
- Rate limiting and caching

**Architecture:**
```rust
// apps/ai/src/
├── main.rs               # Axum server setup
├── transcribe.rs         # Transcription proxy logic
├── llm.rs                # LLM proxy logic
└── ws.rs                 # WebSocket handlers
```

### 5. Control Panel (`apps/control`)

Lightweight system tray application.

**Technology Stack:**
- Framework: Tauri 2 (minimal)
- Purpose: Background service management

### 6. Pro Service (`apps/pro`)

SaaS and MCP (Model Context Protocol) server.

**Technology Stack:**
- Runtime: Cloudflare Workers
- Framework: Hono
- Features: Exa search integration, caching

### 7. GitHub Bot (`apps/bot`)

Repository automation.

**Technology Stack:**
- Framework: Probot
- Language: TypeScript

---

## Rust Crates

### Audio Processing & Capture

| Crate | Purpose |
|-------|---------|
| `audio` | Audio input/output abstraction using CPAL and Rodio |
| `audio-interface` | Platform-agnostic audio API trait |
| `audio-utils` | Audio format conversion, resampling |
| `audio-priority` | Audio stream prioritization |
| `intercept` | Audio interception (Swift integration for macOS system audio) |
| `aec` | Acoustic Echo Cancellation (ONNX-based) |
| `agc` | Automatic Gain Control |

### Voice Activity Detection (VAD)

| Crate | Purpose |
|-------|---------|
| `vad-ext` | External VAD provider integration (Silero) |
| `vad2` | VAD implementation v2 |
| `vad3` | VAD implementation v3 |
| `vvad` | Vector-based VAD |

### Speech-to-Text Providers

| Crate | Provider | Type |
|-------|----------|------|
| `transcribe-interface` | N/A | Provider trait definition |
| `transcribe-proxy` | All | Unified routing proxy |
| `transcribe-openai` | OpenAI | Cloud (Whisper API) |
| `transcribe-aws` | AWS | Cloud (Amazon Transcribe) |
| `transcribe-azure` | Azure | Cloud (Speech Services) |
| `transcribe-deepgram` | Deepgram | Cloud |
| `transcribe-gcp` | Google | Cloud (Speech-to-Text) |
| `transcribe-moonshine` | Moonshine | Local (ONNX) |
| `transcribe-whisper-local` | Whisper.cpp | Local |
| `whisper` | N/A | Whisper model bindings |
| `whisper-local` | N/A | Local Whisper inference |
| `whisper-local-model` | N/A | Model management |
| `moonshine` | N/A | Moonshine model bindings |
| `pyannote-cloud` | Pyannote | Cloud (speaker diarization) |
| `pyannote-local` | Pyannote | Local (speaker diarization) |

### LLM Integration

| Crate | Purpose |
|-------|---------|
| `llm-interface` | LLM provider trait definition |
| `llm-proxy` | Unified LLM routing proxy |
| `llm` | Local LLM orchestration |
| `llama` | Llama.cpp integration via llama-cpp-rs |
| `gguf` | GGUF model format support |
| `onnx` | ONNX runtime for ML models |
| `gbnf` | Grammar-based structured output |
| `kyutai` | Kyutai model integration |

### Database & Persistence

| Crate | Purpose |
|-------|---------|
| `db-core` | Core database layer (LibSQL/Turso) |
| `db-user` | User database models and queries |
| `data` | Data structures and serialization |
| `buffer` | Audio/data buffer management |
| `file` | File operations abstraction |

### Third-Party Integrations

| Crate | Purpose |
|-------|---------|
| `nango` | OAuth integration platform |
| `turso` | Turso database client |
| `s3` | AWS S3 storage |
| `supabase-auth` | Supabase authentication |
| `openai` | OpenAI API wrapper |
| `lago` | Lago billing platform |

### Open Whisper Protocol

| Crate | Purpose |
|-------|---------|
| `owhisper-client` | Protocol client |
| `owhisper-interface` | Protocol definition |
| `owhisper-providers` | Provider implementations |
| `owhisper-config` | Configuration |

### System Integration

| Crate | Purpose |
|-------|---------|
| `detect` | Platform detection (OS, capabilities) |
| `device-heuristic` | Device capability detection |
| `device-monitor` | Hardware monitoring |
| `host` | Host identification (MAC, machine ID) |
| `tcc` | macOS Transparency Consent Control |
| `mac` | macOS-specific utilities |
| `am` / `am2` | Account management |

### Notifications

| Crate | Purpose |
|-------|---------|
| `notification-interface` | Notification protocol |
| `notification` | Cross-platform notifications |
| `notification-macos` | macOS native notifications |
| `notification-linux` | Linux notifications |
| `notification-gpui` | GPUI framework notifications |

### Template & Evaluation

| Crate | Purpose |
|-------|---------|
| `template-app` | Note template rendering (Askama) |
| `template-app-legacy` | Legacy template system |
| `template-eval` | Template evaluation |
| `eval` | LLM evaluation runner |
| `granola` | Model/tool management |

### Utilities

| Crate | Purpose |
|-------|---------|
| `analytics` | PostHog analytics integration |
| `askama-utils` | Askama template utilities |
| `ws-client` | WebSocket client |
| `ws-utils` | WebSocket utilities |
| `language` | Language detection |
| `loops` | Task orchestration |
| `docs` | Documentation generation |
| `extensions-runtime` | Plugin/extension runtime |
| `download-interface` | Download protocol |

---

## TypeScript Packages

| Package | Path | Purpose |
|---------|------|---------|
| `@hypr/ui` | `packages/ui` | Shared UI components (Radix UI, Tailwind, Shadcn-style) |
| `@hypr/db` | `packages/db` | Database schemas and migrations (Drizzle ORM) |
| `@hypr/api-client` | `packages/api-client` | Auto-generated API client (OpenAPI) |
| `@hypr/utils` | `packages/utils` | Shared utilities, hooks, helpers |
| `@hypr/tiptap` | `packages/tiptap` | Rich text editor (TipTap/ProseMirror) |
| `@hypr/codemirror` | `packages/codemirror` | Code editor integration |
| `@hypr/store` | `packages/store` | Global state management |
| `@hypr/supabase` | `packages/supabase` | Supabase client wrapper |
| `@hypr/obsidian` | `packages/obsidian` | Obsidian plugin |
| `calendar` | `extensions/calendar` | Apple Calendar integration UI |
| `shared` | `extensions/shared` | Shared extension utilities |

### UI Package (`packages/ui`)

Comprehensive component library:

```
packages/ui/
├── components/
│   ├── button.tsx
│   ├── dialog.tsx
│   ├── dropdown-menu.tsx
│   ├── input.tsx
│   ├── select.tsx
│   ├── toast.tsx
│   └── ... (50+ components)
├── hooks/
│   ├── use-mobile.tsx
│   └── use-toast.ts
└── lib/
    └── utils.ts
```

### Database Package (`packages/db`)

Drizzle ORM schemas and migrations:

```
packages/db/
├── schemas/
│   ├── users.ts
│   ├── organizations.ts
│   ├── sessions.ts
│   ├── transcripts.ts
│   └── ...
├── migrations/
│   └── *.sql
└── index.ts
```

---

## Tauri Plugins

### Core Functionality

| Plugin | Purpose |
|--------|---------|
| `tauri-plugin-listener` / `listener2` | Real-time event listening, audio streaming |
| `tauri-plugin-local-stt` | Local speech-to-text (Whisper, Moonshine) |
| `tauri-plugin-local-llm` | Local LLM inference (Llama.cpp) |
| `tauri-plugin-db2` | Database access layer |

### System Integration

| Plugin | Purpose |
|--------|---------|
| `tauri-plugin-apple-calendar` | Apple Calendar sync |
| `tauri-plugin-notification` | System notifications |
| `tauri-plugin-tray` | System tray icon |
| `tauri-plugin-audio-priority` | Audio stream prioritization |
| `tauri-plugin-detect` | Platform/device detection |
| `tauri-plugin-permissions` | Permission management |
| `tauri-plugin-windows` | Window management |

### Settings & Storage

| Plugin | Purpose |
|--------|---------|
| `tauri-plugin-settings` | User settings |
| `tauri-plugin-store2` | Persistent key-value storage |
| `tauri-plugin-template` | Template management |
| `tauri-plugin-icon` | Icon management |

### Data & Communication

| Plugin | Purpose |
|--------|---------|
| `tauri-plugin-export` | Export functionality (PDF, Markdown, etc.) |
| `tauri-plugin-importer` | Data importing |
| `tauri-plugin-pdf` | PDF handling |
| `tauri-plugin-webhook` | Webhook handling |
| `tauri-plugin-network` | Network status monitoring |

### Developer & Analytics

| Plugin | Purpose |
|--------|---------|
| `tauri-plugin-analytics` | PostHog event analytics |
| `tauri-plugin-cli2` / `cli` | CLI integration |
| `tauri-plugin-hooks` | Lifecycle hooks |
| `tauri-plugin-tracing` | Distributed tracing |
| `tauri-plugin-misc` | Miscellaneous utilities |
| `tauri-plugin-extensions` | Extension system |
| `tauri-plugin-auth` | Authentication |
| `tauri-plugin-deeplink2` | Deep linking |
| `tauri-plugin-path2` | Path utilities |
| `tauri-plugin-updater2` | App updates |
| `tauri-plugin-sfx` | Sound effects |

---

## Database Architecture

### Local Database (Turso/LibSQL)

Local-first storage with optional encryption:

```sql
-- Core tables
sessions          -- Meeting sessions
transcripts       -- Transcription data
notes             -- User notes
templates         -- Note templates
settings          -- User settings
audio_files       -- Audio file metadata
```

### Cloud Database (Supabase/PostgreSQL)

Cloud sync and collaboration:

```sql
-- Drizzle schemas (packages/db)
users             -- User accounts
organizations     -- Teams/workspaces
memberships       -- User-org relationships
sessions          -- Synced sessions
integrations      -- Third-party OAuth tokens
subscriptions     -- Billing subscriptions
```

### Migration System

Drizzle migrations in `packages/db/migrations/`:
- Supports both local (LibSQL) and cloud (PostgreSQL)
- Versioned schema changes
- Type-safe query building

---

## Audio Processing Pipeline

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Audio     │───▶│    AEC      │───▶│    AGC      │
│   Capture   │    │   (Echo     │    │   (Gain     │
│   (CPAL)    │    │   Cancel)   │    │   Control)  │
└─────────────┘    └─────────────┘    └─────────────┘
                                              │
                                              ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Transcribe  │◀───│   Buffer    │◀───│    VAD      │
│   Proxy     │    │  Management │    │   (Silero)  │
└─────────────┘    └─────────────┘    └─────────────┘
```

### Audio Capture Sources

1. **Microphone** - Direct audio input via CPAL
2. **System Audio** - macOS screen capture API (via Swift `intercept` crate)
3. **Mixed** - Combined microphone + system audio

### Processing Chain

1. **Acoustic Echo Cancellation (AEC)** - Removes speaker feedback
2. **Automatic Gain Control (AGC)** - Normalizes audio levels
3. **Voice Activity Detection (VAD)** - Filters silence, segments speech
4. **Buffer Management** - Chunks audio for transcription

---

## Transcription System

### Provider Interface

```rust
// crates/transcribe-interface/src/lib.rs
#[async_trait]
pub trait TranscribeProvider: Send + Sync {
    async fn transcribe(&self, audio: AudioData) -> Result<Transcript>;
    fn supports_streaming(&self) -> bool;
}
```

### Supported Providers

| Provider | Type | Streaming | Diarization |
|----------|------|-----------|-------------|
| OpenAI Whisper | Cloud | No | No |
| AWS Transcribe | Cloud | Yes | Yes |
| Azure Speech | Cloud | Yes | Yes |
| Google Cloud | Cloud | Yes | Yes |
| Deepgram | Cloud | Yes | Yes |
| Whisper.cpp | Local | No | No |
| Moonshine | Local | No | No |

### Proxy Architecture

```rust
// crates/transcribe-proxy/src/lib.rs
pub struct TranscribeProxy {
    providers: HashMap<String, Box<dyn TranscribeProvider>>,
    default_provider: String,
}

impl TranscribeProxy {
    pub async fn transcribe(&self, request: Request) -> Result<Transcript> {
        let provider = self.select_provider(&request)?;
        provider.transcribe(request.audio).await
    }
}
```

---

## LLM Integration

### Provider Interface

```rust
// crates/llm-interface/src/lib.rs
#[async_trait]
pub trait LlmProvider: Send + Sync {
    async fn complete(&self, prompt: &str) -> Result<String>;
    async fn stream(&self, prompt: &str) -> Result<impl Stream<Item = String>>;
}
```

### Supported Providers

| Provider | Type | Models |
|----------|------|--------|
| OpenAI | Cloud | GPT-4, GPT-3.5 |
| Anthropic | Cloud | Claude 3.x |
| Llama.cpp | Local | Llama 2/3, Mistral, etc. |
| ONNX | Local | Custom ONNX models |

### Local LLM Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   LLM       │───▶│   llama     │───▶│  llama-cpp  │
│   Proxy     │    │   crate     │    │    (C++)    │
└─────────────┘    └─────────────┘    └─────────────┘
                          │
                          ▼
                   ┌─────────────┐
                   │    GGUF     │
                   │   Models    │
                   └─────────────┘
```

### Template System

Note summarization uses Askama templates:

```rust
// crates/template-app/templates/
├── summary.askama       # Meeting summary
├── action_items.askama  # Action item extraction
├── highlights.askama    # Key highlights
└── custom/              # User templates
```

---

## Third-Party Integrations

### OAuth via Nango

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Hyprnote  │───▶│   Nango     │───▶│  Third-Party│
│   (Client)  │    │   (OAuth)   │    │   Services  │
└─────────────┘    └─────────────┘    └─────────────┘
```

### Supported Integrations

| Service | Type | Status |
|---------|------|--------|
| Apple Calendar | Native | Active |
| Obsidian | Plugin | Active |
| Google Calendar | OAuth | Planned |
| Notion | OAuth | Planned |
| Slack | OAuth | Planned |
| HubSpot | OAuth | Planned |
| Salesforce | OAuth | Planned |

### Billing via Stripe

```typescript
// apps/api/src/routes/stripe.ts
- Subscription management
- Usage-based billing
- Webhook handling
- Customer portal
```

### Analytics via PostHog

```rust
// crates/analytics/src/lib.rs
- Event tracking
- Feature flags
- User identification
- Session recording
```

---

## Build System & Tooling

### Monorepo Orchestration

| Tool | Purpose |
|------|---------|
| **Turbo** | Task orchestration, caching |
| **Cargo** | Rust package management |
| **pnpm** | JavaScript package management |
| **Taskfile** | Development automation |

### Configuration Files

| File | Purpose |
|------|---------|
| `Cargo.toml` | Rust workspace (79+ members) |
| `pnpm-workspace.yaml` | npm/pnpm workspace |
| `turbo.json` | Turbo task definitions |
| `dprint.json` | Code formatting |
| `Taskfile.yaml` | Development tasks |
| `tsconfig.json` | TypeScript configuration |

### Key Commands

```bash
# Development
pnpm dev                    # Start desktop app dev
pnpm -r typecheck          # TypeScript type checking
cargo check                 # Rust compilation check
dprint fmt                  # Code formatting

# Building
pnpm build                  # Build all packages
cargo build --release       # Rust release build
pnpm tauri build           # Desktop app build

# Testing
cargo test                  # Rust tests
pnpm test                   # JavaScript tests
```

### Version Requirements

- Rust Edition 2024
- TypeScript 5.8+
- Node.js 22+
- pnpm 10.26.0+
- Tauri 2.9

---

## Security & Privacy

### Privacy-First Design

- **No meeting bots** - Captures local audio only
- **Local-first storage** - Data stays on device by default
- **Encrypted database** - LibSQL encryption for local storage
- **Minimal cloud sync** - Optional, user-controlled

### Authentication

| Method | Provider |
|--------|----------|
| Email/Password | Supabase Auth |
| OAuth (Google, GitHub) | Supabase Auth |
| MFA | Supabase Auth |

### macOS Privacy Controls

```rust
// crates/tcc/src/lib.rs
- Microphone access (TCC)
- Screen recording permission
- Accessibility access
- Full disk access (optional)
```

### Security Measures

- HTTPS/TLS for all network communication
- OAuth 2.0 for third-party integrations
- Encrypted local database
- No plaintext credential storage

---

## Deployment Architecture

### Docker Containers

```yaml
# Docker deployments
apps/api/Dockerfile        # API server (Bun)
apps/ai/Dockerfile         # AI service (Rust)
apps/bot/Dockerfile        # GitHub bot (Node)
apps/pro/Dockerfile        # Pro service (Workers)
```

### Cloud Infrastructure

| Service | Provider | Purpose |
|---------|----------|---------|
| Database | Supabase | PostgreSQL + Auth |
| Storage | AWS S3 | File storage |
| CDN | Cloudflare | Edge caching |
| Analytics | PostHog | Product analytics |
| Errors | Sentry | Error tracking |
| Billing | Stripe | Payments |
| Workflows | Restate | Orchestration |

### Desktop Distribution

- macOS: DMG, App Store (planned)
- Windows: NSIS installer (planned)
- Linux: AppImage, deb, rpm (planned)

---

## Key Architectural Patterns

### 1. Plugin Architecture

Extensible via Tauri plugins:
- Each plugin is a standalone Rust crate
- Plugins expose commands to frontend via Specta
- Type-safe TypeScript bindings auto-generated

### 2. Proxy Pattern

Unified interfaces for multi-provider support:
- `transcribe-proxy` routes to 7+ STT providers
- `llm-proxy` routes to local and cloud LLMs
- `notification` abstracts platform-specific notifications

### 3. Trait-Based Abstraction

Rust traits define provider interfaces:
```rust
trait TranscribeProvider { ... }
trait LlmProvider { ... }
trait NotificationProvider { ... }
```

### 4. Event-Driven Architecture

Real-time updates via:
- Tauri event system
- WebSocket connections
- Supabase Realtime

### 5. Type-Safe Codegen

Specta generates TypeScript types from Rust:
- Plugin commands → TypeScript functions
- Rust structs → TypeScript interfaces
- Compile-time type safety

### 6. Async/Await

Tokio-based async runtime:
- Non-blocking I/O
- Concurrent audio processing
- Parallel API requests

---

## Project Statistics

| Metric | Count |
|--------|-------|
| Rust Crates | 79 |
| Tauri Plugins | 35+ |
| TypeScript Packages | 11 |
| Main Applications | 7 |
| Transcription Providers | 7+ |
| LLM Providers | 4+ |
| Lines of Rust | ~100,000+ |
| Lines of TypeScript | ~50,000+ |

---

## Key Files Reference

### Configuration
- `Cargo.toml` - Rust workspace
- `package.json` - Root npm config
- `turbo.json` - Build orchestration
- `Taskfile.yaml` - Development tasks
- `dprint.json` - Formatting rules

### Entry Points
- `apps/desktop/src/main.tsx` - Desktop frontend
- `apps/desktop/src-tauri/src/main.rs` - Desktop backend
- `apps/api/src/index.ts` - API server
- `apps/ai/src/main.rs` - AI service
- `apps/web/app/routes.ts` - Web routes

### Core Logic
- `crates/audio/src/lib.rs` - Audio capture
- `crates/transcribe-proxy/src/lib.rs` - STT routing
- `crates/llm-proxy/src/lib.rs` - LLM routing
- `plugins/tauri-plugin-listener/src/lib.rs` - Event system

---

## Summary

Hyprnote is a sophisticated, production-grade AI meeting assistant built with:

1. **Privacy-first architecture** - Local processing, encrypted storage
2. **Multi-provider flexibility** - 7+ STT providers, 4+ LLM providers
3. **Cross-platform foundation** - Tauri for desktop, React for web
4. **Modular design** - 79 Rust crates, 35+ plugins
5. **Type-safe integration** - Rust to TypeScript codegen via Specta
6. **Enterprise-ready** - OAuth, billing, analytics, error tracking

The codebase demonstrates excellent separation of concerns, with clear boundaries between audio processing, ML inference, UI rendering, and system integration layers.
