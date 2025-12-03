package extractor

import (
	"fmt"
	"net/url"
)

// YouTubeExtractor handles YouTube video downloads
type YouTubeExtractor struct{}

func (e *YouTubeExtractor) Name() string {
	return "youtube"
}

func (e *YouTubeExtractor) Match(u *url.URL) bool {
	// Host matching is done by registry
	return true
}

func (e *YouTubeExtractor) Extract(url string) (Media, error) {
	return nil, fmt.Errorf("YouTube support coming soon")
}

func init() {
	Register(&YouTubeExtractor{},
		"youtube.com",
		"youtu.be",
		"m.youtube.com",
	)
}
