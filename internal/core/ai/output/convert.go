package output

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Segment represents a timestamped portion of transcript.
type Segment struct {
	Start time.Duration
	End   time.Duration
	Text  string
}

// ParseTranscript parses a markdown transcript into segments.
// Expected format: [HH:MM:SS] or [MM:SS] followed by text
func ParseTranscript(content string) ([]Segment, error) {
	var segments []Segment

	// Match timestamps like [00:00:00] or [00:00]
	timestampRe := regexp.MustCompile(`^\[(\d{1,2}:\d{2}(?::\d{2})?)\]\s*(.*)$`)

	lines := strings.Split(content, "\n")
	var currentSegment *Segment

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip markdown headers and metadata
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "**") || strings.HasPrefix(line, "---") {
			continue
		}

		matches := timestampRe.FindStringSubmatch(line)
		if len(matches) == 3 {
			// Found a new timestamp
			timestamp := parseTimestampString(matches[1])
			text := strings.TrimSpace(matches[2])

			// Close previous segment
			if currentSegment != nil && currentSegment.Text != "" {
				// Set end time to current start
				currentSegment.End = timestamp
				segments = append(segments, *currentSegment)
			}

			// Start new segment
			currentSegment = &Segment{
				Start: timestamp,
				Text:  text,
			}
		} else if currentSegment != nil {
			// Continuation of current segment
			if currentSegment.Text != "" {
				currentSegment.Text += " "
			}
			currentSegment.Text += line
		}
	}

	// Add final segment
	if currentSegment != nil && currentSegment.Text != "" {
		// Estimate end time (add 5 seconds to start)
		if currentSegment.End == 0 {
			currentSegment.End = currentSegment.Start + 5*time.Second
		}
		segments = append(segments, *currentSegment)
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("no timestamped segments found in transcript")
	}

	return segments, nil
}

// parseTimestampString parses HH:MM:SS or MM:SS format to duration.
func parseTimestampString(s string) time.Duration {
	parts := strings.Split(s, ":")
	var hours, minutes, seconds int

	switch len(parts) {
	case 2:
		fmt.Sscanf(parts[0], "%d", &minutes)
		fmt.Sscanf(parts[1], "%d", &seconds)
	case 3:
		fmt.Sscanf(parts[0], "%d", &hours)
		fmt.Sscanf(parts[1], "%d", &minutes)
		fmt.Sscanf(parts[2], "%d", &seconds)
	}

	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second
}

// ToSRT converts segments to SubRip (SRT) format.
func ToSRT(segments []Segment) string {
	var b strings.Builder

	for i, seg := range segments {
		// Sequence number
		b.WriteString(fmt.Sprintf("%d\n", i+1))

		// Timestamps: HH:MM:SS,mmm --> HH:MM:SS,mmm
		start := formatSRTTimestamp(seg.Start)
		end := formatSRTTimestamp(seg.End)
		b.WriteString(fmt.Sprintf("%s --> %s\n", start, end))

		// Text
		b.WriteString(seg.Text)
		b.WriteString("\n\n")
	}

	return b.String()
}

// formatSRTTimestamp formats duration as HH:MM:SS,mmm for SRT.
func formatSRTTimestamp(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := int(d.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}

// ToVTT converts segments to WebVTT format.
func ToVTT(segments []Segment) string {
	var b strings.Builder

	// VTT header
	b.WriteString("WEBVTT\n\n")

	for i, seg := range segments {
		// Optional cue identifier
		b.WriteString(fmt.Sprintf("%d\n", i+1))

		// Timestamps: HH:MM:SS.mmm --> HH:MM:SS.mmm
		start := formatVTTTimestamp(seg.Start)
		end := formatVTTTimestamp(seg.End)
		b.WriteString(fmt.Sprintf("%s --> %s\n", start, end))

		// Text
		b.WriteString(seg.Text)
		b.WriteString("\n\n")
	}

	return b.String()
}

// formatVTTTimestamp formats duration as HH:MM:SS.mmm for VTT.
func formatVTTTimestamp(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := int(d.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}

