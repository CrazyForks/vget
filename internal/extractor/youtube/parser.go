package youtube

import (
	"fmt"
	"strings"
)

func (e *Extractor) parseResponse(resp *InnertubeResponse, session *Session) (*VideoMedia, error) {
	var formats []VideoFormat

	// Build headers including cookies for authenticated downloads
	youtubeHeaders := map[string]string{
		"User-Agent": iosUserAgent,
		"Referer":    "https://www.youtube.com/",
		"Origin":     "https://www.youtube.com",
	}

	// Add authentication headers for downloads
	if session.VisitorData != "" {
		youtubeHeaders["X-Goog-Visitor-Id"] = session.VisitorData
	}
	if cookieStr := e.buildCookieString(session.Cookies); cookieStr != "" {
		youtubeHeaders["Cookie"] = cookieStr
	}

	// Find best audio tracks (one for mp4, one for webm)
	var bestMP4Audio, bestWebMAudio string
	var bestMP4Bitrate, bestWebMBitrate int

	for _, f := range resp.StreamingData.AdaptiveFormats {
		if f.URL == "" || !strings.Contains(f.MimeType, "audio/") {
			continue
		}

		if strings.Contains(f.MimeType, "mp4a") && f.Bitrate > bestMP4Bitrate {
			bestMP4Audio = f.URL
			bestMP4Bitrate = f.Bitrate
		}
		if strings.Contains(f.MimeType, "opus") && f.Bitrate > bestWebMBitrate {
			bestWebMAudio = f.URL
			bestWebMBitrate = f.Bitrate
		}
	}

	// First try HLS manifest (has both video + audio)
	if resp.StreamingData.HLSManifestURL != "" {
		formats = append(formats, VideoFormat{
			URL:     resp.StreamingData.HLSManifestURL,
			Quality: "auto (HLS)",
			Ext:     "m3u8",
			Headers: youtubeHeaders,
		})
	}

	// Add combined formats (video + audio together)
	for _, f := range resp.StreamingData.Formats {
		if f.URL == "" {
			continue
		}

		ext := "mp4"
		if strings.Contains(f.MimeType, "webm") {
			ext = "webm"
		}

		formats = append(formats, VideoFormat{
			URL:     f.URL,
			Quality: f.QualityLabel,
			Ext:     ext,
			Width:   f.Width,
			Height:  f.Height,
			Bitrate: f.Bitrate,
			Headers: youtubeHeaders,
		})
	}

	// Add adaptive formats with paired audio
	for _, f := range resp.StreamingData.AdaptiveFormats {
		if f.URL == "" {
			continue
		}

		// Only include video formats
		if !strings.Contains(f.MimeType, "video/") {
			continue
		}

		ext := "mp4"
		audioURL := bestMP4Audio
		if strings.Contains(f.MimeType, "webm") {
			ext = "webm"
			audioURL = bestWebMAudio
		}

		quality := f.QualityLabel
		if quality == "" {
			quality = fmt.Sprintf("%dp", f.Height)
		}

		qualityLabel := quality
		if audioURL != "" {
			qualityLabel = quality + " (needs merge)"
		} else {
			qualityLabel = quality + " (no audio)"
		}

		formats = append(formats, VideoFormat{
			URL:      f.URL,
			AudioURL: audioURL,
			Quality:  qualityLabel,
			Ext:      ext,
			Width:    f.Width,
			Height:   f.Height,
			Bitrate:  f.Bitrate,
			Headers:  youtubeHeaders,
		})
	}

	if len(formats) == 0 {
		return nil, fmt.Errorf("no downloadable formats found (all may require cipher decryption)")
	}

	// Get thumbnail
	var thumbnail string
	if len(resp.VideoDetails.Thumbnail.Thumbnails) > 0 {
		thumbnail = resp.VideoDetails.Thumbnail.Thumbnails[len(resp.VideoDetails.Thumbnail.Thumbnails)-1].URL
	}

	return &VideoMedia{
		ID:        resp.VideoDetails.VideoID,
		Title:     resp.VideoDetails.Title,
		Uploader:  resp.VideoDetails.Author,
		Thumbnail: thumbnail,
		Formats:   formats,
	}, nil
}
