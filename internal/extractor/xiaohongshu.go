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
	host := u.Hostname()
	return host == "xiaohongshu.com" || host == "www.xiaohongshu.com" ||
		host == "xhslink.com" || host == "www.xhslink.com"
}

func (e *XiaohongshuExtractor) Extract(url string) (Media, error) {
	return nil, fmt.Errorf("Xiaohongshu support coming soon")
}

func init() {
	Register(&XiaohongshuExtractor{})
}
