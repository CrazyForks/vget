package extractor

import (
	"fmt"
	"net/url"
)

// InstagramExtractor handles Instagram video downloads
type InstagramExtractor struct{}

func (e *InstagramExtractor) Name() string {
	return "instagram"
}

func (e *InstagramExtractor) Match(u *url.URL) bool {
	host := u.Hostname()
	return host == "instagram.com" || host == "www.instagram.com"
}

func (e *InstagramExtractor) Extract(url string) (Media, error) {
	return nil, fmt.Errorf("Instagram support coming soon")
}

func init() {
	Register(&InstagramExtractor{})
}
