# Handy Codebase Analysis

## Executive Summary

**Handy** is a free, open-source, privacy-first desktop speech-to-text application built with **Tauri 2.x** (Rust backend + React/TypeScript frontend). Users press a keyboard shortcut, speak, and have transcribed text automatically inserted into any text field - all processed locally without cloud services.

- **Version:** 0.6.9
- **License:** MIT
- **Author:** cjpais
- **Platforms:** macOS, Windows, Linux

---

## Technology Stack

### Frontend

| Technology | Version | Purpose |
|------------|---------|---------|
| React | 18.3.1 | UI framework (functional components + hooks) |
| TypeScript | 5.6.3 | Type safety (strict mode enabled) |
| Vite | 6.4.1 | Build tool and dev server |
| Tailwind CSS | 4.1.16 | Utility-first styling |
| Zustand | 5.0.8 | State management |
| i18next | 25.7.2 | Internationalization (10 languages) |
| Tauri API | 2.9.0 | Backend communication |
| Lucide React | 0.542.0 | Icon library |
| Sonner | 2.0.7 | Toast notifications |
| React-Select | 5.8.0 | Dropdown components |
| Zod | 3.25.76 | Schema validation |

### Backend (Rust)

| Crate | Version | Purpose |
|-------|---------|---------|
| tauri | 2.9.1 | Desktop application framework |
| tokio | 1.43.0 | Async runtime |
| specta / tauri-specta | 2.0.0-rc | Auto-generated TypeScript bindings |
| rusqlite | 0.37 | SQLite database |
| serde | 1.0 | Serialization/deserialization |
| cpal | 0.16.0 | Cross-platform audio I/O |
| hound | 3.5.1 | WAV file handling |
| rubato | 0.16.2 | Audio resampling to 16kHz |
| vad-rs | git | Voice Activity Detection (Silero model) |
| transcribe-rs | 0.1.4 | Whisper/Parakeet inference |
| rdev | git | Global keyboard shortcuts |
| enigo | 0.6.1 | Keyboard/mouse input simulation |
| reqwest | 0.12 | HTTP client for LLM APIs |
| chrono | 0.4 | Date/time handling |

### Tauri Plugins

- `tauri-plugin-store` - JSON-based persistent settings
- `tauri-plugin-fs` - File system access
- `tauri-plugin-process` - Process management
- `tauri-plugin-clipboard-manager` - Clipboard operations
- `tauri-plugin-global-shortcut` - Global hotkeys
- `tauri-plugin-updater` - Auto-update from GitHub releases
- `tauri-plugin-autostart` - Launch on system startup
- `tauri-plugin-log` - File/console logging with rotation
- `tauri-plugin-opener` - Open URLs/files
- `tauri-plugin-os` - OS info detection
- `tauri-plugin-macos-permissions` - macOS permission APIs
- `tauri-plugin-single-instance` - Prevent duplicate app instances
- `tauri-nspanel` (macOS only) - Panel window handling

---

## Architecture Overview

### High-Level Data Flow

```
+------------------------------------------------------------------+
|                         Frontend (React)                          |
|  +--------------+  +--------------+  +----------------------+    |
|  |   Zustand    |  |  Components  |  |   i18n Translations  |    |
|  |    Store     |<-|  (Settings,  |  |   (10 languages)     |    |
|  |              |  |  Onboarding) |  |                      |    |
|  +------+-------+  +--------------+  +----------------------+    |
|         |                                                         |
|         | Tauri Commands (invoke)                                 |
|         v                                                         |
+-------------------------------------------------------------------+
|                    bindings.ts (Auto-generated)                   |
+-------------------------------------------------------------------+
|                         Backend (Rust)                            |
|  +------------+  +------------+  +------------+  +-----------+   |
|  |   Audio    |  |   Model    |  |Transcription| |  History  |   |
|  |  Manager   |  |  Manager   |  |  Manager    | |  Manager  |   |
|  +-----+------+  +-----+------+  +------+------+ +-----+-----+   |
|        |               |                |              |          |
|        v               v                v              v          |
|  +------------------------------------------------------------+  |
|  |              Audio Toolkit (Low-level Processing)          |  |
|  |  - Device Enumeration  - Recording  - VAD (Silero)         |  |
|  |  - Resampling (16kHz)  - Visualizer  - Custom Words        |  |
|  +------------------------------------------------------------+  |
|        |                                                          |
|        v                                                          |
|  +------------------------------------------------------------+  |
|  |                transcribe-rs (ML Inference)                 |  |
|  |     - OpenAI Whisper (Small/Medium/Turbo/Large)            |  |
|  |     - NVIDIA Parakeet V3                                    |  |
|  +------------------------------------------------------------+  |
+-------------------------------------------------------------------+
```

