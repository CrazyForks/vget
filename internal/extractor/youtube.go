package extractor

import (
	"net/url"

	"github.com/guiyumin/vget/internal/extractor/youtube"
)

// VisibleSetter is implemented by extractors that support visible mode
type VisibleSetter interface {
	SetVisible(visible bool)
}

// youtubeWrapper wraps the youtube.Extractor to implement the Extractor interface
type youtubeWrapper struct {
	ext *youtube.Extractor
}

func (w *youtubeWrapper) Name() string {
	return w.ext.Name()
}

func (w *youtubeWrapper) Match(u *url.URL) bool {
	return w.ext.Match(u)
}

func (w *youtubeWrapper) Extract(urlStr string) (Media, error) {
	media, err := w.ext.Extract(urlStr)
	if err != nil {
		return nil, err
	}

	// Convert youtube.VideoMedia to extractor.VideoMedia
	formats := make([]VideoFormat, len(media.Formats))
	for i, f := range media.Formats {
		formats[i] = VideoFormat{
			URL:      f.URL,
			AudioURL: f.AudioURL,
			Quality:  f.Quality,
			Ext:      f.Ext,
			Width:    f.Width,
			Height:   f.Height,
			Bitrate:  f.Bitrate,
			Headers:  f.Headers,
		}
	}

	return &VideoMedia{
		ID:        media.ID,
		Title:     media.Title,
		Uploader:  media.Uploader,
		Thumbnail: media.Thumbnail,
		Formats:   formats,
	}, nil
}

// SetVisible passes through to the underlying extractor
func (w *youtubeWrapper) SetVisible(visible bool) {
	w.ext.SetVisible(visible)
}

func init() {
	Register(&youtubeWrapper{ext: &youtube.Extractor{}},
		"youtube.com",
		"www.youtube.com",
		"youtu.be",
		"m.youtube.com",
		"music.youtube.com",
	)
}
