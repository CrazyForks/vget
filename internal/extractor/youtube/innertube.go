package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/guiyumin/vget/internal/config"
)

// InnertubeResponse represents the /player API response
type InnertubeResponse struct {
	StreamingData struct {
		Formats []struct {
			ITag            int    `json:"itag"`
			URL             string `json:"url"`
			MimeType        string `json:"mimeType"`
			Bitrate         int    `json:"bitrate"`
			Width           int    `json:"width"`
			Height          int    `json:"height"`
			QualityLabel    string `json:"qualityLabel"`
			SignatureCipher string `json:"signatureCipher"`
		} `json:"formats"`
		AdaptiveFormats []struct {
			ITag            int    `json:"itag"`
			URL             string `json:"url"`
			MimeType        string `json:"mimeType"`
			Bitrate         int    `json:"bitrate"`
			Width           int    `json:"width"`
			Height          int    `json:"height"`
			QualityLabel    string `json:"qualityLabel"`
			SignatureCipher string `json:"signatureCipher"`
			ContentLength   string `json:"contentLength"`
		} `json:"adaptiveFormats"`
		HLSManifestURL string `json:"hlsManifestUrl"`
	} `json:"streamingData"`
	VideoDetails struct {
		VideoID          string `json:"videoId"`
		Title            string `json:"title"`
		LengthSeconds    string `json:"lengthSeconds"`
		Author           string `json:"author"`
		ShortDescription string `json:"shortDescription"`
		Thumbnail        struct {
			Thumbnails []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"thumbnails"`
		} `json:"thumbnail"`
	} `json:"videoDetails"`
	PlayabilityStatus struct {
		Status          string `json:"status"`
		Reason          string `json:"reason"`
		PlayableInEmbed bool   `json:"playableInEmbed"`
	} `json:"playabilityStatus"`
}

const (
	iosClientVersion = "20.11.6"
	iosUserAgent     = "com.google.ios.youtube/20.11.6 (iPhone16,2; U; CPU iOS 18_1_0 like Mac OS X;)"
	defaultSTS       = 20073
)

func (e *Extractor) callInnertubeAPI(videoID string, session *Session) (*InnertubeResponse, error) {
	// Use dynamic signatureTimestamp if available
	sts := session.SignatureTimestamp
	if sts == 0 {
		sts = defaultSTS
	}

	payload := map[string]any{
		"context": map[string]any{
			"client": map[string]any{
				"clientName":    "IOS",
				"clientVersion": iosClientVersion,
				"deviceMake":    "Apple",
				"deviceModel":   "iPhone16,2",
				"userAgent":     iosUserAgent,
				"osName":        "iOS",
				"osVersion":     "18.1.0.22B83",
				"hl":            "en",
				"gl":            "US",
				"visitorData":   session.VisitorData,
			},
		},
		"videoId": videoID,
		"playbackContext": map[string]any{
			"contentPlaybackContext": map[string]any{
				"signatureTimestamp": sts,
			},
		},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	if session.POToken != "" {
		payload["serviceIntegrityDimensions"] = map[string]any{
			"poToken": session.POToken,
		}
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := "https://www.youtube.com/youtubei/v1/player?prettyPrint=false"

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", iosUserAgent)
	req.Header.Set("X-Youtube-Client-Name", "5") // iOS client ID
	req.Header.Set("X-Youtube-Client-Version", iosClientVersion)

	// Add authentication headers from session
	if session.VisitorData != "" {
		req.Header.Set("X-Goog-Visitor-Id", session.VisitorData)
	}
	if cookieStr := e.buildCookieString(session.Cookies); cookieStr != "" {
		req.Header.Set("Cookie", cookieStr)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	e.saveDebugResponse(body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response InnertubeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.PlayabilityStatus.Status != "OK" {
		return nil, fmt.Errorf("video not playable: %s - %s",
			response.PlayabilityStatus.Status,
			response.PlayabilityStatus.Reason)
	}

	return &response, nil
}

func (e *Extractor) saveDebugResponse(body []byte) {
	configDir, err := config.ConfigDir()
	if err != nil {
		return
	}

	debugPath := filepath.Join(configDir, "youtube_debug_response.json")
	_ = os.WriteFile(debugPath, body, 0644)
	fmt.Printf("Debug: saved API response to %s\n", debugPath)
}
