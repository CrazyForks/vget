package youtube

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/guiyumin/vget/internal/config"
)

const sessionTTL = 6 * time.Hour

// Session holds the extracted tokens needed for Innertube API
type Session struct {
	POToken            string                 `json:"poToken"`
	VisitorData        string                 `json:"visitorData"`
	Cookies            []*proto.NetworkCookie `json:"cookies,omitempty"`
	ClientVersion      string                 `json:"clientVersion,omitempty"`
	SignatureTimestamp int                    `json:"signatureTimestamp,omitempty"`
	Timestamp          int64                  `json:"timestamp"`
}

// getSession returns a valid session, either from cache or by extracting new tokens
func (e *Extractor) getSession(videoID string) (*Session, error) {
	// Try to load cached session first
	if cached := e.loadSession(); cached != nil {
		return cached, nil
	}

	// Extract new session via browser
	return e.extractSessionTokens(videoID)
}

func (e *Extractor) loadSession() *Session {
	configDir, err := config.ConfigDir()
	if err != nil {
		return nil
	}

	sessionPath := filepath.Join(configDir, "youtube_session.json")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil
	}

	// Check if session is expired
	if session.Timestamp == 0 {
		return nil // Old session format without timestamp
	}
	sessionAge := time.Since(time.Unix(session.Timestamp, 0))
	if sessionAge > sessionTTL {
		fmt.Printf("Cached session expired (%.1f hours old)\n", sessionAge.Hours())
		return nil
	}

	// Verify required fields
	if session.VisitorData == "" {
		return nil
	}

	fmt.Printf("Using cached session (%.1f hours old)\n", sessionAge.Hours())
	return &session
}

func (e *Extractor) saveSession(session *Session) {
	configDir, err := config.ConfigDir()
	if err != nil {
		return
	}

	sessionToSave := Session{
		POToken:            session.POToken,
		VisitorData:        session.VisitorData,
		Cookies:            session.Cookies,
		ClientVersion:      session.ClientVersion,
		SignatureTimestamp: session.SignatureTimestamp,
		Timestamp:          time.Now().Unix(),
	}

	data, err := json.MarshalIndent(sessionToSave, "", "  ")
	if err != nil {
		return
	}

	sessionPath := filepath.Join(configDir, "youtube_session.json")
	_ = os.WriteFile(sessionPath, data, 0600)
}

// buildCookieString builds a Cookie header value from captured cookies
func (e *Extractor) buildCookieString(cookies []*proto.NetworkCookie) string {
	if len(cookies) == 0 {
		return ""
	}
	var parts []string
	for _, c := range cookies {
		// Only include relevant YouTube cookies
		if strings.Contains(c.Domain, "youtube.com") || strings.Contains(c.Domain, "google.com") {
			parts = append(parts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
	}
	return strings.Join(parts, "; ")
}