### Manager Pattern

The backend organizes core functionality into four managers, each handling a specific domain:

```rust
// Initialized at startup and managed via Tauri state
let recording_manager = Arc::new(AudioRecordingManager::new(app_handle)?);
let model_manager = Arc::new(ModelManager::new(app_handle)?);
let transcription_manager = Arc::new(TranscriptionManager::new(app_handle, model_manager.clone())?);
let history_manager = Arc::new(HistoryManager::new(app_handle)?);
```

#### 1. AudioRecordingManager (`managers/audio.rs`)

Handles all audio capture:

- **Microphone modes:** AlwaysOn (stream stays open) vs OnDemand (opens per recording)
- **Voice Activity Detection:** Silero VAD with smoothing (15 frames pre-fill, 15 hangover)
- **Audio levels:** Real-time spectrum visualization callback to frontend
- **System mute:** Mutes system audio during recording (Windows/macOS/Linux)
- **Clamshell mode:** Alternate microphone selection when laptop lid is closed

```rust
pub enum MicrophoneMode {
    AlwaysOn,   // Stream stays open for faster start
    OnDemand,   // Opens stream when needed, saves resources
}

pub enum RecordingState {
    Idle,
    Recording { binding_id: String },
}
```

#### 2. ModelManager (`managers/model.rs`)

Manages speech-to-text models:

- Downloads models from `blob.handy.computer`
- Tracks download progress via events
- Stores models in app data directory
- Supports two engine types: Whisper and Parakeet

Available models:

| Model | Engine | Size | Accuracy | Speed |
|-------|--------|------|----------|-------|
| Small | Whisper | 487 MB | 3/5 | 5/5 |
| Medium | Whisper | 1.5 GB | 4/5 | 4/5 |
| Turbo | Whisper | 1.6 GB | 4/5 | 5/5 |
| Large | Whisper | 3.0 GB | 5/5 | 2/5 |
| Parakeet V3 | Parakeet | ~600 MB | 5/5 | 5/5 |

#### 3. TranscriptionManager (`managers/transcription.rs`)

Manages the ML inference lifecycle:

- **Lazy loading:** Models loaded on first transcription
- **Idle unloading:** Configurable timeout (Never/Immediately/2-60 min/1 hour)
- **Background watcher:** Thread monitors inactivity every 10 seconds
- **Engine abstraction:** Supports both Whisper and Parakeet engines
- **Custom words:** Post-transcription word correction with similarity matching

```rust
enum LoadedEngine {
    Whisper(WhisperEngine),
    Parakeet(ParakeetEngine),
}

pub struct TranscriptionManager {
    engine: Arc<Mutex<Option<LoadedEngine>>>,
    model_manager: Arc<ModelManager>,
    last_activity: Arc<AtomicU64>,        // Unix timestamp in ms
    shutdown_signal: Arc<AtomicBool>,     // For graceful shutdown
    is_loading: Arc<Mutex<bool>>,
    loading_condvar: Arc<Condvar>,        // Wait for async load
}
```

#### 4. HistoryManager (`managers/history.rs`)

SQLite-based transcription history:

```sql
CREATE TABLE transcription_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_name TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    saved BOOLEAN NOT NULL DEFAULT 0,
    title TEXT NOT NULL,
    transcription_text TEXT NOT NULL,
    post_processed_text TEXT,
    post_process_prompt TEXT
);
```

- Configurable history limit (max entries)
- Recording retention periods (Never/3 days/2 weeks/3 months)
- Migration from legacy tauri-plugin-sql system

---

## Backend Directory Structure

