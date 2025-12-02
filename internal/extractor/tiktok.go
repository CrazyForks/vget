package extractor

import (
	"fmt"
	"net/url"
)

// TikTokExtractor handles TikTok video downloads
type TikTokExtractor struct{}

func (e *TikTokExtractor) Name() string {
	return "tiktok"
}

func (e *TikTokExtractor) Match(u *url.URL) bool {
	host := u.Hostname()
	return host == "tiktok.com" || host == "www.tiktok.com" || host == "vm.tiktok.com"
}

func (e *TikTokExtractor) Extract(url string) (Media, error) {
	return nil, fmt.Errorf("TikTok support coming soon")
}

func init() {
	Register(&TikTokExtractor{})
}
