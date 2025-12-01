package content

import (
	"path/filepath"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/components/content/viewer"
	"github.com/avitaltamir/vibecommander/internal/components/terminal"
	"github.com/avitaltamir/vibecommander/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mode determines what the content pane displays.
type Mode int

const (
	ModeViewer Mode = iota
	ModeDiff
	ModeTerminal
	ModeAI
)

func (m Mode) String() string {
	switch m {
	case ModeViewer:
		return "VIEWER"
	case ModeDiff:
		return "DIFF"
	case ModeTerminal:
		return "TERMINAL"
	case ModeAI:
		return "AI ASSISTANT"
	default:
		return "UNKNOWN"
	}
}

// Messages
type (
	// SetModeMsg changes the content pane mode.
	SetModeMsg struct {
		Mode Mode
	}

	// OpenFileMsg requests opening a file in the viewer.
	OpenFileMsg struct {
		Path string
	}

	// LaunchAIMsg requests launching the AI assistant.
	LaunchAIMsg struct {
		Command string   // e.g., "claude"
		Args    []string // e.g., []string{}
	}
)

// Model is the content pane component that routes between different views.
type Model struct {
	components.Base

	mode     Mode
	viewer   viewer.Model
	terminal terminal.Model
	// diff     diff.Model     // Will be added later

	currentPath string
	theme       *theme.Theme
}

// New creates a new content pane model.
func New() Model {
	return Model{
		mode:     ModeViewer,
		viewer:   viewer.New(),
		terminal: terminal.New(),
		theme:    theme.DefaultTheme(),
	}
}

// Init initializes the content pane.
func (m Model) Init() tea.Cmd {
	return m.viewer.Init()
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case SetModeMsg:
		m.mode = msg.Mode
		return m, nil

	case OpenFileMsg:
		m.mode = ModeViewer
		m.currentPath = msg.Path
		return m, viewer.LoadFile(msg.Path)

	case LaunchAIMsg:
		m.mode = ModeAI
		// Focus the terminal since we're switching to AI mode
		m.terminal = m.terminal.Focus()
		// Start the AI command
		var cmd tea.Cmd
		m.terminal, cmd = m.terminal.Update(terminal.StartMsg{
			Cmd:  msg.Command,
			Args: msg.Args,
		})
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case viewer.FileLoadedMsg:
		// Route to viewer
		var cmd tea.Cmd
		m.viewer, cmd = m.viewer.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case terminal.OutputMsg, terminal.ExitMsg:
		// Route to terminal and continue reading
		var cmd tea.Cmd
		m.terminal, cmd = m.terminal.Update(msg)
		cmds = append(cmds, cmd)
		// Continue reading output if still running
		if m.terminal.Running() {
			cmds = append(cmds, m.terminal.ContinueReading())
		}
		return m, tea.Batch(cmds...)

	case tea.MouseMsg:
		// Always pass mouse events to active component for scrolling
		return m.routeMessage(msg)
	}

	// Route other messages to active component
	return m.routeMessage(msg)
}

func (m Model) routeMessage(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.mode {
	case ModeViewer:
		m.viewer, cmd = m.viewer.Update(msg)
	case ModeDiff:
		// m.diff, cmd = m.diff.Update(msg)
	case ModeTerminal, ModeAI:
		m.terminal, cmd = m.terminal.Update(msg)
	}

	return m, cmd
}

// View renders the content pane.
func (m Model) View() string {
	w, h := m.Size()
	if w == 0 || h == 0 {
		return ""
	}

	// Render the title - show filename in viewer mode if file is loaded
	titleText := m.mode.String()
	if m.mode == ModeViewer && m.currentPath != "" {
		titleText = filepath.Base(m.currentPath)
	}
	title := theme.RenderTitle(titleText, m.Focused())

	// Get content based on mode
	var content string
	contentHeight := h - 1 // Account for title

	switch m.mode {
	case ModeViewer:
		content = m.viewer.View()
	case ModeDiff:
		content = m.renderPlaceholder("Diff view coming soon...")
	case ModeTerminal, ModeAI:
		content = m.terminal.View()
	}

	// Ensure content fits
	contentStyle := lipgloss.NewStyle().
		Width(w).
		Height(contentHeight)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		contentStyle.Render(content),
	)
}

func (m Model) renderPlaceholder(text string) string {
	w, h := m.Size()
	style := lipgloss.NewStyle().
		Width(w).
		Height(h - 1).
		Foreground(theme.MutedLavender).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render(text)
}

// Mode returns the current mode.
func (m Model) Mode() Mode {
	return m.mode
}

// SetMode sets the display mode.
func (m *Model) SetMode(mode Mode) {
	m.mode = mode
}

// CurrentPath returns the current file path (if any).
func (m Model) CurrentPath() string {
	return m.currentPath
}

// Focus gives focus to this component.
func (m Model) Focus() Model {
	m.Base.Focus()

	// Propagate focus to active sub-component
	switch m.mode {
	case ModeViewer:
		m.viewer = m.viewer.Focus()
	case ModeTerminal, ModeAI:
		m.terminal = m.terminal.Focus()
	}

	return m
}

// Blur removes focus from this component.
func (m Model) Blur() Model {
	m.Base.Blur()

	// Propagate blur to active sub-component
	switch m.mode {
	case ModeViewer:
		m.viewer = m.viewer.Blur()
	case ModeTerminal, ModeAI:
		m.terminal = m.terminal.Blur()
	}

	return m
}

// SetSize updates the component's dimensions.
func (m Model) SetSize(width, height int) Model {
	m.Base.SetSize(width, height)

	// Propagate size to all sub-components (minus space for title)
	contentHeight := height - 1
	if contentHeight < 0 {
		contentHeight = 0
	}

	m.viewer = m.viewer.SetSize(width, contentHeight)
	m.terminal = m.terminal.SetSize(width, contentHeight)
	// m.diff = m.diff.SetSize(width, contentHeight)

	return m
}

// ScrollPercent returns the scroll position of the current view.
func (m Model) ScrollPercent() float64 {
	switch m.mode {
	case ModeViewer:
		return m.viewer.ScrollPercent()
	default:
		return 0
	}
}

// IsTerminalRunning returns true if the terminal is running a process.
func (m Model) IsTerminalRunning() bool {
	return (m.mode == ModeTerminal || m.mode == ModeAI) && m.terminal.Running()
}
