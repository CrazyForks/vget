package login

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guiyumin/vget/internal/core/config"
	"github.com/guiyumin/vget/internal/core/site/bilibili"
	"github.com/spf13/cobra"
	qrcode "github.com/yeqown/go-qrcode/v2"
)

// Bilibili styles
var (
	biliTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00A1D6")) // Bilibili blue

	biliStepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	biliKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00A1D6")).
			Bold(true)

	biliHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	biliSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	biliErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

// BilibiliCmd returns the bilibili login command
func BilibiliCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bilibili",
		Short: "Login to Bilibili",
		Long:  "Login to Bilibili to download member-only or VIP content.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoginSelector()
		},
	}

	cmd.AddCommand(bilibiliQRCmd())
	cmd.AddCommand(bilibiliCookieCmd())
	cmd.AddCommand(bilibiliStatusCmd())

	return cmd
}

// BilibiliLogoutCmd returns the bilibili logout command
func BilibiliLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bilibili",
		Short: "Clear Bilibili credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.LoadOrDefault()
			cfg.Bilibili.Cookie = ""
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Println("✓ Bilibili credentials cleared")
			return nil
		},
	}
}

func bilibiliQRCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "qr",
		Short: "Login via QR code",
		Long:  "Login to Bilibili by scanning a QR code with the Bilibili mobile app.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQRLogin()
		},
	}
}

func bilibiliCookieCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cookie",
		Short: "Login via browser cookie",
		Long: `Login to Bilibili by pasting your cookie from browser.

To get your cookie:
  1. Open bilibili.com in browser and log in
  2. Press F12 to open DevTools
  3. Go to Application tab
  4. Find Cookies → bilibili.com
  5. Copy SESSDATA, bili_jct, DedeUserID values`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCookieLogin()
		},
	}
}

func bilibiliStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check Bilibili login status",
		Run: func(cmd *cobra.Command, args []string) {
			cfg := config.LoadOrDefault()
			if cfg.Bilibili.Cookie != "" && strings.Contains(cfg.Bilibili.Cookie, "SESSDATA") {
				fmt.Println("✓ Bilibili: logged in")
			} else {
				fmt.Println("✗ Bilibili: not logged in")
			}
		},
	}
}

// Login Method Selector TUI

type loginMethod int

const (
	methodQR loginMethod = iota
	methodCookie
)

type selectorModel struct {
	choices   []string
	cursor    int
	selected  loginMethod
	cancelled bool
}

func newSelectorModel() selectorModel {
	return selectorModel{
		choices: []string{
			"扫码登录",
			"Cookie 登录",
		},
		cursor: 0,
	}
}

