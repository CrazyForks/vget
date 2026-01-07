package gdrive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/guiyumin/vget/internal/core/config"
)

const (
	// Google Drive API base URL
	driveAPIBase = "https://www.googleapis.com/drive/v3"

	// Google OAuth token URL
	tokenURL = "https://oauth2.googleapis.com/token"
)

// Client is a Google Drive API client
type Client struct {
	accessToken  string // in-memory only, fetched on first use
	refreshToken string
	httpClient   *http.Client
}

// FileInfo contains information about a Drive file
type FileInfo struct {
	ID       string
	Name     string
	Path     string
	Size     int64
	IsDir    bool
	MimeType string
}

// driveFile represents a file in Google Drive API response
type driveFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Size     string `json:"size"`
	Parents  []string `json:"parents"`
}

// driveFileList represents a list of files from Google Drive API
type driveFileList struct {
	Files         []driveFile `json:"files"`
	NextPageToken string      `json:"nextPageToken"`
}

// tokenResponse represents OAuth token refresh response
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// NewClient creates a new Google Drive client from config
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.Google.RefreshToken == "" {
		return nil, fmt.Errorf("google drive not connected, run 'vget login google' first")
	}

	return &Client{
		refreshToken: cfg.Google.RefreshToken,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// refreshAccessToken gets a fresh access token using the refresh token
func (c *Client) refreshAccessToken() error {
	// Get client credentials from environment or use vget.io as proxy
	clientID := getEnvOrDefault("GOOGLE_CLIENT_ID", "")
	clientSecret := getEnvOrDefault("GOOGLE_CLIENT_SECRET", "")

	// If no local credentials, use vget.io token refresh endpoint
	if clientID == "" || clientSecret == "" {
		return c.refreshViaVgetIO()
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", c.refreshToken)
	data.Set("grant_type", "refresh_token")

	resp, err := c.httpClient.PostForm(tokenURL, data)
	if err != nil {
		return fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = token.AccessToken
	return nil
}

// refreshViaVgetIO refreshes the token via vget.io proxy
func (c *Client) refreshViaVgetIO() error {
	req, err := http.NewRequest("POST", "https://vget.io/api/auth/google/refresh", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.refreshToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = token.AccessToken
	return nil
}

// doRequest makes an authenticated request to Google Drive API
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	// Get token on first use
	if c.accessToken == "" {
		if err := c.refreshAccessToken(); err != nil {
			return nil, err
		}
	}

	resp, err := c.doRequestWithToken(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	// If 401, refresh token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.refreshAccessToken(); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
		return c.doRequestWithToken(ctx, method, endpoint, body)
	}

	return resp, nil
}

// doRequestWithToken makes a request with the current access token
func (c *Client) doRequestWithToken(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, driveAPIBase+endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// List returns files in a folder
func (c *Client) List(ctx context.Context, folderPath string) ([]FileInfo, error) {
	// Resolve folder path to folder ID
	folderID, err := c.resolvePath(ctx, folderPath)
	if err != nil {
		return nil, err
	}

	// Build query
	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	fields := "files(id,name,mimeType,size,parents),nextPageToken"

	endpoint := fmt.Sprintf("/files?q=%s&fields=%s&pageSize=1000",
		url.QueryEscape(query),
		url.QueryEscape(fields))

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list files: %s", string(body))
	}

	var fileList driveFileList
	if err := json.NewDecoder(resp.Body).Decode(&fileList); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]FileInfo, 0, len(fileList.Files))
	for _, f := range fileList.Files {
		var size int64
		if f.Size != "" {
			fmt.Sscanf(f.Size, "%d", &size)
		}

		result = append(result, FileInfo{
			ID:       f.ID,
			Name:     f.Name,
			Path:     joinPath(folderPath, f.Name),
			Size:     size,
			IsDir:    f.MimeType == "application/vnd.google-apps.folder",
			MimeType: f.MimeType,
		})
	}

	return result, nil
}

// Stat returns information about a file or folder
func (c *Client) Stat(ctx context.Context, path string) (*FileInfo, error) {
	fileID, err := c.resolvePath(ctx, path)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/files/%s?fields=id,name,mimeType,size", fileID)
	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to stat file: %s", string(body))
	}

	var f driveFile
	if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
		return nil, err
	}

	var size int64
	if f.Size != "" {
		fmt.Sscanf(f.Size, "%d", &size)
	}

	return &FileInfo{
		ID:       f.ID,
		Name:     f.Name,
		Path:     path,
		Size:     size,
		IsDir:    f.MimeType == "application/vnd.google-apps.folder",
		MimeType: f.MimeType,
	}, nil
}

