package login

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guiyumin/vget/internal/core/config"
	"github.com/spf13/cobra"
)

// Google OAuth configuration
// Users should set up their own OAuth app at https://console.cloud.google.com
// and configure these via environment or the web auth flow at vget.io
const (
	defaultAuthURL = "https://vget.io/api/auth/google"
	localPort      = 9876 // Local callback port
)

// Google styles
var (
	googleTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#4285F4")) // Google blue

	googleStepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	googleKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4285F4")).
			Bold(true)

	googleHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	googleSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	googleErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))
)

// GoogleCmd returns the google login command
func GoogleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "google",
		Aliases: []string{"gdrive", "drive"},
		Short:   "Connect Google Drive",
		Long: `Connect your Google Drive account to vget.

This opens a browser window where you'll sign in with Google and authorize vget
to access your Drive.

After authorization, you can:

  1. List files in Google Drive:
     vget ls gdrive:/folder

  2. Download files from Google Drive:
     vget gdrive:/folder/video.mp4

  3. Download to Google Drive:
     vget <url> --output gdrive:/folder/video.mp4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGoogleAuth()
		},
	}

	cmd.AddCommand(googleStatusCmd())
	cmd.AddCommand(googleManualCmd())

	return cmd
}

// GoogleLogoutCmd returns the google logout command
func GoogleLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "google",
		Short: "Disconnect Google Drive",
		Long: `Remove Google Drive connection and clear stored tokens.

Note: This only removes the tokens from vget. To fully revoke access,
visit https://myaccount.google.com/permissions and remove vget.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.LoadOrDefault()
			email := cfg.Google.Email

			cfg.Google = config.GoogleConfig{}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			if email != "" {
				fmt.Printf("✓ Google Drive disconnected (%s)\n", email)
			} else {
				fmt.Println("✓ Google Drive credentials cleared")
			}
			fmt.Println("\nTo fully revoke access, visit:")
			fmt.Println("  https://myaccount.google.com/permissions")
			return nil
		},
	}
}

func googleStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check Google Drive connection status",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.LoadOrDefault()
			if cfg.Google.RefreshToken != "" {
				fmt.Printf("✓ Google Drive: connected (%s)\n", cfg.Google.Email)
			} else {
				fmt.Println("✗ Google Drive: not connected")
				fmt.Println("  Run 'vget login google' to connect")
			}
		},
	}
}

func googleManualCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "manual",
		Short: "Manually enter Google OAuth token",
		Long: `Manually enter a Google OAuth token JSON.

Use this if the automatic browser flow doesn't work (e.g., on headless servers).

Steps:
  1. Open https://vget.io/api/auth/google?returnTo=cli in a browser
  2. Complete the Google sign-in
  3. Copy the JSON token displayed
  4. Paste it when prompted`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runManualGoogleAuth()
		},
	}
}

// Token response from OAuth callback
type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Email        string `json:"email"`
}

// TUI model for Google OAuth flow
type googleAuthModel struct {
	state     googleAuthState
	spinner   spinner.Model
	email     string
	error     string
	cancelled bool
	server    *http.Server
	tokenCh   chan *googleTokenResponse
	errCh     chan error
}

type googleAuthState int

const (
	googleStateStarting googleAuthState = iota
	googleStateWaiting
	googleStateSuccess
	googleStateError
)

type googleTokenMsg struct {
	token *googleTokenResponse
	err   error
}

func newGoogleAuthModel() googleAuthModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#4285F4"))

	return googleAuthModel{
		state:   googleStateStarting,
		spinner: s,
		tokenCh: make(chan *googleTokenResponse, 1),
		errCh:   make(chan error, 1),
	}
}

func (m googleAuthModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.startAuthFlow,
	)
}