```
src-tauri/src/
├── lib.rs                  # Entry point, Tauri setup, plugin initialization
├── main.rs                 # Windows subsystem configuration
├── managers/
│   ├── mod.rs
│   ├── audio.rs           # AudioRecordingManager (445 lines)
│   ├── model.rs           # ModelManager
│   ├── transcription.rs   # TranscriptionManager (450 lines)
│   └── history.rs         # HistoryManager
├── audio_toolkit/
│   ├── mod.rs
│   ├── audio/
│   │   ├── mod.rs
│   │   ├── device.rs      # Device enumeration (cpal)
│   │   ├── recorder.rs    # AudioRecorder with VAD
│   │   ├── resampler.rs   # Convert to 16kHz
│   │   ├── visualizer.rs  # Spectrum/level computation
│   │   └── utils.rs       # WAV file saving
│   ├── vad/
│   │   ├── mod.rs
│   │   ├── silero.rs      # Silero VAD (ONNX model)
│   │   └── smoothed.rs    # SmoothedVad wrapper
│   ├── text.rs            # Custom word replacement
│   └── constants.rs       # Audio constants
├── commands/
│   ├── mod.rs             # General commands
│   ├── models.rs          # Model operations
│   ├── audio.rs           # Audio device commands
│   ├── transcription.rs   # Transcription state
│   └── history.rs         # History operations
├── settings.rs            # AppSettings, 22KB, 30+ configuration fields
├── shortcut.rs            # Global keyboard shortcuts (835 lines)
├── actions.rs             # TranscribeAction pipeline (18KB)
├── tray.rs                # System tray icon/menu
├── tray_i18n.rs           # Tray menu translations
├── overlay.rs             # Recording overlay window
├── clipboard.rs           # Cross-platform clipboard
├── input.rs               # Enigo keyboard/mouse state
├── audio_feedback.rs      # Start/stop sound notifications
├── llm_client.rs          # HTTP client for LLM post-processing
├── signal_handle.rs       # Unix signal handling (SIGUSR2)
├── apple_intelligence.rs  # macOS Apple Intelligence integration
├── helpers/
│   └── clamshell.rs       # Laptop lid detection
└── utils.rs               # Utility functions
```

---

## Frontend Directory Structure

```
src/
├── main.tsx               # React entry point, i18n init
├── App.tsx                # Main component, routing
├── App.css                # Global styles
├── bindings.ts            # Auto-generated Tauri types (27KB)
├── vite-env.d.ts          # Vite type definitions
├── stores/
│   └── settingsStore.ts   # Zustand store (500 lines)
├── hooks/
│   ├── useSettings.ts     # Settings hook
│   └── useModels.ts       # Model management hook
├── i18n/
│   ├── index.ts           # i18n setup
│   ├── languages.ts       # Language metadata
│   └── locales/
│       ├── en/translation.json  # English (source)
│       ├── es/translation.json  # Spanish
│       ├── fr/translation.json  # French
│       ├── de/translation.json  # German
│       ├── it/translation.json  # Italian
│       ├── ja/translation.json  # Japanese
│       ├── pl/translation.json  # Polish
│       ├── ru/translation.json  # Russian
│       ├── vi/translation.json  # Vietnamese
│       └── zh/translation.json  # Chinese
├── components/
│   ├── Sidebar.tsx        # Navigation sidebar
│   ├── AccessibilityPermissions.tsx
│   ├── settings/          # 27+ settings components
│   │   ├── GeneralSettings.tsx
│   │   ├── AdvancedSettings.tsx
│   │   ├── HistorySettings.tsx
│   │   ├── DebugSettings.tsx
│   │   ├── PostProcessingSettings.tsx
│   │   ├── AboutSettings.tsx
│   │   ├── MicrophoneSelector.tsx
│   │   ├── OutputDeviceSelector.tsx
│   │   ├── HandyShortcut.tsx
│   │   ├── CustomWords.tsx
│   │   ├── SoundPicker.tsx
│   │   └── ... (many more)
│   ├── model-selector/
│   │   ├── ModelSelector.tsx
│   │   ├── ModelDropdown.tsx
│   │   ├── ModelCard.tsx
│   │   ├── DownloadProgressDisplay.tsx
│   │   └── ModelStatusButton.tsx
│   ├── onboarding/
│   │   ├── Onboarding.tsx
│   │   └── ModelCard.tsx
│   ├── ui/                # Reusable UI components
│   │   ├── Button.tsx
│   │   ├── Input.tsx
│   │   ├── Textarea.tsx
│   │   ├── ToggleSwitch.tsx
│   │   ├── Slider.tsx
│   │   ├── Select.tsx
│   │   ├── Dropdown.tsx
│   │   ├── SettingContainer.tsx
│   │   ├── SettingsGroup.tsx
│   │   └── ...
│   ├── shared/
│   │   └── ProgressBar.tsx
│   ├── footer/
│   │   └── Footer.tsx
│   └── update-checker/
├── overlay/               # Recording overlay (separate window)
│   ├── index.html
│   ├── main.tsx
│   ├── RecordingOverlay.tsx
│   └── RecordingOverlay.css
```

---

## Key Features

### 1. Transcription Pipeline

