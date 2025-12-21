package gitpanel

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/git"
	"github.com/avitaltamir/vibecommander/internal/theme"
)

// Messages
type (
	// StageToggleMsg is sent when user toggles staging for a file.
	StageToggleMsg struct {
		Path     string
		IsStaged bool // Current state (will be toggled)
	}

	// OpenCommitMsg is sent when user wants to open the commit dialog.
	OpenCommitMsg struct{}

	// OpenFileMsg is sent when user wants to open a file in the viewer.
	OpenFileMsg struct {
		Path string
	}
)

// FileEntry represents a file in the git panel list.
type FileEntry struct {
	Path     string
	Status   git.FileStatus
	IsStaged bool
}

// KeyMap defines the key bindings for the git panel.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	Toggle   key.Binding
	Commit   key.Binding
	Open     key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("space"),
		),
		Commit: key.NewBinding(
			key.WithKeys("c"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter"),
		),
	}
}

// Model is the git panel component.
type Model struct {
	components.Base

	gitStatus *git.Status
	entries   []FileEntry // Sorted list of file entries
	cursor    int
	offset    int

	keys  KeyMap
	theme *theme.Theme
}

// New creates a new git panel model.
func New() Model {
	return Model{
		keys:  DefaultKeyMap(),
		theme: theme.DefaultTheme(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !m.Focused() {
			return m, nil
		}
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)

	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)

	case key.Matches(msg, m.keys.PageUp):
		_, h := m.Size()
		m.moveCursor(-h / 2)

	case key.Matches(msg, m.keys.PageDown):
		_, h := m.Size()
		m.moveCursor(h / 2)

	case key.Matches(msg, m.keys.Home):
		m.cursor = 0
		m.offset = 0

	case key.Matches(msg, m.keys.End):
		if len(m.entries) > 0 {
			m.cursor = len(m.entries) - 1
			m.ensureVisible()
		}

	case key.Matches(msg, m.keys.Toggle):
		return m.handleToggle()

	case key.Matches(msg, m.keys.Commit):
		// Only allow commit if there are staged files
		if m.hasStagedFiles() {
			return m, func() tea.Msg { return OpenCommitMsg{} }
		}

	case key.Matches(msg, m.keys.Open):
		// Open selected file in viewer
		if m.cursor >= 0 && m.cursor < len(m.entries) {
			entry := m.entries[m.cursor]
			return m, func() tea.Msg {
				return OpenFileMsg{Path: entry.Path}
			}
		}
	}

	return m, nil
}

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (Model, tea.Cmd) {
	mouse := msg.Mouse()
	switch mouse.Button {
	case tea.MouseWheelUp:
		m.moveCursor(-3)
	case tea.MouseWheelDown:
		m.moveCursor(3)
	}
	return m, nil
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (Model, tea.Cmd) {
	mouse := msg.Mouse()
	if mouse.Button == tea.MouseLeft {
		// Calculate which item was clicked (account for border)
		clickedIdx := m.offset + mouse.Y - 1
		if clickedIdx >= 0 && clickedIdx < len(m.entries) {
			m.cursor = clickedIdx
			m.MarkDirty()
			// Open the file (same as pressing Enter)
			entry := m.entries[m.cursor]
			return m, func() tea.Msg {
				return OpenFileMsg{Path: entry.Path}
			}
		}
	}
	return m, nil
}

func (m Model) handleToggle() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return m, nil
	}

	entry := m.entries[m.cursor]
	return m, func() tea.Msg {
		return StageToggleMsg{
			Path:     entry.Path,
			IsStaged: entry.IsStaged,
		}
	}
}

func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
	m.MarkDirty()
}

