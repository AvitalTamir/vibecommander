package content

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/components/content/diff"
	"github.com/avitaltamir/vibecommander/internal/components/content/viewer"
	"github.com/avitaltamir/vibecommander/internal/components/terminal"
	"github.com/avitaltamir/vibecommander/internal/git"
	"github.com/avitaltamir/vibecommander/internal/theme"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Mode determines what the content pane displays.
type Mode int

const (
	ModeViewer Mode = iota
	ModeDiff
	ModeTerminal
	ModeAI
)

// ContentSource identifies a source of content in the panel.
// This prepares for future split-panel support while currently
// only one source is displayed at a time.
type ContentSource int

const (
	SourceNone ContentSource = iota
	SourceFile               // File viewer or diff
	SourceAI                 // AI terminal
)

// SourceInfo contains information about a content source for header display.
type SourceInfo struct {
	Source        ContentSource
	Title         string  // Display title (filename or AI command name)
	ScrollPercent float64 // -1 for terminals
	IsRunning     bool    // For AI: whether process is running
	IsActive      bool    // Whether this source is currently displayed
}

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

	// FileWithDiffMsg is sent after checking if a file has a diff.
	FileWithDiffMsg struct {
		Path    string
		Diff    string
		Content string
		HasDiff bool
		Err     error
	}

	// SwitchSourceMsg requests switching to a different content source.
	// Used when clicking on headers in the dual-header display.
	SwitchSourceMsg struct {
		Source ContentSource
	}
)

// Model is the content pane component that routes between different views.
type Model struct {
	components.Base

	mode     Mode
	lastMode Mode // Track previous mode for lazy size updates
	viewer   viewer.Model
	terminal terminal.Model
	diff     diff.Model

	currentPath string
	aiCommand   string // Stores the AI command name (e.g., "claude", "aider")
	gitProvider git.Provider
	theme       *theme.Theme

	// Track content sources for dual-header display
	// These allow showing both headers even when only one content is active
	hasFileContent bool // True if a file has been loaded
	hasAIContent   bool // True if AI terminal has been started
	// Cached dimensions for lazy sizing of inactive components
	lastWidth         int
	lastContentHeight int
}

// New creates a new content pane model.
func New() Model {
	return Model{
		mode:     ModeViewer,
		viewer:   viewer.New(),
		terminal: terminal.New(),
		diff:     diff.New(),
		theme:    theme.DefaultTheme(),
	}
}

// SetGitProvider sets the git provider for diff functionality.
func (m *Model) SetGitProvider(provider git.Provider) {
	m.gitProvider = provider
}

