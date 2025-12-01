# vget

오디오, 비디오, 팟캐스트 등을 다운로드하는 다목적 명령줄 도구.

[English](README.md) | [简体中文](README_zh.md) | [日本語](README_jp.md) | [Español](README_es.md) | [Français](README_fr.md) | [Deutsch](README_de.md)

## 설치

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

[Releases](https://github.com/guiyumin/vget/releases/latest)에서 `vget-windows-amd64.exe`를 다운로드하고 PATH에 추가하세요.

## 명령어

### 설정 초기화

```bash
vget init
```

대화형 마법사를 실행하여 설정 파일을 생성합니다.

### 미디어 다운로드

```bash
vget <url>
```

지원되는 URL에서 오디오 또는 비디오를 다운로드합니다.

**옵션:**

- `-o, --output <file>` - 출력 파일명
- `-q, --quality <quality>` - 선호 품질 (예: 1080p, 720p)
- `--info` - 다운로드 없이 미디어 정보만 표시

**예시:**

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o my_video.mp4
vget --info https://example.com/video
```

### 팟캐스트 검색

```bash
vget search --podcast <query>
```

팟캐스트를 대화형으로 검색합니다. 중국어 검색은 小宇宙를, 그 외에는 iTunes를 사용합니다.

**예시:**

```bash
vget search --podcast "tech news"
vget search --podcast "科技"
```

### vget 업데이트

```bash
vget update
```

vget을 최신 버전으로 업데이트합니다.

## 지원 소스

| 소스           | 유형            | 상태   |
| -------------- | --------------- | ------ |
| Twitter/X      | 비디오          | 지원됨 |
| 小宇宙 FM      | 오디오 (팟캐스트) | 지원됨 |
| Apple Podcasts | 오디오 (팟캐스트) | 지원됨 |

## 설정

설정 파일 위치:

| OS          | 경로                        |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

`vget init`으로 대화형으로 설정 파일을 생성하거나 수동으로 생성하세요:

```yaml
language: kr # en, zh, jp, kr, es, fr, de
```

## 언어

vget은 여러 언어를 지원합니다:

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## 라이선스

Apache License 2.0
