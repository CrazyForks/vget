package bilibili

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/guiyumin/vget/internal/core/config"
)

// Auth handles Bilibili authentication via QR code or cookie
type Auth struct {
	client *http.Client
}

// QRSession holds the QR code login session data
type QRSession struct {
	URL       string // QR code content URL (to be encoded as QR)
	QRCodeKey string // Key for polling status
}

// QRStatus represents the status of QR code login
type QRStatus int

const (
	QRWaiting   QRStatus = 86101 // Not scanned yet
	QRScanned   QRStatus = 86090 // Scanned, waiting for confirmation
	QRExpired   QRStatus = 86038 // QR code expired
	QRConfirmed QRStatus = 0     // Login successful
)

// Credentials stores the login credentials
type Credentials struct {
	SESSDATA   string
	BiliJCT    string
	DedeUserID string
}

// NewAuth creates a new Auth instance
func NewAuth() *Auth {
	return &Auth{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateQRCode requests a new QR code for login
func (a *Auth) GenerateQRCode() (*QRSession, error) {
	api := "https://passport.bilibili.com/x/passport-login/web/qrcode/generate?source=main-fe-header"

	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			URL       string `json:"url"`
			QRCodeKey string `json:"qrcode_key"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", result.Message, result.Code)
	}

	return &QRSession{
		URL:       result.Data.URL,
		QRCodeKey: result.Data.QRCodeKey,
	}, nil
}

// PollQRStatus checks the status of QR code login
// Returns the status code, credentials (on success), and any error
func (a *Auth) PollQRStatus(qrcodeKey string) (QRStatus, *Credentials, error) {
	api := fmt.Sprintf("https://passport.bilibili.com/x/passport-login/web/qrcode/poll?qrcode_key=%s&source=main-fe-header",
		url.QueryEscape(qrcodeKey))

	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return 0, nil, err
	}
	a.setHeaders(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			URL          string `json:"url"`
			RefreshToken string `json:"refresh_token"`
			Timestamp    int64  `json:"timestamp"`
			Code         int    `json:"code"`
			Message      string `json:"message"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	status := QRStatus(result.Data.Code)

	// If login confirmed, extract credentials from the URL
	if status == QRConfirmed && result.Data.URL != "" {
		creds, err := a.parseCredentialsFromURL(result.Data.URL)
		if err != nil {
			return status, nil, fmt.Errorf("failed to parse credentials: %w", err)
		}
		return status, creds, nil
	}

	return status, nil, nil
}

// parseCredentialsFromURL extracts SESSDATA, bili_jct, DedeUserID from the callback URL
func (a *Auth) parseCredentialsFromURL(urlStr string) (*Credentials, error) {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	query := parsed.Query()
	creds := &Credentials{
		SESSDATA:   query.Get("SESSDATA"),
		BiliJCT:    query.Get("bili_jct"),
		DedeUserID: query.Get("DedeUserID"),
	}

	if creds.SESSDATA == "" {
		return nil, fmt.Errorf("SESSDATA not found in response")
	}

	return creds, nil
}

// SaveCredentials saves credentials to config file
func (a *Auth) SaveCredentials(creds *Credentials) error {
	cfg := config.LoadOrDefault()
	cfg.Bilibili.Cookie = creds.ToCookieString()
	return config.Save(cfg)
}

// ToCookieString converts credentials to cookie format
func (c *Credentials) ToCookieString() string {
	return fmt.Sprintf("SESSDATA=%s; bili_jct=%s; DedeUserID=%s",
		c.SESSDATA, c.BiliJCT, c.DedeUserID)
}

// LoadCredentials loads saved credentials from config
func (a *Auth) LoadCredentials() *Credentials {
	cfg := config.LoadOrDefault()
	if cfg.Bilibili.Cookie == "" {
		return nil
	}

	return ParseCookieString(cfg.Bilibili.Cookie)
}

// ParseCookieString parses a cookie string into credentials
func ParseCookieString(cookie string) *Credentials {
	creds := &Credentials{}

	for part := range strings.SplitSeq(cookie, ";") {
		part = strings.TrimSpace(part)
		if val, ok := strings.CutPrefix(part, "SESSDATA="); ok {
			creds.SESSDATA = val
		} else if val, ok := strings.CutPrefix(part, "bili_jct="); ok {
			creds.BiliJCT = val
		} else if val, ok := strings.CutPrefix(part, "DedeUserID="); ok {
			creds.DedeUserID = val
		}
	}

	return creds
}

// ValidateCredentials checks if credentials are valid by calling user info API
func (a *Auth) ValidateCredentials(creds *Credentials) (string, error) {
	api := "https://api.bilibili.com/x/web-interface/nav"

	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return "", err
	}
	a.setHeaders(req)
	req.Header.Set("Cookie", creds.ToCookieString())

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			IsLogin bool   `json:"isLogin"`
			UName   string `json:"uname"`
			Mid     int64  `json:"mid"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Code != 0 {
		return "", fmt.Errorf("API error: %s (code: %d)", result.Message, result.Code)
	}

	if !result.Data.IsLogin {
		return "", fmt.Errorf("credentials are invalid or expired")
	}

	return result.Data.UName, nil
}

// String returns a human-readable status string
func (s QRStatus) String() string {
	switch s {
	case QRWaiting:
		return "等待扫码"
	case QRScanned:
		return "扫码成功，请在手机上确认"
	case QRExpired:
		return "二维码已过期"
	case QRConfirmed:
		return "登录成功"
	default:
		return fmt.Sprintf("未知状态: %d", s)
	}
}

// setHeaders sets common request headers
func (a *Auth) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("Accept", "application/json")
}
