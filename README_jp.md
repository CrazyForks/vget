# vget

オーディオ、ビデオ、ポッドキャストなどをダウンロードする多機能コマンドラインツール。

[English](README.md) | [简体中文](README_zh.md) | [한국어](README_kr.md) | [Español](README_es.md) | [Français](README_fr.md) | [Deutsch](README_de.md)

## インストール

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

[Releases](https://github.com/guiyumin/vget/releases/latest) から `vget-windows-amd64.exe` をダウンロードし、PATH に追加してください。

## コマンド

### 設定の初期化

```bash
vget init
```

対話式ウィザードで設定ファイルを作成します。

### メディアのダウンロード

```bash
vget <url>
```

対応する URL からオーディオまたはビデオをダウンロードします。

**オプション：**

- `-o, --output <file>` - 出力ファイル名
- `-q, --quality <quality>` - 優先品質（例：1080p, 720p）
- `--info` - ダウンロードせずにメディア情報を表示

**例：**

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o my_video.mp4
vget --info https://example.com/video
```

### ポッドキャスト検索

```bash
vget search --podcast <query>
```

ポッドキャストをインタラクティブに検索します。中国語のクエリは小宇宙、その他は iTunes を使用します。

**例：**

```bash
vget search --podcast "tech news"
vget search --podcast "科技"
```

### vget の更新

```bash
vget update
```

vget を最新バージョンに更新します。

## 対応ソース

| ソース         | タイプ          | 状態   |
| -------------- | --------------- | ------ |
| Twitter/X      | ビデオ          | 対応済 |
| 小宇宙 FM      | オーディオ      | 対応済 |
| Apple Podcasts | オーディオ      | 対応済 |

## 設定

設定ファイルの場所：

| OS          | パス                        |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

`vget init` で対話的に設定ファイルを作成するか、手動で作成してください：

```yaml
language: jp # en, zh, jp, kr, es, fr, de
```

## 言語

vget は複数の言語をサポートしています：

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## ライセンス

Apache License 2.0