func (m googleAuthModel) startAuthFlow() tea.Msg {
	// Start local callback server
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		return googleTokenMsg{err: fmt.Errorf("failed to start callback server: %w", err)}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Parse token from query params or POST body
		tokenJSON := r.URL.Query().Get("token")
		if tokenJSON == "" {
			// Try to read from POST body
			if r.Method == http.MethodPost {
				var token googleTokenResponse
				if err := json.NewDecoder(r.Body).Decode(&token); err == nil {
					m.tokenCh <- &token
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprint(w, successHTML)
					return
				}
			}
			m.errCh <- fmt.Errorf("no token received")
			http.Error(w, "No token received", http.StatusBadRequest)
			return
		}

		var token googleTokenResponse
		if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
			m.errCh <- fmt.Errorf("invalid token: %w", err)
			http.Error(w, "Invalid token", http.StatusBadRequest)
			return
		}

		m.tokenCh <- &token
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, successHTML)
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			m.errCh <- err
		}
	}()

	// Open browser to auth URL
	authURL := fmt.Sprintf("%s?returnTo=http://127.0.0.1:%d/callback", defaultAuthURL, localPort)
	if err := openBrowser(authURL); err != nil {
		// If browser fails, show URL for manual opening
		return googleTokenMsg{err: fmt.Errorf("failed to open browser: %w\n\nPlease open this URL manually:\n%s", err, authURL)}
	}

	return nil
}

func (m googleAuthModel) waitForToken() tea.Cmd {
	return func() tea.Msg {
		select {
		case token := <-m.tokenCh:
			return googleTokenMsg{token: token}
		case err := <-m.errCh:
			return googleTokenMsg{err: err}
		case <-time.After(5 * time.Minute):
			return googleTokenMsg{err: fmt.Errorf("authentication timed out")}
		}
	}
}

func (m googleAuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			if m.server != nil {
				m.server.Shutdown(context.Background())
			}
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case googleTokenMsg:
		if msg.err != nil {
			m.state = googleStateError
			m.error = msg.err.Error()
			return m, nil
		}

		if msg.token != nil {
			// Save only refresh_token and email to config (access_token is short-lived)
			cfg := config.LoadOrDefault()
			cfg.Google.RefreshToken = msg.token.RefreshToken
			cfg.Google.Email = msg.token.Email

			if err := config.Save(cfg); err != nil {
				m.state = googleStateError
				m.error = fmt.Sprintf("failed to save config: %v", err)
				return m, nil
			}

			m.state = googleStateSuccess
			m.email = msg.token.Email
			if m.server != nil {
				m.server.Shutdown(context.Background())
			}
			return m, tea.Quit
		}

		// No token yet, keep waiting
		m.state = googleStateWaiting
		return m, m.waitForToken()
	}

	// If we just started, begin waiting for token
	if m.state == googleStateStarting {
		m.state = googleStateWaiting
		return m, m.waitForToken()
	}

	return m, nil
}

