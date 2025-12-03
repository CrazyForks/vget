package extractor

import (
	"fmt"
	"net/url"
)

// XiaohongshuExtractor handles Xiaohongshu video downloads
type XiaohongshuExtractor struct{}

func (e *XiaohongshuExtractor) Name() string {
	return "xiaohongshu"
}

func (e *XiaohongshuExtractor) Match(u *url.URL) bool {
	// Host matching is done by registry, this is called after host match
	return true
}

func (e *XiaohongshuExtractor) Extract(url string) (Media, error) {
	return nil, fmt.Errorf("Xiaohongshu support coming soon")
}

func init() {
	Register(&XiaohongshuExtractor{},
		"xiaohongshu.com",
		"xhslink.com",
	)
}
