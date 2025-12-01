# vget

Herramienta de línea de comandos versátil para descargar audio, video, podcasts y más.

[English](README.md) | [简体中文](README_zh.md) | [日本語](README_jp.md) | [한국어](README_kr.md) | [Français](README_fr.md) | [Deutsch](README_de.md)

## Instalación

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

Descarga `vget-windows-amd64.exe` desde [Releases](https://github.com/guiyumin/vget/releases/latest) y agrégalo al PATH.

## Comandos

### Inicializar configuración

```bash
vget init
```

Ejecuta un asistente interactivo para crear el archivo de configuración.

### Descargar medios

```bash
vget <url>
```

Descarga audio o video desde una URL compatible.

**Opciones:**

- `-o, --output <file>` - Nombre del archivo de salida
- `-q, --quality <quality>` - Calidad preferida (ej: 1080p, 720p)
- `--info` - Mostrar información del medio sin descargar

**Ejemplos:**

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o mi_video.mp4
vget --info https://example.com/video
```

### Buscar podcasts

```bash
vget search --podcast <query>
```

Busca podcasts de forma interactiva. Las búsquedas en chino usan Xiaoyuzhou, las demás usan iTunes.

**Ejemplos:**

```bash
vget search --podcast "tech news"
vget search --podcast "科技"
```

### Actualizar vget

```bash
vget update
```

Actualiza vget a la última versión.

## Fuentes compatibles

| Fuente         | Tipo            | Estado     |
| -------------- | --------------- | ---------- |
| Twitter/X      | Video           | Soportado  |
| Xiaoyuzhou FM  | Audio (Podcast) | Soportado  |
| Apple Podcasts | Audio (Podcast) | Soportado  |

## Configuración

Ubicación del archivo de configuración:

| SO          | Ruta                        |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

Ejecuta `vget init` para crear el archivo de configuración interactivamente, o créalo manualmente:

```yaml
language: es # en, zh, jp, kr, es, fr, de
```

## Idiomas

vget soporta múltiples idiomas:

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## Licencia

Apache License 2.0
