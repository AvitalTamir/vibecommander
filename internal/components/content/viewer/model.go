package viewer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type (
	// FileLoadedMsg is sent when a file has been loaded.
	FileLoadedMsg struct {
		Path    string
		Content string
		Err     error
	}
)

// Model is the content viewer component.
type Model struct {
	components.Base

	viewport viewport.Model
	path     string
	content  string
	ready    bool
	err      error

	theme *theme.Theme
}

// New creates a new content viewer model.
func New() Model {
	return Model{
		theme: theme.DefaultTheme(),
	}
}

// Init initializes the viewer.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// This is handled by SetSize from parent
		return m, nil

	case tea.MouseMsg:
		// Always handle mouse for scrolling, even when not focused
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case FileLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.content = ""
			m.viewport.SetContent(m.renderError(msg.Err))
		} else {
			m.path = msg.Path
			m.content = msg.Content
			m.err = nil
			m.viewport.SetContent(m.renderContent())
			m.viewport.GotoTop()
		}
		return m, nil
	}

	// Handle keyboard only when focused
	if m.Focused() {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the viewer.
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

	return style.Render("Select a file to view its contents...")
}

func (m Model) renderError(err error) string {
	style := lipgloss.NewStyle().
		Foreground(theme.NeonRed).
		Bold(true)

	return style.Render("Error: " + err.Error())
}

func (m Model) renderContent() string {
	if m.content == "" {
		return lipgloss.NewStyle().
			Foreground(theme.MutedLavender).
			Italic(true).
			Render("(empty file)")
	}

	// Get syntax highlighted content
	highlighted := m.highlightSyntax()

	// Add line numbers
	lines := strings.Split(highlighted, "\n")
	var result strings.Builder

	lineNumStyle := theme.DiffLineNumberStyle
	sepStyle := lipgloss.NewStyle().Foreground(theme.DimPurple)

	for i, line := range lines {
		lineNum := lineNumStyle.Render(padLeft(i+1, 4))
		result.WriteString(lineNum)
		result.WriteString(sepStyle.Render(" │ "))
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// highlightSyntax returns syntax-highlighted content
func (m Model) highlightSyntax() string {
	// Try to get lexer by filename
	var lexer chroma.Lexer
	if m.path != "" {
		lexer = lexers.Match(filepath.Base(m.path))
	}

	// Fallback: try to analyze content
	if lexer == nil {
		lexer = lexers.Analyse(m.content)
	}

	// Final fallback: plain text
	if lexer == nil {
		lexer = lexers.Fallback
	}

	lexer = chroma.Coalesce(lexer)

	// Use a dark theme that works well with terminal
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	// Use terminal256 formatter for colored output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize and format
	iterator, err := lexer.Tokenise(nil, m.content)
	if err != nil {
		return m.content
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return m.content
	}

	return buf.String()
}

// LoadFile loads a file into the viewer.
func LoadFile(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		if err != nil {
			return FileLoadedMsg{Path: path, Err: err}
		}
		return FileLoadedMsg{Path: path, Content: string(content)}
	}
}

// SetContent sets the content directly (for non-file content).
func (m *Model) SetContent(content string) {
	m.content = content
	m.path = ""
	m.err = nil
	m.viewport.SetContent(m.renderContent())
	m.viewport.GotoTop()
}

// Path returns the current file path.
func (m Model) Path() string {
	return m.path
}

// Content returns the current content.
func (m Model) Content() string {
	return m.content
}

// Clear clears the viewer.
func (m *Model) Clear() {
	m.path = ""
	m.content = ""
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

	// Initialize or resize viewport
	if !m.ready {
		m.viewport = viewport.New(width, height)
		m.viewport.MouseWheelEnabled = true
		m.viewport.MouseWheelDelta = 3
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = height
	}

	// Re-render content to fit new width
	if m.content != "" {
		m.viewport.SetContent(m.renderContent())
	}

	return m
}

// ScrollPercent returns the current scroll position as a percentage.
func (m Model) ScrollPercent() float64 {
	return m.viewport.ScrollPercent()
}

// Helper to pad line numbers
func padLeft(n, width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s += " "
	}
	ns := strings.TrimLeft(s+string(rune('0'+n%10)), " ")
	if n >= 10 {
		ns = string(rune('0'+n/10%10)) + string(rune('0'+n%10))
	}
	if n >= 100 {
		ns = string(rune('0'+n/100%10)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
	}
	if n >= 1000 {
		ns = string(rune('0'+n/1000%10)) + string(rune('0'+n/100%10)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
	}

	// Pad to width
	for len(ns) < width {
		ns = " " + ns
	}
	return ns
}
