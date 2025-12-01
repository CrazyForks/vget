# vget

多功能命令行下载工具，支持音频、视频、播客等。

[English](README.md) | [日本語](README_jp.md) | [한국어](README_kr.md) | [Español](README_es.md) | [Français](README_fr.md) | [Deutsch](README_de.md)

## 安装

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

从 [Releases](https://github.com/guiyumin/vget/releases/latest) 下载 `vget-windows-amd64.exe` 并添加到系统 PATH。

## 命令

### 初始化配置

```bash
vget init
```

运行交互式向导创建配置文件。

### 下载媒体

```bash
vget <url>
```

从支持的 URL 下载音频或视频。

**参数：**

- `-o, --output <file>` - 输出文件名
- `-q, --quality <quality>` - 首选质量（如 1080p, 720p）
- `--info` - 仅显示媒体信息，不下载

**示例：**

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o my_video.mp4
vget --info https://example.com/video
```

### 搜索播客

```bash
vget search --podcast <query>
```

交互式搜索播客。中文搜索使用小宇宙，其他语言使用 iTunes。

**示例：**

```bash
vget search --podcast "tech news"
vget search --podcast "科技"
```

### 更新 vget

```bash
vget update
```

更新 vget 到最新版本。

## 支持的来源

| 来源           | 类型            | 状态   |
| -------------- | --------------- | ------ |
| Twitter/X      | 视频            | 已支持 |
| 小宇宙 FM      | 音频（播客）    | 已支持 |
| Apple Podcasts | 音频（播客）    | 已支持 |

## 配置

配置文件位置：

| 操作系统    | 路径                        |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

运行 `vget init` 交互式创建配置文件，或手动创建：

```yaml
language: zh # en, zh, jp, kr, es, fr, de
```

## 语言

vget 支持多种语言：

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## 许可证

Apache License 2.0
