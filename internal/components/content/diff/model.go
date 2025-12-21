package diff

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/theme"
)

// Messages
type (
	// DiffLoadedMsg is sent when a diff has been loaded.
	DiffLoadedMsg struct {
		Path string
		Diff string
		Err  error
	}
)

// Model is the diff viewer component.
type Model struct {
	components.Base

	viewport viewport.Model
	path     string
	diff     string
	ready    bool
	err      error

	theme *theme.Theme
}

// New creates a new diff viewer model.
func New() Model {
	return Model{
		theme: theme.DefaultTheme(),
	}
}

// Init initializes the diff viewer.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, nil

	case tea.MouseWheelMsg:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case DiffLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.diff = ""
			m.viewport.SetContent(m.renderError(msg.Err))
		} else {
			m.path = msg.Path
			m.diff = msg.Diff
			m.err = nil
			m.viewport.SetContent(m.renderDiff())
			m.viewport.GotoTop()
		}
		return m, nil

	case tea.KeyPressMsg:
		if !m.Focused() {
			return m, nil
		}

		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	if m.Focused() {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the diff viewer.
func (m Model) View() string {
	if !m.ready {
		return m.renderPlaceholder()
	}

	return m.viewport.View()
}

func (m Model) renderPlaceholder() string {
	w, h := m.Size()
	style := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Foreground(theme.MutedLavender).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render("No changes to display...")
}

func (m Model) renderError(err error) string {
	style := lipgloss.NewStyle().
		Foreground(theme.NeonRed).
		Bold(true)

	return style.Render("Error: " + err.Error())
}

func (m Model) renderDiff() string {
	if m.diff == "" {
		return lipgloss.NewStyle().
			Foreground(theme.MutedLavender).
			Italic(true).
			Render("(no changes)")
	}

	lines := strings.Split(m.diff, "\n")
	var result strings.Builder

	lineNumStyle := theme.DiffLineNumberStyle
	sepStyle := lipgloss.NewStyle().Foreground(theme.DimPurple)

	for i, line := range lines {
		lineNum := lineNumStyle.Render(padLeft(i+1, 4))
		sep := sepStyle.Render(" │ ")

		var styledLine string
		if len(line) > 0 {
			switch line[0] {
			case '+':
				if strings.HasPrefix(line, "+++") {
					// File header
					styledLine = theme.DiffHunkStyle.Render(line)
				} else {
					// Added line
					styledLine = theme.DiffAddedStyle.
						Background(theme.BgDiffAdded).
						Render(line)
				}
			case '-':
				if strings.HasPrefix(line, "---") {
					// File header
					styledLine = theme.DiffHunkStyle.Render(line)
				} else {
					// Removed line
					styledLine = theme.DiffRemovedStyle.
						Background(theme.BgDiffRemoved).
						Render(line)
				}
			case '@':
				// Hunk header
				styledLine = theme.DiffHunkStyle.
					Background(theme.BgDiffHunk).
					Render(line)
			case 'd':
				if strings.HasPrefix(line, "diff ") {
					// Diff header
					styledLine = lipgloss.NewStyle().
						Foreground(theme.CyberCyan).
						Bold(true).
						Render(line)
				} else {
					styledLine = theme.DiffContextStyle.Render(line)
				}
			case 'i', 'n', 's', 'o':
				// index, new file mode, similarity, old mode, etc.
				if strings.HasPrefix(line, "index ") ||
					strings.HasPrefix(line, "new file") ||
					strings.HasPrefix(line, "similarity") ||
					strings.HasPrefix(line, "old mode") ||
					strings.HasPrefix(line, "new mode") {
					styledLine = lipgloss.NewStyle().
						Foreground(theme.MutedLavender).
						Render(line)
				} else {
					styledLine = theme.DiffContextStyle.Render(line)
				}
			default:
				// Context line
				styledLine = theme.DiffContextStyle.Render(line)
			}
		} else {
			styledLine = ""
		}

		result.WriteString(lineNum)
		result.WriteString(sep)
		result.WriteString(styledLine)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// SetContent sets the diff content directly.
func (m *Model) SetContent(diff string, path string) {
	m.diff = diff
	m.path = path
	m.err = nil
	m.viewport.SetContent(m.renderDiff())
	m.viewport.GotoTop()
}

// Path returns the current file path.
func (m Model) Path() string {
	return m.path
}

// Diff returns the current diff content.
func (m Model) Diff() string {
	return m.diff
}

// HasContent returns true if there's diff content to display.
func (m Model) HasContent() bool {
	return m.diff != ""
}

// Clear clears the diff viewer.
func (m *Model) Clear() {
	m.path = ""
	m.diff = ""
	m.err = nil
	m.ready = false
	m.viewport.SetContent("")
}

// Focus gives focus to this component.
func (m Model) Focus() Model {
	m.Base.Focus()
	return m
}

// Blur removes focus from this component.
func (m Model) Blur() Model {
	m.Base.Blur()
	return m
}

// SetSize updates the component's dimensions.
func (m Model) SetSize(width, height int) Model {
	m.Base.SetSize(width, height)

	if !m.ready {
		m.viewport = viewport.New(
			viewport.WithWidth(width),
			viewport.WithHeight(height),
		)
		m.viewport.MouseWheelEnabled = true
		m.viewport.MouseWheelDelta = 3
		m.ready = true
	} else {
		m.viewport.SetWidth(width)
		m.viewport.SetHeight(height)
	}

	if m.diff != "" {
		m.viewport.SetContent(m.renderDiff())
	}

	return m
}

// ScrollPercent returns the current scroll position as a percentage (0-100).
func (m Model) ScrollPercent() float64 {
	return m.viewport.ScrollPercent() * 100
}

// Helper to pad line numbers
func padLeft(n, width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s += " "
	}
	ns := ""
	if n == 0 {
		ns = "0"
	} else {
		temp := n
		for temp > 0 {
			ns = string(rune('0'+temp%10)) + ns
			temp /= 10
		}
	}

	for len(ns) < width {
		ns = " " + ns
	}
	return ns
}
