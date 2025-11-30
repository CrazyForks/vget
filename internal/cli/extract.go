package cli

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guiyumin/vget/internal/extractor"
	"github.com/guiyumin/vget/internal/i18n"
)

var (
	extractInfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	extractDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	extractErrStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

// extractState holds extraction state
type extractState struct {
	mu     sync.RWMutex
	done   bool
	err    error
	result *extractor.VideoInfo
}

func (s *extractState) setDone(result *extractor.VideoInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
	s.result = result
}

func (s *extractState) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
	s.done = true
}

func (s *extractState) get() (bool, error, *extractor.VideoInfo) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.done, s.err, s.result
}

type extractTickMsg time.Time

type extractModel struct {
	spinner spinner.Model
	t       *i18n.Translations
	url     string
	state   *extractState
}

func newExtractModel(url, lang string, state *extractState) extractModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return extractModel{
		spinner: s,
		t:       i18n.T(lang),
		url:     url,
		state:   state,
	}
}

func extractTickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return extractTickMsg(t)
	})
}

func (m extractModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, extractTickCmd())
}

func (m extractModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case extractTickMsg:
		done, _, _ := m.state.get()
		if done {
			return m, tea.Quit
		}
		return m, extractTickCmd()
	}

	return m, nil
}

func (m extractModel) View() string {
	done, err, result := m.state.get()

	if err != nil {
		return fmt.Sprintf("\n  %s %s: %v\n\n",
			extractErrStyle.Render("✗"),
			m.t.Errors.ExtractionFailed,
			err,
		)
	}

	if done && result != nil {
		return fmt.Sprintf("\n  %s %s\n  ID: %s  |  Formats: %d\n\n",
			extractDoneStyle.Render("✓"),
			m.t.Download.Completed,
			extractInfoStyle.Render(result.ID),
			len(result.Formats),
		)
	}

	return fmt.Sprintf("\n  %s %s: %s\n\n",
		m.spinner.View(),
		m.t.Download.Extracting,
		extractInfoStyle.Render(m.url),
	)
}

// runExtractWithSpinner runs extraction with a spinner TUI
func runExtractWithSpinner(ext extractor.Extractor, url, lang string) (*extractor.VideoInfo, error) {
	state := &extractState{}

	// Start extraction in background
	go func() {
		result, err := ext.Extract(url)
		if err != nil {
			state.setError(err)
		} else {
			state.setDone(result)
		}
	}()

	model := newExtractModel(url, lang, state)
	p := tea.NewProgram(model)
	_, err := p.Run()
	if err != nil {
		return nil, err
	}

	done, extractErr, result := state.get()
	if extractErr != nil {
		return nil, extractErr
	}
	if !done {
		return nil, fmt.Errorf("extraction cancelled")
	}

	return result, nil
}
