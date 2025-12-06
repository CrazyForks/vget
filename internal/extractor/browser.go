package extractor

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/guiyumin/vget/internal/config"
)

// BrowserExtractor uses browser automation to intercept media URLs
type BrowserExtractor struct {
	site    *config.Site
	visible bool
}

// NewBrowserExtractor creates a new browser extractor for the given site
func NewBrowserExtractor(site *config.Site, visible bool) *BrowserExtractor {
	return &BrowserExtractor{site: site, visible: visible}
}

func (e *BrowserExtractor) Name() string {
	return "browser"
}

func (e *BrowserExtractor) Match(u *url.URL) bool {
	return true // Called only when site matches
}

func (e *BrowserExtractor) Extract(rawURL string) (Media, error) {
	if e.site == nil {
		return nil, fmt.Errorf("no site configuration provided")
	}

	// Parse the page URL to get origin
	pageURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	pageOrigin := fmt.Sprintf("%s://%s", pageURL.Scheme, pageURL.Host)

	// Determine what extension to look for
	targetExt := "." + e.site.Type // e.g., ".m3u8", ".mp4"

	fmt.Printf("Looking for %s requests...\n", e.site.Type)

	// Launch browser
	l := e.createLauncher(!e.visible) // headless unless --visible flag
	defer l.Cleanup()

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	defer page.MustClose()

	// Enable Network domain to capture ALL requests (including from Workers)
	_ = proto.NetworkEnable{}.Call(page)

	var mediaURL string
	var mu sync.Mutex
	done := make(chan struct{})
	closed := false

	// Listen for network requests at CDP level
	go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
		reqURL := e.Request.URL
		if strings.Contains(strings.ToLower(reqURL), targetExt) {
			mu.Lock()
			if mediaURL == "" {
				mediaURL = reqURL
				fmt.Printf("Captured: %s\n", reqURL)
				if !closed {
					closed = true
					close(done)
				}
			}
			mu.Unlock()
		}
	})()

	// Navigate
	_ = page.Navigate(rawURL)
	_ = page.WaitLoad()

	// Wait for capture or timeout
	select {
	case <-done:
		// Found
	case <-time.After(15 * time.Second):
		// Timeout
	}

	// If not found via interception, try fallback methods
	if mediaURL == "" {
		fmt.Println("Trying fallback: performance_api")
		mediaURL = e.findM3U8InPerformance(page, targetExt)
	}
	if mediaURL == "" {
		fmt.Println("Trying fallback: video_player")
		mediaURL = e.findM3U8FromVideoPlayer(page)
	}
	if mediaURL == "" {
		fmt.Println("Trying fallback: page_source")
		html, _ := page.HTML()
		mediaURL = e.findM3U8InSource(html)
	}

	if mediaURL == "" {
		return nil, fmt.Errorf("no %s request captured", e.site.Type)
	}

	fmt.Printf("Found: %s\n", mediaURL)

	// Extract page title
	title := page.MustEval(`() => document.title`).String()
	title = strings.TrimSpace(title)
	if title == "" {
		pageURL, _ := url.Parse(rawURL)
		title = filepath.Base(pageURL.Path)
		if title == "" || title == "/" {
			title = pageURL.Host
		}
	}

	// Generate ID from URL
	parsedURL, _ := url.Parse(mediaURL)
	id := filepath.Base(parsedURL.Path)
	if idx := strings.LastIndex(id, "."); idx > 0 {
		id = id[:idx]
	}
	if id == "" || id == "/" {
		id = "video"
	}

	return &VideoMedia{
		ID:    id,
		Title: title,
		Formats: []VideoFormat{
			{
				URL:     mediaURL,
				Quality: "best",
				Ext:     e.site.Type,
				Headers: map[string]string{"Referer": rawURL, "Origin": pageOrigin},
			},
		},
	}, nil
}

// findM3U8InPerformance uses the browser's Performance API to find resource requests
func (e *BrowserExtractor) findM3U8InPerformance(page *rod.Page, targetExt string) string {
	// Query the Performance API for all resource entries
	result, err := page.Eval(`() => {
		return performance.getEntriesByType('resource')
			.map(r => r.name)
			.filter(url => url.toLowerCase().includes('.m3u8') || url.toLowerCase().includes('.ts') || url.toLowerCase().includes('hls'));
	}`)
	if err != nil {
		return ""
	}

	// Parse the result
	arr := result.Value.Arr()
	for _, v := range arr {
		url := v.String()
		if strings.Contains(strings.ToLower(url), targetExt) {
			return url
		}
	}

	return ""
}

// findM3U8InSource searches for m3u8 URLs in page HTML/JavaScript source
func (e *BrowserExtractor) findM3U8InSource(html string) string {
	// Common patterns for m3u8 URLs in page source
	patterns := []string{
		// Direct m3u8 URL patterns
		`https?://[^"'\s<>]+\.m3u8[^"'\s<>]*`,
		// URL in quotes
		`["']([^"']*\.m3u8[^"']*)["']`,
		// source attribute
		`src\s*[=:]\s*["']([^"']*\.m3u8[^"']*)["']`,
		// file/url parameter
		`(?:file|url|source|src)\s*[=:]\s*["']([^"']+)["']`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			var url string
			if len(match) > 1 {
				url = match[1]
			} else {
				url = match[0]
			}

			// Must contain m3u8
			if !strings.Contains(strings.ToLower(url), ".m3u8") {
				continue
			}

			// Skip data URLs and invalid URLs
			if strings.HasPrefix(url, "data:") {
				continue
			}

			// Clean up the URL
			url = strings.TrimSpace(url)
			if url != "" {
				return url
			}
		}
	}

	return ""
}

// findM3U8FromVideoPlayer queries the video player for its source URL
func (e *BrowserExtractor) findM3U8FromVideoPlayer(page *rod.Page) string {
	// Try to get the source from various video player APIs
	result, err := page.Eval(`() => {
		// Check for HLS.js
		if (window.Hls && window.hls) {
			return window.hls.url || '';
		}
		// Check for video.js
		const vjsPlayer = document.querySelector('.video-js');
		if (vjsPlayer && vjsPlayer.player) {
			const src = vjsPlayer.player.currentSrc();
			if (src && src.includes('.m3u8')) return src;
		}
		// Check video element sources
		const video = document.querySelector('video');
		if (video) {
			if (video.src && video.src.includes('.m3u8')) return video.src;
			const source = video.querySelector('source[src*=".m3u8"]');
			if (source) return source.src;
		}
		// Check for any global player variable
		if (window.player && window.player.src) {
			const src = typeof window.player.src === 'function' ? window.player.src() : window.player.src;
			if (src && src.includes('.m3u8')) return src;
		}
		return '';
	}`)
	if err != nil {
		return ""
	}
	return result.Value.String()
}

func (e *BrowserExtractor) createLauncher(headless bool) *launcher.Launcher {
	userDataDir := e.getUserDataDir()

	l := launcher.New().
		Headless(headless).
		UserDataDir(userDataDir).
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage")

	return l
}

func (e *BrowserExtractor) getUserDataDir() string {
	configDir, err := config.ConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "vget-browser")
	}
	return filepath.Join(configDir, "browser")
}
