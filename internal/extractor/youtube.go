package extractor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// YouTubeDockerRequiredError indicates YouTube extraction needs Docker
type YouTubeDockerRequiredError struct {
	URL string
}

func (e *YouTubeDockerRequiredError) Error() string {
	return "YouTube extraction requires Docker"
}

// ytdlpExtractor uses yt-dlp/youtube-dl for YouTube extraction (Docker only)
type ytdlpExtractor struct{}

func (e *ytdlpExtractor) Name() string {
	return "YouTube (yt-dlp)"
}

func (e *ytdlpExtractor) Match(u *url.URL) bool {
	host := strings.ToLower(u.Host)
	return host == "youtube.com" ||
		host == "www.youtube.com" ||
		host == "youtu.be" ||
		host == "m.youtube.com" ||
		host == "music.youtube.com"
}

func (e *ytdlpExtractor) Extract(urlStr string) (Media, error) {
	if !isRunningInDocker() {
		return nil, &YouTubeDockerRequiredError{URL: urlStr}
	}

	// Try yt-dlp first, fall back to youtube-dl
	info, err := extractWithYtdlp(urlStr)
	if err != nil {
		info, err = extractWithYoutubeDL(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to extract YouTube video: %w", err)
		}
	}

	return info, nil
}

// ytdlpInfo represents the JSON output from yt-dlp/youtube-dl
type ytdlpInfo struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Uploader  string        `json:"uploader"`
	Thumbnail string        `json:"thumbnail"`
	Formats   []ytdlpFormat `json:"formats"`
}

type ytdlpFormat struct {
	URL        string            `json:"url"`
	FormatID   string            `json:"format_id"`
	Ext        string            `json:"ext"`
	Width      int               `json:"width"`
	Height     int               `json:"height"`
	Vcodec     string            `json:"vcodec"`
	Acodec     string            `json:"acodec"`
	TBR        float64           `json:"tbr"`
	HTTPHeader map[string]string `json:"http_headers"`
}

func extractWithYtdlp(urlStr string) (Media, error) {
	return extractWithCmd("yt-dlp", urlStr)
}

func extractWithYoutubeDL(urlStr string) (Media, error) {
	return extractWithCmd("youtube-dl", urlStr)
}

func extractWithCmd(cmd string, urlStr string) (Media, error) {
	// Use -j to get JSON metadata, -f best for best single format
	out, err := exec.Command(cmd, "-j", "--no-playlist", urlStr).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s failed: %s", cmd, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("%s not found or failed: %w", cmd, err)
	}

	var info ytdlpInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("failed to parse %s output: %w", cmd, err)
	}

	// Convert to VideoMedia
	formats := make([]VideoFormat, 0, len(info.Formats))
	for _, f := range info.Formats {
		// Skip formats without URL
		if f.URL == "" {
			continue
		}

		// Build quality string
		quality := f.FormatID
		if f.Height > 0 {
			quality = fmt.Sprintf("%dp", f.Height)
		}

		vf := VideoFormat{
			URL:     f.URL,
			Quality: quality,
			Ext:     f.Ext,
			Width:   f.Width,
			Height:  f.Height,
			Headers: f.HTTPHeader,
		}

		if f.TBR > 0 {
			vf.Bitrate = int(f.TBR * 1000)
		}

		formats = append(formats, vf)
	}

	return &VideoMedia{
		ID:        info.ID,
		Title:     info.Title,
		Uploader:  info.Uploader,
		Thumbnail: info.Thumbnail,
		Formats:   formats,
	}, nil
}

// isRunningInDocker detects if we're running inside a Docker container
func isRunningInDocker() bool {
	// Method 1: Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Method 2: Check cgroup (Linux)
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}

	// Method 3: Check for kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	return false
}

func init() {
	Register(&ytdlpExtractor{},
		"youtube.com",
		"www.youtube.com",
		"youtu.be",
		"m.youtube.com",
		"music.youtube.com",
	)
}
