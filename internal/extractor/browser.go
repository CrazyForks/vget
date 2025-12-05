package extractor

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	site *config.Site
}

// NewBrowserExtractor creates a new browser extractor for the given site
func NewBrowserExtractor(site *config.Site) *BrowserExtractor {
	return &BrowserExtractor{site: site}
}

func (e *BrowserExtractor) Name() string {
	return "browser"
}

func (e *BrowserExtractor) Match(u *url.URL) bool {
	return true // Called only when site matches
}

// capturedRequest holds information about an intercepted request
type capturedRequest struct {
	url     string
	headers map[string]string
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
	l := e.createLauncher(true) // headless
	defer l.Cleanup()

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	defer page.MustClose()

	// Set up request interception
	var captured *capturedRequest
	var mu sync.Mutex
	done := make(chan struct{})

	// Enable network events
	router := page.HijackRequests()
	defer router.Stop()

	router.MustAdd("*", func(ctx *rod.Hijack) {
		reqURL := ctx.Request.URL().String()

		// Check if this is the type we're looking for
		if strings.Contains(strings.ToLower(reqURL), targetExt) {
			mu.Lock()
			if captured == nil {
				// Use the page URL as Referer and Origin
				// (browser auto-sets these but hijacking may not expose them)
				headers := map[string]string{
					"Referer": rawURL,
					"Origin":  pageOrigin,
				}

				captured = &capturedRequest{
					url:     reqURL,
					headers: headers,
				}
				fmt.Printf("Captured %s URL!\n", e.site.Type)
				close(done)
			}
			mu.Unlock()
		}

		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()

	// Navigate to page
	fmt.Printf("Loading page: %s\n", rawURL)
	err = page.Navigate(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait for either capture or timeout
	select {
	case <-done:
		// Got it!
	case <-time.After(30 * time.Second):
		mu.Lock()
		if captured == nil {
			mu.Unlock()
			return nil, fmt.Errorf("timeout: no %s request found after 30 seconds", e.site.Type)
		}
		mu.Unlock()
	}

	mu.Lock()
	result := captured
	mu.Unlock()

	if result == nil {
		return nil, fmt.Errorf("no %s request captured", e.site.Type)
	}

	// Extract page title
	title := page.MustEval(`() => document.title`).String()
	title = strings.TrimSpace(title)
	if title == "" {
		// Fallback to URL path
		pageURL, _ := url.Parse(rawURL)
		title = filepath.Base(pageURL.Path)
		if title == "" || title == "/" {
			title = pageURL.Host
		}
	}

	// Generate ID from URL
	parsedURL, _ := url.Parse(result.url)
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
				URL:     result.url,
				Quality: "best",
				Ext:     e.site.Type,
				Headers: result.headers,
			},
		},
	}, nil
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
