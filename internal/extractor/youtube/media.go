package youtube

// VideoFormat represents a downloadable video format
type VideoFormat struct {
	URL      string
	AudioURL string
	Quality  string
	Ext      string
	Width    int
	Height   int
	Bitrate  int
	Headers  map[string]string
}

// VideoMedia represents extracted video information
type VideoMedia struct {
	ID        string
	Title     string
	Uploader  string
	Thumbnail string
	Formats   []VideoFormat
}