func (m googleAuthModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(googleTitleStyle.Render("  ━━━ Google Drive Authorization ━━━"))
	b.WriteString("\n\n")

	switch m.state {
	case googleStateStarting:
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" Starting authorization...\n")

	case googleStateWaiting:
		b.WriteString(googleStepStyle.Render("  A browser window has opened."))
		b.WriteString("\n")
		b.WriteString(googleStepStyle.Render("  Please sign in with Google and authorize vget."))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" Waiting for authorization...\n")

	case googleStateSuccess:
		b.WriteString(googleSuccessStyle.Render("  ✓ Google Drive connected!"))
		b.WriteString("\n")
		if m.email != "" {
			b.WriteString(googleStepStyle.Render(fmt.Sprintf("  Account: %s", m.email)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(googleStepStyle.Render("  You can now:"))
		b.WriteString("\n\n")
		b.WriteString(googleStepStyle.Render("  1. List files in Google Drive:"))
		b.WriteString("\n")
		b.WriteString(googleKeyStyle.Render("     vget ls gdrive:/folder"))
		b.WriteString("\n\n")
		b.WriteString(googleStepStyle.Render("  2. Download files from Google Drive:"))
		b.WriteString("\n")
		b.WriteString(googleKeyStyle.Render("     vget gdrive:/folder/video.mp4"))
		b.WriteString("\n\n")
		b.WriteString(googleStepStyle.Render("  3. Download to Google Drive:"))
		b.WriteString("\n")
		b.WriteString(googleKeyStyle.Render("     vget <url> --output gdrive:/folder/video.mp4"))
		b.WriteString("\n")

	case googleStateError:
		b.WriteString(googleErrorStyle.Render("  ✗ Error: " + m.error))
		b.WriteString("\n\n")
		b.WriteString(googleHelpStyle.Render("  Try again with: vget login google"))
		b.WriteString("\n")
		b.WriteString(googleHelpStyle.Render("  Or use manual mode: vget login google manual"))
		b.WriteString("\n")
	}

	if m.state != googleStateSuccess && m.state != googleStateError {
		b.WriteString("\n")
		b.WriteString(googleHelpStyle.Render("  Press q or Esc to cancel"))
		b.WriteString("\n")
	}

	return b.String()
}

func runGoogleAuth() error {
	m := newGoogleAuthModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	result := finalModel.(googleAuthModel)
	if result.cancelled {
		fmt.Println("  Cancelled")
		return nil
	}

	return nil
}

func runManualGoogleAuth() error {
	fmt.Println()
	fmt.Println(googleTitleStyle.Render("  ━━━ Manual Google Drive Authorization ━━━"))
	fmt.Println()
	fmt.Println(googleStepStyle.Render("  1. Open this URL in a browser:"))
	fmt.Println()
	fmt.Println(googleKeyStyle.Render("     " + defaultAuthURL + "?returnTo=cli"))
	fmt.Println()
	fmt.Println(googleStepStyle.Render("  2. Sign in with Google and authorize vget"))
	fmt.Println(googleStepStyle.Render("  3. Copy the JSON token displayed"))
	fmt.Println(googleStepStyle.Render("  4. Paste it below and press Enter:"))
	fmt.Println()
	fmt.Print("  Token: ")

	var tokenJSON string
	fmt.Scanln(&tokenJSON)

	if tokenJSON == "" {
		fmt.Println(googleErrorStyle.Render("  ✗ No token provided"))
		return nil
	}

	var token googleTokenResponse
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		fmt.Println(googleErrorStyle.Render("  ✗ Invalid token format"))
		return nil
	}

	cfg := config.LoadOrDefault()
	cfg.Google.RefreshToken = token.RefreshToken
	cfg.Google.Email = token.Email

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Println(googleSuccessStyle.Render("  ✓ Google Drive connected!"))
	if token.Email != "" {
		fmt.Printf("  Account: %s\n", token.Email)
	}
	fmt.Println()
	fmt.Println(googleStepStyle.Render("  You can now:"))
	fmt.Println()
	fmt.Println(googleStepStyle.Render("  1. List files in Google Drive:"))
	fmt.Println(googleKeyStyle.Render("     vget ls gdrive:/folder"))
	fmt.Println()
	fmt.Println(googleStepStyle.Render("  2. Download files from Google Drive:"))
	fmt.Println(googleKeyStyle.Render("     vget gdrive:/folder/video.mp4"))
	fmt.Println()
	fmt.Println(googleStepStyle.Render("  3. Download to Google Drive:"))
	fmt.Println(googleKeyStyle.Render("     vget <url> --output gdrive:/folder/video.mp4"))

	return nil
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

const successHTML = `<!DOCTYPE html>
<html>
<head>
  <title>vget - Authorization Successful</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 100vh;
      margin: 0;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    }
    .container {
      background: white;
      padding: 40px 60px;
      border-radius: 16px;
      box-shadow: 0 20px 60px rgba(0,0,0,0.3);
      text-align: center;
    }
    .check {
      width: 80px;
      height: 80px;
      background: #10b981;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 0 auto 20px;
    }
    .check svg {
      width: 40px;
      height: 40px;
      stroke: white;
      stroke-width: 3;
    }
    h1 {
      color: #1f2937;
      margin: 0 0 10px;
      font-size: 24px;
    }
    p {
      color: #6b7280;
      margin: 0;
      font-size: 16px;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="check">
      <svg viewBox="0 0 24 24" fill="none">
        <path d="M5 13l4 4L19 7" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    </div>
    <h1>Authorization Successful!</h1>
    <p>You can close this window and return to vget.</p>
  </div>
</body>
</html>`