// resolvePath resolves a path like "/folder/subfolder" to a file ID
func (c *Client) resolvePath(ctx context.Context, path string) (string, error) {
	// Normalize path
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "root", nil
	}

	parts := strings.Split(path, "/")
	currentID := "root"

	for _, name := range parts {
		if name == "" {
			continue
		}

		// Search for file with this name in current folder
		query := fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false",
			escapeQuery(name), currentID)
		endpoint := fmt.Sprintf("/files?q=%s&fields=files(id,mimeType)&pageSize=1",
			url.QueryEscape(query))

		resp, err := c.doRequest(ctx, "GET", endpoint, nil)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}

		var fileList driveFileList
		if err := json.NewDecoder(resp.Body).Decode(&fileList); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("failed to parse response: %w", err)
		}
		resp.Body.Close()

		if len(fileList.Files) == 0 {
			return "", fmt.Errorf("path not found: %s", path)
		}

		currentID = fileList.Files[0].ID
	}

	return currentID, nil
}

// IsGDrivePath checks if a path is a Google Drive path (gdrive:/path)
func IsGDrivePath(path string) bool {
	return strings.HasPrefix(path, "gdrive:") ||
		strings.HasPrefix(path, "gdrive/") ||
		strings.HasPrefix(path, "drive:")
}

// ParseGDrivePath extracts the path from a gdrive: URL
func ParseGDrivePath(remotePath string) (string, error) {
	// Handle gdrive:/path, gdrive:path, drive:/path formats
	for _, prefix := range []string{"gdrive:", "drive:"} {
		if path, found := strings.CutPrefix(remotePath, prefix); found {
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			return path, nil
		}
	}
	return "", fmt.Errorf("invalid Google Drive path: %s", remotePath)
}

// escapeQuery escapes special characters in Drive query strings
func escapeQuery(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}

// joinPath joins path components
func joinPath(base, name string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return "/" + name
	}
	return base + "/" + name
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Google Docs MIME type mappings for export
var googleDocsExportFormats = map[string]struct {
	MimeType  string
	Extension string
}{
	"application/vnd.google-apps.document":     {"application/pdf", "pdf"},
	"application/vnd.google-apps.spreadsheet":  {"application/pdf", "pdf"},
	"application/vnd.google-apps.presentation": {"application/pdf", "pdf"},
	"application/vnd.google-apps.drawing":      {"application/pdf", "pdf"},
}

// IsGoogleDoc checks if a file is a Google Docs/Sheets/Slides file
func IsGoogleDoc(mimeType string) bool {
	_, ok := googleDocsExportFormats[mimeType]
	return ok
}

// GetDownloadURL returns the download URL for a file
// For Google Docs, it returns an export URL; for regular files, a direct download URL
func (c *Client) GetDownloadURL(fileID, mimeType string) (string, error) {
	// Ensure we have a valid token
	if c.accessToken == "" {
		if err := c.refreshAccessToken(); err != nil {
			return "", err
		}
	}

	if export, ok := googleDocsExportFormats[mimeType]; ok {
		// Google Docs need to be exported
		return fmt.Sprintf("%s/files/%s/export?mimeType=%s",
			driveAPIBase, fileID, url.QueryEscape(export.MimeType)), nil
	}

	// Regular files can be downloaded directly
	return fmt.Sprintf("%s/files/%s?alt=media", driveAPIBase, fileID), nil
}

// GetAuthHeader returns the Authorization header for download requests
func (c *Client) GetAuthHeader() (string, error) {
	if c.accessToken == "" {
		if err := c.refreshAccessToken(); err != nil {
			return "", err
		}
	}
	return "Bearer " + c.accessToken, nil
}

// Download downloads a file and returns a reader
func (c *Client) Download(ctx context.Context, fileID, mimeType string) (io.ReadCloser, error) {
	downloadURL, err := c.GetDownloadURL(fileID, mimeType)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Handle 401 - refresh and retry
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.refreshAccessToken(); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: %s", string(body))
	}

	return resp.Body, nil
}