func (m selectorModel) Init() tea.Cmd {
	return nil
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = loginMethod(m.cursor)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectorModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(biliTitleStyle.Render("  ━━━ Bilibili 登录 ━━━"))
	b.WriteString("\n\n")
	b.WriteString(biliStepStyle.Render("  请选择登录方式："))
	b.WriteString("\n\n")

	for i, choice := range m.choices {
		cursor := "  "
		style := biliStepStyle
		if m.cursor == i {
			cursor = biliKeyStyle.Render("▸ ")
			style = biliKeyStyle
		}
		b.WriteString("  ")
		b.WriteString(cursor)
		b.WriteString(style.Render(choice))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(biliHelpStyle.Render("  ↑/↓ 选择 • Enter 确认 • q 取消"))
	b.WriteString("\n")

	return b.String()
}

func runLoginSelector() error {
	m := newSelectorModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	result := finalModel.(selectorModel)
	if result.cancelled {
		fmt.Println("  已取消")
		return nil
	}

	switch result.selected {
	case methodQR:
		return runQRLogin()
	case methodCookie:
		return runCookieLogin()
	}

	return nil
}

// Cookie Login TUI

type cookieLoginModel struct {
	inputs    []textinput.Model
	focused   int
	saved     bool
	cancelled bool
	error     string
}

func newCookieLoginModel() cookieLoginModel {
	inputs := make([]textinput.Model, 3)

	// SESSDATA input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "粘贴 SESSDATA 值..."
	inputs[0].CharLimit = 500
	inputs[0].Width = 50
	inputs[0].Prompt = "  SESSDATA    > "
	inputs[0].PromptStyle = biliKeyStyle
	inputs[0].Focus()

	// bili_jct input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "粘贴 bili_jct 值..."
	inputs[1].CharLimit = 100
	inputs[1].Width = 50
	inputs[1].Prompt = "  bili_jct    > "
	inputs[1].PromptStyle = biliKeyStyle

	// DedeUserID input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "粘贴 DedeUserID 值..."
	inputs[2].CharLimit = 50
	inputs[2].Width = 50
	inputs[2].Prompt = "  DedeUserID  > "
	inputs[2].PromptStyle = biliKeyStyle

	// Load existing cookie if any
	cfg := config.LoadOrDefault()
	if cfg.Bilibili.Cookie != "" {
		for part := range strings.SplitSeq(cfg.Bilibili.Cookie, ";") {
			part = strings.TrimSpace(part)
			if val, ok := strings.CutPrefix(part, "SESSDATA="); ok {
				inputs[0].SetValue(val)
			} else if val, ok := strings.CutPrefix(part, "bili_jct="); ok {
				inputs[1].SetValue(val)
			} else if val, ok := strings.CutPrefix(part, "DedeUserID="); ok {
				inputs[2].SetValue(val)
			}
		}
	}

	return cookieLoginModel{
		inputs:  inputs,
		focused: 0,
	}
}

func (m cookieLoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m cookieLoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "tab", "down":
			m.inputs[m.focused].Blur()
			m.focused = (m.focused + 1) % len(m.inputs)
			m.inputs[m.focused].Focus()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.inputs[m.focused].Blur()
			m.focused--
			if m.focused < 0 {
				m.focused = len(m.inputs) - 1
			}
			m.inputs[m.focused].Focus()
			return m, textinput.Blink

		case "enter":
			if m.focused < len(m.inputs)-1 {
				m.inputs[m.focused].Blur()
				m.focused++
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}

			sessdata := strings.TrimSpace(m.inputs[0].Value())
			biliJct := strings.TrimSpace(m.inputs[1].Value())
			dedeUserID := strings.TrimSpace(m.inputs[2].Value())

			if sessdata == "" {
				m.error = "SESSDATA 不能为空"
				m.focused = 0
				m.inputs[0].Focus()
				return m, textinput.Blink
			}

			cookie := fmt.Sprintf("SESSDATA=%s; bili_jct=%s; DedeUserID=%s", sessdata, biliJct, dedeUserID)

			cfg := config.LoadOrDefault()
			cfg.Bilibili.Cookie = cookie
			if err := config.Save(cfg); err != nil {
				m.error = fmt.Sprintf("保存失败: %v", err)
				return m, nil
			}

			m.saved = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	cmds = append(cmds, cmd)
	m.error = ""

	return m, tea.Batch(cmds...)
}

func (m cookieLoginModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(biliTitleStyle.Render("  ━━━ Bilibili 登录 ━━━"))
	b.WriteString("\n\n")

	b.WriteString(biliTitleStyle.Render("  获取 Cookie 的方法："))
	b.WriteString("\n\n")
	b.WriteString(biliStepStyle.Render("  1. 在浏览器中打开 "))
	b.WriteString(biliKeyStyle.Render("bilibili.com"))
	b.WriteString(biliStepStyle.Render(" 并登录"))
	b.WriteString("\n")
	b.WriteString(biliStepStyle.Render("  2. 按 "))
	b.WriteString(biliKeyStyle.Render("F12"))
	b.WriteString(biliStepStyle.Render(" 打开开发者工具"))
	b.WriteString("\n")
	b.WriteString(biliStepStyle.Render("  3. 点击顶部「"))
	b.WriteString(biliKeyStyle.Render("Application"))
	b.WriteString(biliStepStyle.Render("」或「"))
	b.WriteString(biliKeyStyle.Render("应用"))
	b.WriteString(biliStepStyle.Render("」标签"))
	b.WriteString("\n")
	b.WriteString(biliStepStyle.Render("  4. 左侧展开 "))
	b.WriteString(biliKeyStyle.Render("Cookies"))
	b.WriteString(biliStepStyle.Render(" → 点击 "))
	b.WriteString(biliKeyStyle.Render("bilibili.com"))
	b.WriteString("\n")
	b.WriteString(biliStepStyle.Render("  5. 分别复制以下三个值:"))
	b.WriteString("\n\n")

	b.WriteString(biliHelpStyle.Render("  ─────────────────────────────────────────────────────────"))
	b.WriteString("\n\n")

	for i, input := range m.inputs {
		b.WriteString(input.View())
		if i < len(m.inputs)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	if m.error != "" {
		b.WriteString("\n")
		b.WriteString(biliErrorStyle.Render("  ✗ " + m.error))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(biliHelpStyle.Render("  Tab/↓ 下一项 • Shift+Tab/↑ 上一项 • Enter 保存 • Esc 取消"))
	b.WriteString("\n")

	return b.String()
}

func runCookieLogin() error {
	m := newCookieLoginModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	result := finalModel.(cookieLoginModel)
	if result.cancelled {
		fmt.Println("  已取消")
		return nil
	}

	if result.saved {
		fmt.Println(biliSuccessStyle.Render("  ✓ Bilibili Cookie 已保存"))
	}

	return nil
}

// QR Login TUI

type qrLoginState int

const (
	qrStateGenerating qrLoginState = iota
	qrStateWaiting
	qrStateScanned
	qrStateSuccess
	qrStateExpired
	qrStateError
)

type qrLoginModel struct {
	auth      *bilibili.Auth
	session   *bilibili.QRSession
	state     qrLoginState
	spinner   spinner.Model
	username  string
	error     string
	cancelled bool
}

type qrPollMsg struct {
	status bilibili.QRStatus
	creds  *bilibili.Credentials
	err    error
}

type qrGeneratedMsg struct {
	session *bilibili.QRSession
	err     error
}

func newQRLoginModel() qrLoginModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00A1D6"))

	return qrLoginModel{
		auth:    bilibili.NewAuth(),
		state:   qrStateGenerating,
		spinner: s,
	}
}

func (m qrLoginModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.generateQR,
	)
}

func (m qrLoginModel) generateQR() tea.Msg {
	session, err := m.auth.GenerateQRCode()
	return qrGeneratedMsg{session: session, err: err}
}

func (m qrLoginModel) pollStatus() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(time.Second)
		status, creds, err := m.auth.PollQRStatus(m.session.QRCodeKey)
		return qrPollMsg{status: status, creds: creds, err: err}
	}
}

