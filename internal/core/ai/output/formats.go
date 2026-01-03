package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/guiyumin/vget/internal/core/ai/transcriber"
)

// WriteSRT writes a transcription result to an SRT subtitle file.
func WriteSRT(outputPath string, result *transcriber.Result) error {
	var b strings.Builder

	if len(result.Segments) == 0 {
		// No segments, write raw text as single subtitle
		b.WriteString("1\n")
		b.WriteString("00:00:00,000 --> 00:00:30,000\n")
		b.WriteString(result.RawText)
		b.WriteString("\n")
	} else {
		for i, seg := range result.Segments {
			text := strings.TrimSpace(seg.Text)
			if text == "" {
				continue
			}

			// Subtitle number (1-indexed)
			b.WriteString(fmt.Sprintf("%d\n", i+1))

			// Time range: HH:MM:SS,mmm --> HH:MM:SS,mmm
			b.WriteString(fmt.Sprintf("%s --> %s\n",
				formatSRTTimestamp(seg.Start),
				formatSRTTimestamp(seg.End),
			))

			// Subtitle text
			b.WriteString(text)
			b.WriteString("\n\n")
		}
	}

	return os.WriteFile(outputPath, []byte(b.String()), 0644)
}

// WriteVTT writes a transcription result to a WebVTT subtitle file.
func WriteVTT(outputPath string, result *transcriber.Result) error {
	var b strings.Builder

	// WebVTT header
	b.WriteString("WEBVTT\n\n")

	if len(result.Segments) == 0 {
		// No segments, write raw text as single cue
		b.WriteString("00:00:00.000 --> 00:00:30.000\n")
		b.WriteString(result.RawText)
		b.WriteString("\n")
	} else {
		for i, seg := range result.Segments {
			text := strings.TrimSpace(seg.Text)
			if text == "" {
				continue
			}

			// Optional cue identifier
			b.WriteString(fmt.Sprintf("%d\n", i+1))

			// Time range: HH:MM:SS.mmm --> HH:MM:SS.mmm
			b.WriteString(fmt.Sprintf("%s --> %s\n",
				formatVTTTimestamp(seg.Start),
				formatVTTTimestamp(seg.End),
			))

			// Cue text
			b.WriteString(text)
			b.WriteString("\n\n")
		}
	}

	return os.WriteFile(outputPath, []byte(b.String()), 0644)
}

// WriteTranscriptWithFormat writes a transcription result to a file in the specified format.
// Supported formats: "md" (markdown), "srt", "vtt"
func WriteTranscriptWithFormat(basePath, sourcePath, format string, result *transcriber.Result) (string, error) {
	var outputPath string
	var err error

	switch format {
	case "srt":
		outputPath = basePath + ".srt"
		err = WriteSRT(outputPath, result)
	case "vtt":
		outputPath = basePath + ".vtt"
		err = WriteVTT(outputPath, result)
	case "md", "":
		outputPath = basePath + ".transcript.md"
		err = WriteTranscript(outputPath, sourcePath, result)
	default:
		// Default to markdown
		outputPath = basePath + ".transcript.md"
		err = WriteTranscript(outputPath, sourcePath, result)
	}

	if err != nil {
		return "", err
	}
	return outputPath, nil
}

// Note: formatSRTTimestamp and formatVTTTimestamp are defined in convert.go