The complete flow from button press to text insertion:

```
User presses shortcut
         |
         v
+---------------------+
| Show Recording      |
| Overlay (waveform)  |
+---------------------+
| Play start sound    |
+---------------------+
| Start audio capture |
| (16kHz, mono)       |
+---------------------+
| Apply system mute   |
| (if enabled)        |
+---------+-----------+
          |
          v (user speaks)
          |
+---------------------+
| VAD filters silence |
| (Silero + smoothing)|
+---------------------+
| Detect silence      |
| timeout -> stop     |
+---------+-----------+
          |
          v
+---------------------+
| Transcribe audio    |
| (Whisper/Parakeet)  |
+---------------------+
| Apply custom words  |
| (similarity match)  |
+---------------------+
| Post-process (LLM)  |
| [optional]          |
+---------------------+
| Chinese conversion  |
| (Simplified<->Trad) |
+---------+-----------+
          |
          v
+---------------------+
| Copy to clipboard   |
| (if enabled)        |
+---------------------+
| Paste into active   |
| application         |
+---------------------+
| Save to history     |
| (SQLite)            |
+---------------------+
| Remove system mute  |
+---------------------+
| Play stop sound     |
+---------------------+
```

### 2. Voice Activity Detection (VAD)

The SmoothedVad wrapper reduces false positives:

```rust
pub struct SmoothedVad {
    inner: Box<dyn Vad>,
    pre_fill_frames: usize,   // Frames of speech before activating (15)
    hangover_frames: usize,   // Frames of silence before deactivating (15)
    min_speech_frames: usize, // Minimum speech duration (2)
}
```

### 3. Custom Word Correction

Post-transcription word replacement with fuzzy matching:

```rust
pub fn apply_custom_words(
    text: &str,
    custom_words: &[String],
    threshold: f64,  // Default 0.18, lower = stricter matching
) -> String
```

Uses Jaro-Winkler similarity from the `strsim` crate.

### 4. Post-Processing (LLM Integration)

Optional LLM post-processing to fix grammar, formatting, or apply custom prompts:

**Supported Providers:**
- Apple Intelligence (macOS arm64 only)
- OpenAI API
- OpenAI-compatible APIs (custom base URL)

```rust
pub struct PostProcessProvider {
    pub id: String,
    pub name: String,
    pub description: String,
    pub base_url: Option<String>,
    pub api_key: Option<String>,
    pub model: Option<String>,
    pub requires_api_key: bool,
}
```

### 5. Settings System

Comprehensive settings management with 30+ configurable options:

```rust
pub struct AppSettings {
    // Model
    pub selected_model: String,
    pub model_unload_timeout: ModelUnloadTimeout,

    // Audio
    pub always_on_microphone: bool,
    pub selected_microphone: Option<String>,
    pub clamshell_microphone: Option<String>,
    pub selected_output_device: Option<String>,
    pub mute_while_recording: bool,

    // Audio Feedback
    pub audio_feedback: bool,
    pub audio_feedback_volume: f32,
    pub sound_theme: SoundTheme,

    // Transcription
    pub translate_to_english: bool,
    pub selected_language: String,
    pub custom_words: Vec<String>,
    pub word_correction_threshold: f64,

    // Output
    pub paste_method: PasteMethod,
    pub clipboard_handling: ClipboardHandling,
    pub append_trailing_space: bool,

    // Post-Processing
    pub post_process_enabled: bool,
    pub post_process_provider_id: Option<String>,
    pub post_process_providers: Vec<PostProcessProvider>,
    pub post_process_prompts: Vec<PostProcessPrompt>,

    // UI
    pub overlay_position: OverlayPosition,
    pub start_hidden: bool,
    pub autostart_enabled: bool,
    pub app_language: String,

    // History
    pub history_limit: i64,
    pub recording_retention_period: RecordingRetentionPeriod,

    // Shortcuts
    pub bindings: HashMap<String, ShortcutBinding>,
    pub push_to_talk: bool,

    // Debug
    pub debug_mode: bool,
    pub log_level: LogLevel,
}
```

### 6. Global Keyboard Shortcuts

Dynamic shortcut registration with rebinding support:

```rust
pub struct ShortcutBinding {
    pub id: String,
    pub name: String,
    pub description: String,
    pub default_binding: String,
    pub current_binding: String,
}
```

Default shortcuts:
- `Ctrl+Shift+Space` / `Cmd+Shift+Space` - Toggle recording
- `Escape` - Cancel current operation

### 7. Internationalization (i18n)

