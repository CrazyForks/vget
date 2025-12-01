# vget

Outil en ligne de commande polyvalent pour télécharger audio, vidéo, podcasts et plus.

[English](README.md) | [简体中文](README_zh.md) | [日本語](README_jp.md) | [한국어](README_kr.md) | [Español](README_es.md) | [Deutsch](README_de.md)

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

Téléchargez `vget-windows-amd64.exe` depuis [Releases](https://github.com/guiyumin/vget/releases/latest) et ajoutez-le au PATH.

## Commandes

### Initialiser la configuration

```bash
vget init
```

Lance un assistant interactif pour créer le fichier de configuration.

### Télécharger des médias

```bash
vget <url>
```

Télécharge audio ou vidéo depuis une URL prise en charge.

**Options :**

- `-o, --output <file>` - Nom du fichier de sortie
- `-q, --quality <quality>` - Qualité préférée (ex : 1080p, 720p)
- `--info` - Afficher les informations du média sans télécharger

**Exemples :**

```bash
vget https://twitter.com/user/status/123456789
vget https://www.xiaoyuzhoufm.com/episode/abc123
vget https://example.com/video -o ma_video.mp4
vget --info https://example.com/video
```

### Rechercher des podcasts

```bash
vget search --podcast <query>
```

Recherche interactive de podcasts. Les requêtes en chinois utilisent Xiaoyuzhou, les autres utilisent iTunes.

**Exemples :**

```bash
vget search --podcast "tech news"
vget search --podcast "科技"
```

### Mettre à jour vget

```bash
vget update
```

Met à jour vget vers la dernière version.

## Sources prises en charge

| Source         | Type            | Statut     |
| -------------- | --------------- | ---------- |
| Twitter/X      | Vidéo           | Supporté   |
| Xiaoyuzhou FM  | Audio (Podcast) | Supporté   |
| Apple Podcasts | Audio (Podcast) | Supporté   |

## Configuration

Emplacement du fichier de configuration :

| OS          | Chemin                      |
| ----------- | --------------------------- |
| macOS/Linux | `~/.config/vget/config.yml` |
| Windows     | `%APPDATA%\vget\config.yml` |

Exécutez `vget init` pour créer le fichier de configuration de manière interactive, ou créez-le manuellement :

```yaml
language: fr # en, zh, jp, kr, es, fr, de
```

## Langues

vget prend en charge plusieurs langues :

- English (en)
- 中文 (zh)
- 日本語 (jp)
- 한국어 (kr)
- Español (es)
- Français (fr)
- Deutsch (de)

## Licence

Apache License 2.0