func (m *Model) ensureVisible() {
	_, h := m.Size()
	viewportHeight := h - 2 // Account for borders

	if viewportHeight <= 0 {
		return
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+viewportHeight {
		m.offset = m.cursor - viewportHeight + 1
	}
}

// View renders the git panel.
func (m Model) View() string {
	w, h := m.Size()
	if w <= 0 || h <= 0 {
		return ""
	}

	var lines []string

	viewportHeight := h - 2 // Account for borders
	if viewportHeight <= 0 {
		return ""
	}

	// Render entries
	for i := m.offset; i < len(m.entries) && len(lines) < viewportHeight; i++ {
		entry := m.entries[i]
		line := m.renderEntry(entry, i == m.cursor)
		lines = append(lines, line)
	}

	// Fill remaining space with empty lines
	for len(lines) < viewportHeight {
		lines = append(lines, strings.Repeat(" ", w-2))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderEntry(entry FileEntry, selected bool) string {
	w, _ := m.Size()
	contentWidth := w - 4 // Account for borders and padding

	// Status indicator
	var statusIndicator string

	if entry.IsStaged {
		statusIndicator = "●"
	} else {
		statusIndicator = "○"
	}

	// File status code
	var statusCode string
	if entry.IsStaged {
		statusCode = string(entry.Status.Staging)
	} else {
		statusCode = string(entry.Status.Worktree)
	}

	// Truncate path if needed
	path := entry.Path
	prefixLen := 5 // "● M "
	maxPathLen := contentWidth - prefixLen
	if len(path) > maxPathLen && maxPathLen > 3 {
		path = "..." + path[len(path)-maxPathLen+3:]
	}

	// Build the plain text line first
	plainLine := statusIndicator + " " + statusCode + " " + path

	// Pad to full width
	lineLen := prefixLen + len(path)
	if lineLen < contentWidth {
		plainLine += strings.Repeat(" ", contentWidth-lineLen)
	}

	// Apply styling
	if selected {
		// When selected, use file tree selection style (full line highlight)
		return theme.FileTreeSelected.Width(contentWidth).Render(plainLine)
	}

	// Not selected - apply color to indicator only
	var statusStyle lipgloss.Style
	if entry.IsStaged {
		statusStyle = theme.GitStatusAdded
	} else {
		statusStyle = theme.GitStatusModified
	}

	coloredIndicator := statusStyle.Render(statusIndicator)
	return coloredIndicator + " " + statusCode + " " + path + strings.Repeat(" ", max(0, contentWidth-lineLen))
}

// SetGitStatus updates the git status and rebuilds the file list.
func (m Model) SetGitStatus(status *git.Status) Model {
	m.gitStatus = status
	m.rebuildEntries()
	m.MarkDirty()
	return m
}

func (m *Model) rebuildEntries() {
	m.entries = nil

	if m.gitStatus == nil {
		return
	}

	// Collect all changed files
	for path, status := range m.gitStatus.Files {
		if !status.HasChanges() {
			continue
		}

		// Check if file is staged
		isStaged := status.IsStaged()

		m.entries = append(m.entries, FileEntry{
			Path:     path,
			Status:   status,
			IsStaged: isStaged,
		})
	}

	// Sort: staged files first, then by path
	sort.Slice(m.entries, func(i, j int) bool {
		if m.entries[i].IsStaged != m.entries[j].IsStaged {
			return m.entries[i].IsStaged // Staged files first
		}
		return m.entries[i].Path < m.entries[j].Path
	})

	// Clamp cursor
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) hasStagedFiles() bool {
	for _, entry := range m.entries {
		if entry.IsStaged {
			return true
		}
	}
	return false
}

// StagedCount returns the number of staged files.
func (m Model) StagedCount() int {
	count := 0
	for _, entry := range m.entries {
		if entry.IsStaged {
			count++
		}
	}
	return count
}

// UnstagedCount returns the number of unstaged files.
func (m Model) UnstagedCount() int {
	count := 0
	for _, entry := range m.entries {
		if !entry.IsStaged {
			count++
		}
	}
	return count
}

// Focus sets the focus state to true.
func (m Model) Focus() Model {
	m.Base.Focus()
	return m
}

// Blur sets the focus state to false.
func (m Model) Blur() Model {
	m.Base.Blur()
	return m
}

// SetSize updates the component's dimensions.
func (m Model) SetSize(width, height int) Model {
	m.Base.SetSize(width, height)
	m.ensureVisible()
	return m
}