10 supported languages with ESLint enforcement (no hardcoded strings):

- English (en) - source
- Spanish (es)
- French (fr)
- German (de)
- Italian (it)
- Japanese (ja)
- Polish (pl)
- Russian (ru)
- Vietnamese (vi)
- Chinese (zh)

---

## State Management

### Frontend (Zustand)

The settings store (`settingsStore.ts`) centralizes all app state:

```typescript
interface SettingsStore {
  // State
  settings: Settings | null;
  defaultSettings: Settings | null;
  isLoading: boolean;
  isUpdating: Record<string, boolean>;
  audioDevices: AudioDevice[];
  outputDevices: AudioDevice[];
  customSounds: { start: boolean; stop: boolean };
  postProcessModelOptions: Record<string, string[]>;

  // Actions
  initialize: () => Promise<void>;
  updateSetting: <K extends keyof Settings>(key: K, value: Settings[K]) => Promise<void>;
  resetSetting: (key: keyof Settings) => Promise<void>;
  updateBinding: (id: string, binding: string) => Promise<void>;
  // ... more actions
}
```

Settings updates are optimistic with automatic rollback on error:

```typescript
// Optimistic update
set((state) => ({ settings: { ...state.settings, [key]: value } }));

// Call backend
await settingUpdaters[key](value);

// Rollback on error
catch (error) {
  set({ settings: { ...settings, [key]: originalValue } });
}
```

### Backend (Arc<Mutex<T>>)

Thread-safe shared state using Rust's ownership model:

```rust
// Manager state wrapped in Arc<Mutex<>>
pub struct TranscriptionManager {
    engine: Arc<Mutex<Option<LoadedEngine>>>,
    current_model_id: Arc<Mutex<Option<String>>>,
    is_loading: Arc<Mutex<bool>>,
}

// Atomic types for lock-free access
last_activity: Arc<AtomicU64>,
shutdown_signal: Arc<AtomicBool>,
```

---

## Data Persistence

### Settings Storage

- **Format:** JSON via `tauri-plugin-store`
- **Location:**
  - macOS: `~/Library/Application Support/com.pais.handy/`
  - Windows: `%APPDATA%\com.pais.handy\`
  - Linux: `~/.config/com.pais.handy/`

### History Database

- **Format:** SQLite 3 (`history.db`)
- **Migrations:** Tracked via `user_version` pragma
- **Location:** App data directory

### Models

- **Location:** `{app_data}/models/`
- **Format:** Binary files (Whisper GGML) or directories (Parakeet)
- **Sizes:** 487 MB - 3 GB depending on model

### Audio Recordings

- **Location:** `{app_data}/recordings/`
- **Format:** WAV (16kHz, mono, f32)
- **Retention:** Configurable (Never keep / 3 days / 2 weeks / 3 months)

---

## Platform-Specific Features

### macOS

- **Metal acceleration** for ML inference
- **Accessibility permissions** required for keyboard simulation
- **Apple Intelligence** integration (arm64 only)
- **System mute** via AppleScript
- **NSPanel** for overlay window
- **Launch agent** for autostart

### Windows

- **Vulkan acceleration** for ML inference
- **Azure code signing**
- **COM interface** for audio muting (IAudioEndpointVolume)
- **Win32 API** for device enumeration

### Linux

- **OpenBLAS + Vulkan** acceleration
- **Multi-backend audio muting:** PipeWire (wpctl), PulseAudio (pactl), ALSA (amixer)
- **Limited Wayland support** (rdev limitations)
- **Overlay disabled by default** (X11/Wayland compatibility)
- **AppImage + RPM** distribution

---

## Build Configuration

### Development Commands

```bash
# Prerequisites
# - Rust (latest stable)
# - Bun (https://bun.sh)

# Install dependencies
bun install

# Download VAD model (required)
mkdir -p src-tauri/resources/models
curl -o src-tauri/resources/models/silero_vad_v4.onnx \
  https://blob.handy.computer/silero_vad_v4.onnx

# Development mode
bun run tauri dev
# If cmake error on macOS:
CMAKE_POLICY_VERSION_MINIMUM=3.5 bun run tauri dev

# Production build
bun run tauri build

