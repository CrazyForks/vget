package extractor

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// YouTubeDockerRequiredError indicates YouTube extraction needs Docker
type YouTubeDockerRequiredError struct {
	URL string
}

func (e *YouTubeDockerRequiredError) Error() string {
	return "YouTube extraction requires Docker"
}

// YouTubeDirectDownload indicates yt-dlp should handle the download directly
type YouTubeDirectDownload struct {
	URL       string
	OutputDir string
}

// Implement Media interface for YouTubeDirectDownload
func (y *YouTubeDirectDownload) GetID() string       { return y.URL }
func (y *YouTubeDirectDownload) GetTitle() string    { return "YouTube Video" }
func (y *YouTubeDirectDownload) GetUploader() string { return "" }
func (y *YouTubeDirectDownload) Type() MediaType     { return MediaTypeVideo }

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

	// For YouTube, we return a special marker that tells the CLI
	// to use yt-dlp for direct download instead of vget's downloader
	// OutputDir will be set by CLI from config
	return &YouTubeDirectDownload{
		URL: urlStr,
	}, nil
}

// DownloadWithYtdlp downloads a YouTube video using yt-dlp directly
func DownloadWithYtdlp(url, outputDir string) error {
	// Try yt-dlp first
	outputTemplate := filepath.Join(outputDir, "%(title)s.%(ext)s")

	cmd := exec.Command("yt-dlp",
		"-f", "bv*+ba/b", // best video + best audio, or best combined
		"--merge-output-format", "mp4",
		"--no-playlist",
		"--remote-components", "ejs:github", // download JS challenge solver
		"-o", outputTemplate,
		"--progress",
		url,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err == nil {
		return nil
	}

	// Fallback to youtube-dl
	cmd = exec.Command("youtube-dl",
		"-f", "bestvideo+bestaudio/best",
		"--merge-output-format", "mp4",
		"--no-playlist",
		"-o", outputTemplate,
		url,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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