func (m qrLoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.cancelled = true
			return m, tea.Quit
		case "r":
			if m.state == qrStateExpired || m.state == qrStateError {
				m.state = qrStateGenerating
				m.error = ""
				return m, m.generateQR
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case qrGeneratedMsg:
		if msg.err != nil {
			m.state = qrStateError
			m.error = msg.err.Error()
			return m, nil
		}
		m.session = msg.session
		m.state = qrStateWaiting
		printQRCode(m.session.URL)
		return m, m.pollStatus()

	case qrPollMsg:
		if msg.err != nil {
			m.state = qrStateError
			m.error = msg.err.Error()
			return m, nil
		}

		switch msg.status {
		case bilibili.QRWaiting:
			m.state = qrStateWaiting
			return m, m.pollStatus()

		case bilibili.QRScanned:
			m.state = qrStateScanned
			return m, m.pollStatus()

		case bilibili.QRExpired:
			m.state = qrStateExpired
			return m, nil

		case bilibili.QRConfirmed:
			m.state = qrStateSuccess
			if err := m.auth.SaveCredentials(msg.creds); err != nil {
				m.state = qrStateError
				m.error = err.Error()
				return m, nil
			}
			username, err := m.auth.ValidateCredentials(msg.creds)
			if err != nil {
				m.username = msg.creds.DedeUserID
			} else {
				m.username = username
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m qrLoginModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(biliTitleStyle.Render("  ━━━ Bilibili 扫码登录 ━━━"))
	b.WriteString("\n\n")

	switch m.state {
	case qrStateGenerating:
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" 正在生成二维码...\n")

	case qrStateWaiting:
		b.WriteString(biliStepStyle.Render("  请使用 Bilibili 客户端扫描上方二维码"))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" 等待扫码...\n")

	case qrStateScanned:
		b.WriteString(biliSuccessStyle.Render("  ✓ 扫码成功！"))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" 请在手机上确认登录...\n")

	case qrStateSuccess:
		b.WriteString(biliSuccessStyle.Render("  ✓ 登录成功！"))
		b.WriteString("\n")
		if m.username != "" {
			b.WriteString(biliStepStyle.Render(fmt.Sprintf("  欢迎，%s", m.username)))
			b.WriteString("\n")
		}

	case qrStateExpired:
		b.WriteString(biliErrorStyle.Render("  ✗ 二维码已过期"))
		b.WriteString("\n\n")
		b.WriteString(biliHelpStyle.Render("  按 r 重新生成二维码，按 q 退出"))
		b.WriteString("\n")

	case qrStateError:
		b.WriteString(biliErrorStyle.Render("  ✗ 错误: " + m.error))
		b.WriteString("\n\n")
		b.WriteString(biliHelpStyle.Render("  按 r 重试，按 q 退出"))
		b.WriteString("\n")
	}

	if m.state != qrStateSuccess && m.state != qrStateExpired && m.state != qrStateError {
		b.WriteString("\n")
		b.WriteString(biliHelpStyle.Render("  按 q 或 Esc 取消"))
		b.WriteString("\n")
	}

	return b.String()
}

func printQRCode(url string) {
	qr, err := qrcode.NewWith(url, qrcode.WithErrorCorrectionLevel(qrcode.ErrorCorrectionLow))
	if err != nil {
		fmt.Printf("  无法生成二维码: %v\n", err)
		return
	}

	w := vGetCompactQRWriter()
	if err := qr.Save(w); err != nil {
		fmt.Printf("  无法生成二维码: %v\n", err)
	}
	w.Close()

	fmt.Println()
	fmt.Println(biliHelpStyle.Render("  或在浏览器打开: " + url))
	fmt.Println()
}

func runQRLogin() error {
	m := newQRLoginModel()
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	result := finalModel.(qrLoginModel)
	if result.cancelled {
		fmt.Println("  已取消")
		return nil
	}

	return nil
}
