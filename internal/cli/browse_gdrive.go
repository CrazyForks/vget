package cli

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guiyumin/vget/internal/core/gdrive"
)

type gdriveBrowseModel struct {
	client       *gdrive.Client
	currentPath  string
	entries      []gdrive.FileInfo
	cursor       int
	scrollOffset int
	width        int
	height       int
	err          error
	loading      bool
	done         bool
	selectedFile *gdrive.FileInfo // Selected file for download
	keyBindings  browseKeyMap
}

// Message types
type gdriveLoadedMsg struct {
	entries []gdrive.FileInfo
	err     error
}

func newGDriveBrowseModel(client *gdrive.Client, initialPath string) gdriveBrowseModel {
	return gdriveBrowseModel{
		client:      client,
		currentPath: initialPath,
		loading:     true,
		keyBindings: defaultBrowseKeyMap(),
	}
}

func (m gdriveBrowseModel) Init() tea.Cmd {
	return m.loadDirectory()
}

func (m gdriveBrowseModel) loadDirectory() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		entries, err := m.client.List(ctx, m.currentPath)
		if err != nil {
			return gdriveLoadedMsg{err: err}
		}

		// Sort: directories first, then alphabetically
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir != entries[j].IsDir {
				return entries[i].IsDir // directories first
			}
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})

		return gdriveLoadedMsg{entries: entries}
	}
}

func (m gdriveBrowseModel) visibleLines() int {
	if m.height <= 0 {
		return browseMaxVisibleLines
	}
	// Reserve: title (2) + path (2) + footer (3) + padding
	available := m.height - 10
	if available > browseMaxVisibleLines {
		return browseMaxVisibleLines
	}
	if available < 5 {
		return 5
	}
	return available
}

func (m *gdriveBrowseModel) adjustScroll() {
	visible := m.visibleLines()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
}

func (m gdriveBrowseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case gdriveLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.entries = msg.entries
		m.cursor = 0
		m.scrollOffset = 0
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			// Only allow quit while loading
			if key.Matches(msg, m.keyBindings.Quit) {
				return m, tea.Quit
			}
			return m, nil
		}

		if m.err != nil {
			// On error, allow quit or back
			if key.Matches(msg, m.keyBindings.Quit) {
				return m, tea.Quit
			}
			if key.Matches(msg, m.keyBindings.Back) {
				// Try to go back
				return m.goUp()
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, m.keyBindings.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keyBindings.Up):
			if m.cursor > 0 {
				m.cursor--
				m.adjustScroll()
			}

		case key.Matches(msg, m.keyBindings.Down):
			if m.cursor < len(m.entries)-1 {
				m.cursor++
				m.adjustScroll()
			}

		case key.Matches(msg, m.keyBindings.Enter):
			if len(m.entries) == 0 {
				return m, nil
			}
			entry := m.entries[m.cursor]
			if entry.IsDir {
				// Navigate into directory
				m.currentPath = path.Join(m.currentPath, entry.Name)
				m.loading = true
				m.entries = nil
				return m, m.loadDirectory()
			} else {
				// Select file for download
				m.selectedFile = &m.entries[m.cursor]
				m.done = true
				return m, tea.Quit
			}

		case key.Matches(msg, m.keyBindings.Back):
			return m.goUp()
		}
	}

	return m, nil
}

func (m gdriveBrowseModel) goUp() (tea.Model, tea.Cmd) {
	if m.currentPath == "/" {
		return m, nil // Already at root
	}
	m.currentPath = path.Dir(m.currentPath)
	if m.currentPath == "." {
		m.currentPath = "/"
	}
	m.loading = true
	m.entries = nil
	m.err = nil
	return m, m.loadDirectory()
}

func (m gdriveBrowseModel) View() string {
	var b strings.Builder

	// Title
	title := fmt.Sprintf("gdrive:%s", m.currentPath)
	b.WriteString(browseTitleStyle.Render("  Browse: ") + browsePathStyle.Render(title) + "\n\n")

	if m.loading {
		b.WriteString("  Loading...\n")
	} else if m.err != nil {
		b.WriteString(fmt.Sprintf("  Error: %v\n", m.err))
		b.WriteString("\n  Press b to go back, q to quit\n")
	} else if len(m.entries) == 0 {
		b.WriteString("  (empty directory)\n")
	} else {
		visible := m.visibleLines()
		endIdx := m.scrollOffset + visible
		if endIdx > len(m.entries) {
			endIdx = len(m.entries)
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			entry := m.entries[i]

			// Cursor indicator
			cursor := "  "
			if i == m.cursor {
				cursor = browseSelectedStyle.Render("> ")
			}

			// Icon and name
			var icon, name, size string
			if entry.IsDir {
				icon = browseDirStyle.Render("📁 ")
				name = entry.Name + "/"
				if i == m.cursor {
					name = browseSelectedStyle.Render(name)
				} else {
					name = browseDirStyle.Render(name)
				}
			} else {
				icon = browseFileStyle.Render("📄 ")
				name = entry.Name
				if i == m.cursor {
					name = browseSelectedStyle.Render(name)
				} else {
					name = browseFileStyle.Render(name)
				}
				size = browseSizeStyle.Render(fmt.Sprintf(" (%s)", formatSize(entry.Size)))
			}

			b.WriteString(fmt.Sprintf("%s%s%s%s\n", cursor, icon, name, size))
		}

		// Scroll indicator
		if len(m.entries) > visible {
			scrollInfo := fmt.Sprintf(" (%d-%d of %d)", m.scrollOffset+1, endIdx, len(m.entries))
			b.WriteString(browseSizeStyle.Render(scrollInfo) + "\n")
		}
	}

	b.WriteString("\n")

	// Help text
	help := "↑/↓ navigate • enter select • b back • q quit"
	b.WriteString(browseHelpStyle.Render("  " + help) + "\n")

	content := browseContainerStyle.Render(b.String())

	if m.width > 0 && m.height > 0 {
		content = lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)
	}

	return content
}

// GDriveBrowseResult holds the result of browsing
type GDriveBrowseResult struct {
	SelectedFile *gdrive.FileInfo // Selected file info
	Cancelled    bool             // User quit without selecting
}

// RunGDriveBrowseTUI runs the Google Drive file browser TUI
func RunGDriveBrowseTUI(client *gdrive.Client, initialPath string) (*GDriveBrowseResult, error) {
	model := newGDriveBrowseModel(client, initialPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	m := finalModel.(gdriveBrowseModel)
	if m.done && m.selectedFile != nil {
		return &GDriveBrowseResult{SelectedFile: m.selectedFile}, nil
	}

	return &GDriveBrowseResult{Cancelled: true}, nil
}