# Linting and formatting
bun run lint              # ESLint for frontend
bun run lint:fix          # ESLint with auto-fix
bun run format            # Prettier + cargo fmt
bun run format:check      # Check formatting
```

### Release Profile (Cargo.toml)

```toml
[profile.release]
lto = true           # Link-time optimization
codegen-units = 1    # Single codegen unit for better optimization
strip = true         # Strip symbols
panic = "abort"      # Abort on panic (smaller binary)
```

### Tauri Configuration Highlights

```json
{
  "productName": "Handy",
  "version": "0.6.9",
  "identifier": "com.pais.handy",
  "bundle": {
    "macOS": {
      "minimumSystemVersion": "10.13"
    }
  },
  "plugins": {
    "updater": {
      "endpoints": ["https://github.com/cjpais/handy/releases/latest/download/latest.json"]
    }
  }
}
```

---

## Code Quality Patterns

### Type Safety

**TypeScript strict mode** with auto-generated bindings:

```typescript
// bindings.ts - Generated by tauri-specta
export type AppSettings = {
  selected_model: string;
  always_on_microphone: boolean;
  // ... 30+ typed fields
}

export const commands = {
  getAppSettings: async (): Promise<Result<AppSettings, string>>,
  // ... all commands with full type safety
}
```

### Error Handling

**Rust:** Explicit error handling with `anyhow`:

```rust
pub fn transcribe(&self, audio: Vec<f32>) -> Result<String> {
    // ...
    whisper_engine
        .transcribe_samples(audio, Some(params))
        .map_err(|e| anyhow::anyhow!("Whisper transcription failed: {}", e))?
}
```

**TypeScript:** Result type pattern:

```typescript
const result = await commands.getAppSettings();
if (result.status === "ok") {
  // Use result.data
} else {
  console.error("Error:", result.error);
}
```

### Event-Driven Architecture

Backend to Frontend communication via Tauri events:

```rust
// Rust: Emit event
self.app_handle.emit("model-state-changed", ModelStateEvent {
    event_type: "loading_completed".to_string(),
    model_id: Some(model_id.to_string()),
    model_name: Some(model_info.name.clone()),
    error: None,
})?;
```

```typescript
// TypeScript: Listen for event
listen<ModelStateEvent>("model-state-changed", (event) => {
  if (event.payload.event_type === "loading_completed") {
    // Update UI
  }
});
```

---

## Notable Design Decisions

### 1. Privacy-First Architecture

- All processing happens locally (no cloud APIs for transcription)
- Optional LLM post-processing clearly marked
- Audio recordings stored locally with configurable retention
- No telemetry or analytics

### 2. Dual Engine Support

Supports both Whisper (OpenAI) and Parakeet (NVIDIA) engines:

```rust
enum LoadedEngine {
    Whisper(WhisperEngine),   // GGML-based, 4 model sizes
    Parakeet(ParakeetEngine), // NVIDIA's state-of-the-art
}
```

### 3. Lazy Model Loading

Models loaded only when needed, with configurable unload timeouts to balance memory usage and response time:

```rust
pub enum ModelUnloadTimeout {
    Never,       // Keep loaded forever
    Immediately, // Unload after each use
    Sec5,        // Debug only
    Min2, Min5, Min10, Min15, Min30, Min60,
}
```

### 4. Cross-Platform Audio Handling

Platform-specific implementations for audio muting:

- **Windows:** COM/Win32 IAudioEndpointVolume
- **macOS:** AppleScript
- **Linux:** wpctl (PipeWire) -> pactl (PulseAudio) -> amixer (ALSA) fallback chain

### 5. Type-Safe Frontend-Backend Bridge

Using `tauri-specta` for automatic TypeScript binding generation:

```rust
#[tauri::command]
#[specta::specta]
fn get_app_settings(app: AppHandle) -> Result<AppSettings, String> {
    // ...
}
```

Generates type-safe TypeScript:

```typescript
export const commands = {
  getAppSettings: async (): Promise<Result<AppSettings, string>>
}
```

---

## Potential Improvements

1. **Testing:** No visible test suite - could benefit from unit/integration tests
2. **Documentation:** Limited inline documentation in some modules
3. **Error Recovery:** Some error paths could be more graceful
4. **Accessibility:** Could improve screen reader support
5. **Plugin System:** Architecture could support community plugins

---

## Conclusion

Handy demonstrates a well-architected Tauri application with:

- Clean separation between frontend (React/TypeScript) and backend (Rust)
- Strong typing throughout the stack via auto-generated bindings
- Thoughtful state management (Zustand + Arc<Mutex<T>>)
- Privacy-focused local processing
- Cross-platform support with platform-specific optimizations
- Comprehensive settings system
- Event-driven communication patterns

The codebase follows modern best practices for desktop application development and provides a solid foundation for a production-ready speech-to-text tool.