// ensureActiveComponentSized ensures the currently active component has correct dimensions
func (m *Model) ensureActiveComponentSized() {
	if m.lastWidth == 0 || m.lastContentHeight == 0 {
		return
	}
	switch m.mode {
	case ModeViewer:
		m.viewer = m.viewer.SetSize(m.lastWidth, m.lastContentHeight)
	case ModeDiff:
		m.diff = m.diff.SetSize(m.lastWidth, m.lastContentHeight)
	case ModeTerminal, ModeAI:
		m.terminal = m.terminal.SetSize(m.lastWidth, m.lastContentHeight)
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
		if m.mode != msg.Mode {
			m.lastMode = m.mode
			m.mode = msg.Mode
			// Ensure the newly active component has correct size
			m.ensureActiveComponentSized()
		}
		return m, nil

	case SwitchSourceMsg:
		// Switch to the requested content source
		switch msg.Source {
		case SourceFile:
			if m.hasFileContent {
				// Return to file view (viewer or diff mode, depending on last state)
				// For simplicity, default to viewer mode
				m.mode = ModeViewer
				m.viewer = m.viewer.Focus()
				m.terminal = m.terminal.Blur()
			}
		case SourceAI:
			if m.hasAIContent {
				m.mode = ModeAI
				m.terminal = m.terminal.Focus()
				m.viewer = m.viewer.Blur()
			}
		}
		return m, nil

	case OpenFileMsg:
		m.currentPath = msg.Path
		m.hasFileContent = true
		// Check if file has git changes - if so, show diff
		if m.gitProvider != nil {
			return m, m.loadFileWithDiffCheck(msg.Path)
		}
		if m.mode != ModeViewer {
			m.lastMode = m.mode
			m.mode = ModeViewer
			m.ensureActiveComponentSized()
		}
		return m, viewer.LoadFile(msg.Path)

	case LaunchAIMsg:
		if m.mode != ModeAI {
			m.lastMode = m.mode
			m.mode = ModeAI
			m.ensureActiveComponentSized()
		}
		m.aiCommand = msg.Command // Store the AI command name
		m.hasAIContent = true
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

	case diff.DiffLoadedMsg:
		// Route to diff viewer
		var cmd tea.Cmd
		m.diff, cmd = m.diff.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case FileWithDiffMsg:
		if msg.Err != nil {
			// Error loading - show in viewer
			if m.mode != ModeViewer {
				m.lastMode = m.mode
				m.mode = ModeViewer
				m.ensureActiveComponentSized()
			}
			var cmd tea.Cmd
			m.viewer, cmd = m.viewer.Update(viewer.FileLoadedMsg{
				Path: msg.Path,
				Err:  msg.Err,
			})
			return m, cmd
		}
		if msg.HasDiff && msg.Diff != "" {
			// Show diff view
			if m.mode != ModeDiff {
				m.lastMode = m.mode
				m.mode = ModeDiff
				m.ensureActiveComponentSized()
			}
			m.diff.SetContent(msg.Diff, msg.Path)
			return m, nil
		}
		// No diff - show normal viewer
		if m.mode != ModeViewer {
			m.lastMode = m.mode
			m.mode = ModeViewer
			m.ensureActiveComponentSized()
		}
		var cmd tea.Cmd
		m.viewer, cmd = m.viewer.Update(viewer.FileLoadedMsg{
			Path:    msg.Path,
			Content: msg.Content,
		})
		return m, cmd

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

	case tea.MouseWheelMsg:
		// Always pass mouse wheel events to active component for scrolling
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
		m.diff, cmd = m.diff.Update(msg)
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

	// Render the title - show filename in viewer/diff mode if file is loaded
	titleText := m.mode.String()
	if (m.mode == ModeViewer || m.mode == ModeDiff) && m.currentPath != "" {
		prefix := ""
		if m.mode == ModeDiff {
			prefix = "DIFF: "
		}
		titleText = prefix + filepath.Base(m.currentPath)
	}
	title := theme.RenderTitle(titleText, m.Focused())

	// Get content based on mode
	var content string
	contentHeight := h - 1 // Account for title

	switch m.mode {
	case ModeViewer:
		content = m.viewer.View()
	case ModeDiff:
		content = m.diff.View()
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
		Height(h-1).
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
	case ModeDiff:
		m.diff = m.diff.Focus()
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
	case ModeDiff:
		m.diff = m.diff.Blur()
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

	// On first call (lastWidth == 0), size all components so they're ready
	// After that, only update the active component's size immediately
	if m.lastWidth == 0 {
		m.viewer = m.viewer.SetSize(width, contentHeight)
		m.terminal = m.terminal.SetSize(width, contentHeight)
		m.diff = m.diff.SetSize(width, contentHeight)
	} else {
		switch m.mode {
		case ModeViewer:
			m.viewer = m.viewer.SetSize(width, contentHeight)
		case ModeDiff:
			m.diff = m.diff.SetSize(width, contentHeight)
		case ModeTerminal, ModeAI:
			m.terminal = m.terminal.SetSize(width, contentHeight)
		}
	}

	// Store dimensions for lazy sizing of inactive components
	m.lastWidth = width
	m.lastContentHeight = contentHeight

	return m
}

// ScrollPercent returns the scroll position of the current view.
func (m Model) ScrollPercent() float64 {
	switch m.mode {
	case ModeViewer:
		return m.viewer.ScrollPercent()
	case ModeDiff:
		return m.diff.ScrollPercent()
	default:
		return 0
	}
}

// loadFileWithDiffCheck loads a file and checks if it has git changes.
func (m Model) loadFileWithDiffCheck(path string) tea.Cmd {
	return func() tea.Msg {
		// Read the file content
		fileContent, err := readFile(path)
		if err != nil {
			return FileWithDiffMsg{Path: path, Err: err}
		}

		// Check for git diff
		if m.gitProvider != nil {
			diffContent, err := m.gitProvider.GetDiff(context.Background(), path)
			if err == nil && diffContent != "" {
				return FileWithDiffMsg{
					Path:    path,
					Diff:    diffContent,
					Content: fileContent,
					HasDiff: true,
				}
			}
		}

		// No diff - return content for normal viewing
		return FileWithDiffMsg{
			Path:    path,
			Content: fileContent,
			HasDiff: false,
		}
	}
}

// readFile reads a file and returns its content.
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// IsTerminalRunning returns true if the terminal is running a process.
func (m Model) IsTerminalRunning() bool {
	return (m.mode == ModeTerminal || m.mode == ModeAI) && m.terminal.Running()
}

// AICommandName returns the capitalized AI command name (e.g., "Claude", "Aider").
func (m Model) AICommandName() string {
	if m.aiCommand == "" {
		return "AI"
	}
	// Capitalize first letter
	return strings.ToUpper(m.aiCommand[:1]) + m.aiCommand[1:]
}

// ContentView returns just the inner content without any title or border.
// This is used by app.go to wrap content with the new border system.
func (m Model) ContentView() string {
	switch m.mode {
	case ModeViewer:
		return m.viewer.View()
	case ModeDiff:
		return m.diff.View()
	case ModeTerminal, ModeAI:
		return m.terminal.View()
	default:
		return ""
	}
}

// HasActiveSearch returns whether the viewer has an active search.
func (m Model) HasActiveSearch() bool {
	if m.mode == ModeViewer {
		return m.viewer.HasActiveSearch()
	}
	return false
}

// TitleInfo returns the title text and scroll percent for the current mode.
func (m Model) TitleInfo() (title string, scrollPercent float64) {
	switch m.mode {
	case ModeViewer:
		if m.currentPath != "" {
			title = filepath.Base(m.currentPath)
		} else {
			title = "VIEWER"
		}
		scrollPercent = m.viewer.ScrollPercent()
	case ModeDiff:
		if m.currentPath != "" {
			title = filepath.Base(m.currentPath)
		} else {
			title = "DIFF"
		}
		scrollPercent = m.diff.ScrollPercent()
	case ModeAI:
		title = m.AICommandName()
		scrollPercent = -1 // Don't show scroll for terminal
	case ModeTerminal:
		title = "TERMINAL"
		scrollPercent = -1
	default:
		title = "CONTENT"
		scrollPercent = -1
	}
	return
}

// SourcesInfo returns information about all available content sources.
// This enables the dual-header display showing both file and AI titles.
// Returns a slice of SourceInfo for sources that have content.
func (m Model) SourcesInfo() []SourceInfo {
	var sources []SourceInfo

	// Add file source if content exists
	if m.hasFileContent {
		fileInfo := SourceInfo{
			Source:   SourceFile,
			IsActive: m.mode == ModeViewer || m.mode == ModeDiff,
		}
		if m.currentPath != "" {
			fileInfo.Title = filepath.Base(m.currentPath)
		} else {
			fileInfo.Title = "VIEWER"
		}
		if m.mode == ModeViewer {
			fileInfo.ScrollPercent = m.viewer.ScrollPercent()
		} else if m.mode == ModeDiff {
			fileInfo.ScrollPercent = m.diff.ScrollPercent()
		} else {
			fileInfo.ScrollPercent = -1
		}
		sources = append(sources, fileInfo)
	}

	// Add AI source if content exists
	if m.hasAIContent {
		aiInfo := SourceInfo{
			Source:        SourceAI,
			Title:         m.AICommandName(),
			ScrollPercent: -1,
			IsRunning:     m.terminal.Running(),
			IsActive:      m.mode == ModeAI,
		}
		sources = append(sources, aiInfo)
	}

	return sources
}

// ActiveSource returns the currently active content source.
func (m Model) ActiveSource() ContentSource {
	switch m.mode {
	case ModeViewer, ModeDiff:
		return SourceFile
	case ModeAI, ModeTerminal:
		return SourceAI
	default:
		return SourceNone
	}
}

// HasMultipleSources returns true if both file and AI content are available.
func (m Model) HasMultipleSources() bool {
	return m.hasFileContent && m.hasAIContent
}
