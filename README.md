# vget

Versatile command-line toolkit for downloading audio, video, podcasts, and more.

[简体中文](README_zh.md) | [日本語](README_jp.md) | [한국어](README_kr.md) | [Español](README_es.md) | [Français](README_fr.md) | [Deutsch](README_de.md)

## Installation

### macOS

```bash
curl -fsSL https://github.com/guiyumin/vget/releases/latest/download/vget-darwin-arm64 -o vget
chmod +x vget
sudo mv vget /usr/local/bin/
```

### Linux / WSL

```bash
curl -fsSL https://github.com/guiyumin/vget/releases/latest/download/vget-linux-amd64 -o vget
chmod +x vget
sudo mv vget /usr/local/bin/
```

### Windows

Download `vget-windows-amd64.exe` from [Releases](https://github.com/guiyumin/vget/releases/latest) and add it to your PATH.

## Commands

### Initialize config

```bash
vget init
```

Run an interactive wizard to create a config file.

### Download media

```bash
vget <url>
```

Download audio or video from a supported URL.

**Flags:**

- `-o, --output <file>` - Output filename
- `-q, --quality <quality>` - Preferred quality (e.g., 1080p, 720p)
- `--info` - Show media info without downloading

**Examples:**

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o my_video.mp4
vget --info https://example.com/video
```

### Search podcasts

```bash
vget search --podcast <query>
```

Search for podcasts interactively. Chinese queries use Xiaoyuzhou, others use iTunes.

**Examples:**

```bash
vget search --podcast "tech news"
vget search --podcast "科技"
```

### Update vget

```bash
vget update
```

Update vget to the latest version.

## Supported Sources

| Source         | Type            | Status    |
| -------------- | --------------- | --------- |
| Twitter/X      | Video           | Supported |
| Xiaoyuzhou FM  | Audio (Podcast) | Supported |
| Apple Podcasts | Audio (Podcast) | Supported |

## Configuration

Config file location:

| OS          | Path                        |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

Run `vget init` to create the config file interactively, or create it manually:

```yaml
language: en # en, zh, jp, kr, es, fr, de
```

## Languages

vget supports multiple languages:

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## License

Apache License 2.0
