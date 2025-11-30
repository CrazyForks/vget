package downloader

import (
	"fmt"
	"time"
)

// Downloader handles file downloads with progress reporting
type Downloader struct {
	lang string
}

// New creates a new Downloader
func New(lang string) *Downloader {
	return &Downloader{
		lang: lang,
	}
}

// Download downloads a file from URL to the specified path using TUI
func (d *Downloader) Download(url, output, videoID string) error {
	return RunDownloadTUI(url, output, videoID, d.lang)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "??:??"
	}
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	if m > 60 {
		h := m / 60
		m = m % 60
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
