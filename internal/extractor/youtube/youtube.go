package youtube

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Extractor handles YouTube video downloads using browser automation + Innertube API
type Extractor struct {
	visible bool // show browser window for debugging
}

// SetVisible configures whether to show the browser window
func (e *Extractor) SetVisible(visible bool) {
	e.visible = visible
}

func (e *Extractor) Name() string {
	return "youtube"
}

func (e *Extractor) Match(u *url.URL) bool {
	host := strings.ToLower(u.Host)
	return strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be")
}

// Extract extracts video info from YouTube URL
func (e *Extractor) Extract(rawURL string) (*VideoMedia, error) {
	videoID := e.extractVideoID(rawURL)
	if videoID == "" {
		return nil, fmt.Errorf("could not extract video ID from URL: %s", rawURL)
	}

	fmt.Printf("Extracting YouTube video: %s\n", videoID)

	// Step 1: Get session tokens via browser (or from cache)
	session, err := e.getSession(videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.POToken != "" {
		fmt.Printf("Got session - POToken: %d chars, VisitorData: %s...\n",
			len(session.POToken), truncate(session.VisitorData, 20))
	} else {
		fmt.Println("Warning: No POToken captured, trying without it...")
	}

	// Step 2: Call Innertube API with tokens
	response, err := e.callInnertubeAPI(videoID, session)
	if err != nil {
		return nil, fmt.Errorf("failed to call Innertube API: %w", err)
	}

	// Step 3: Parse response into Media
	return e.parseResponse(response, session)
}

func (e *Extractor) extractVideoID(rawURL string) string {
	patterns := []string{
		`(?:youtube\.com/watch\?v=|youtu\.be/|youtube\.com/embed/|youtube\.com/v/|youtube\.com/shorts/)([a-zA-Z0-9_-]{11})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(rawURL)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Try to find v= parameter
	u, err := url.Parse(rawURL)
	if err == nil {
		if v := u.Query().Get("v"); v != "" {
			return v
		}
	}

	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
