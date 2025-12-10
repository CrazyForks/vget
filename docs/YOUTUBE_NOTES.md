# YouTube Support Notes

## Status: Delegated to yt-dlp (Docker Only)

After extensive research and failed attempts, we've concluded that building a native Go YouTube extractor is not viable.

## What We Tried (2025-12-04)

### The Go Implementation Worked... Briefly

- Browser automation (Rod + stealth) captured BotGuard tokens
- Innertube API with iOS client returned unencrypted stream URLs (no cipher)
- Separate video/audio streams downloaded and merged with ffmpeg

### Then It Broke

1. **BotGuard Detection** - YouTube's anti-bot (Error 153) detected rod/stealth automation
2. **IP Binding** - Stream URLs are bound to the requesting IP; VPNs/IPv6 cause 403s
3. **Rate Limiting** - Heavy testing flagged our IP/session; even new IPs didn't help
4. **Constant Changes** - YouTube updates anti-bot weekly; we can't keep up

## Why We Don't Build Our Own Extractor

### The Problems Are Real

1. **Aggressive Anti-Bot Detection**
   - PO Tokens (Proof of Origin) require JavaScript execution
   - N parameter challenge requires solving obfuscated JS functions
   - SAPISID hash authentication with rotating signatures
   - Client version checks that change frequently
   - Rate limiting that bans IPs quickly

2. **Constantly Moving Target**
   - YouTube updates their anti-bot mechanisms weekly
   - yt-dlp has 1000+ contributors constantly reverse-engineering changes
   - A solo developer cannot keep up with Google's anti-bot team

3. **IP Bans Are Inevitable**
   - Even with all the right tokens and signatures, YouTube rate-limits aggressively
   - Residential IPs get banned after moderate usage
   - Datacenter IPs are blocked almost immediately

4. **Resource Requirements**
   - Requires JavaScript runtime (Node.js/Deno) for challenge solving
   - Needs rotating residential proxies ($$$)
   - Cookie/session management is complex

## Our Solution

We delegate YouTube extraction to **yt-dlp** and **youtube-dl**, but only in Docker:

- **In Docker**: vget shells out to yt-dlp/youtube-dl
- **Outside Docker**: vget shows an error suggesting Docker usage

### Why Docker Only?

1. Windows/Mac users won't have Python installed
2. Bundling yt-dlp in the Go binary is impractical
3. Docker image includes all dependencies (Python, ffmpeg, Node.js)
4. NAS users (Synology, QNAP, Unraid) commonly use Docker

## User Responsibilities

**IMPORTANT**: Users must provide their own infrastructure:

1. **Residential Proxy / Rotating IPs** - YouTube will ban datacenter IPs and rate-limit residential IPs. Users need to configure their own proxy solution. **This is not optional for sustained usage.**

2. **Cookies (Optional)** - For age-restricted or premium content, users can mount a cookies file.

3. **Rate Limiting** - Users should use `--sleep-interval` with yt-dlp to avoid bans.

## Usage

```bash
# Basic usage (user handles proxy externally)
docker run -v ~/downloads:/downloads guiyumin/vget "https://youtube.com/watch?v=xxx"

# With proxy configured in environment
docker run -e HTTP_PROXY=http://proxy:port -v ~/downloads:/downloads guiyumin/vget "https://youtube.com/watch?v=xxx"

# With cookies file for premium/age-restricted content
docker run -v ~/downloads:/downloads -v ~/cookies.txt:/home/vget/cookies.txt guiyumin/vget "https://youtube.com/watch?v=xxx"
```

## Alternatives for Users

If Docker isn't an option, users should use yt-dlp directly:

```bash
# Install yt-dlp
pip install yt-dlp

# Download video
yt-dlp "https://youtube.com/watch?v=xxx"

# With proxy
yt-dlp --proxy http://proxy:port "https://youtube.com/watch?v=xxx"
```

## Old Troubleshooting (For Reference)

These were issues with our native Go implementation:

### 403 on download
1. Clear browser profile: `rm -rf ~/.config/vget/browser/`
2. Disable IPv6: `sudo networksetup -setv6off Wi-Fi`
3. Try a different network/IP
4. Wait for rate limiting to expire (24-48 hours)

### No POToken captured
- YouTube detecting automation (Error 153)
- go-rod/stealth needs constant updates

### IP mismatch
- VPN must tunnel ALL traffic (not just browser)
- Disable IPv6 to force IPv4
- Browser, API call, and download must use same IP

## References

- [yt-dlp GitHub](https://github.com/yt-dlp/yt-dlp)
- [youtube-dl GitHub](https://github.com/ytdl-org/youtube-dl)
- [yt-dlp Wiki: Rate Limiting](https://github.com/yt-dlp/yt-dlp/wiki/Extractors#this-content-isnt-available-try-again-later)

## Lessons Learned

1. Don't fight Google's anti-bot team alone
2. Leverage existing open-source solutions (yt-dlp has 1000+ contributors)
3. Make infrastructure (proxies, IPs) the user's responsibility
4. Docker is the right abstraction for complex dependencies
5. Know when to give up and delegate
