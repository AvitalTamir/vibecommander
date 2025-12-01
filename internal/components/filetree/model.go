package filetree

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/git"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type (
	// LoadedMsg is sent when children have been loaded.
	LoadedMsg struct {
		Path     string
		Children []*Node
		Err      error
	}

	// SelectMsg is sent when a file is selected (to open it).
	SelectMsg struct {
		Path  string
		IsDir bool
	}
)

// KeyMap defines the key bindings for the file tree.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Enter    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
	Toggle   key.Binding
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
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
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
	}
}

// Model is the file tree component.
type Model struct {
	components.Base

	root       *Node
	visible    []*Node // Flattened visible nodes
	cursor     int     // Current cursor position
	offset     int     // Scroll offset for viewport
	showHidden bool
	loading    map[string]bool // Paths currently being loaded

	gitStatus *git.Status // Current git status
	workDir   string      // Working directory for relative paths

	keys  KeyMap
	theme *theme.Theme
}

// New creates a new file tree model rooted at the current directory.
func New() Model {
	cwd, _ := os.Getwd()

	m := Model{
		loading: make(map[string]bool),
		keys:    DefaultKeyMap(),
		theme:   theme.DefaultTheme(),
		workDir: cwd,
	}

	// Initialize root to current working directory
	if cwd != "" {
		if root, err := NewRootNode(cwd); err == nil {
			m.root = root
		}
	}

	return m
}

// NewWithPath creates a new file tree rooted at the given path.
func NewWithPath(path string) (Model, error) {
	m := New()

	root, err := NewRootNode(path)
	if err != nil {
		return m, err
	}

	m.root = root
	return m, nil
}

// Init initializes the file tree.
func (m Model) Init() tea.Cmd {
	if m.root == nil {
		return nil
	}

	// Load root children
	return m.loadChildren(m.root.Path)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	// LoadedMsg should always be handled regardless of focus
	case LoadedMsg:
		return m.handleLoaded(msg)

	case tea.KeyMsg:
		if !m.Focused() {
			return m, nil
		}
		return m.handleKey(msg)

	case tea.MouseMsg:
		if !m.Focused() {
			return m, nil
		}
		return m.handleMouse(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
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
		if len(m.visible) > 0 {
			m.cursor = len(m.visible) - 1
			m.ensureVisible()
		}

	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Right):
		return m.handleSelect()

	case key.Matches(msg, m.keys.Left):
		return m.handleBack()

	case key.Matches(msg, m.keys.Toggle):
		return m.handleToggle()
	}

	return m, nil
}

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.moveCursor(-3)
	case tea.MouseButtonWheelDown:
		m.moveCursor(3)
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress {
			// Calculate which item was clicked
			clickedIdx := m.offset + msg.Y - 2 // Account for border and title
			if clickedIdx >= 0 && clickedIdx < len(m.visible) {
				m.cursor = clickedIdx
				return m.handleSelect()
			}
		}
	}
	return m, nil
}

func (m Model) handleLoaded(msg LoadedMsg) (Model, tea.Cmd) {
	delete(m.loading, msg.Path)

	if msg.Err != nil {
		// Could send error message to status bar
		return m, nil
	}

	if m.root == nil {
		return m, nil
	}

	node := m.root.FindByPath(msg.Path)
	if node == nil {
		return m, nil
	}

	node.Children = msg.Children
	node.Loaded = true

	// Update parent references and depths
	for _, child := range node.Children {
		child.Parent = node
		child.Depth = node.Depth + 1
	}

	m.rebuildVisible()
	return m, nil
}

func (m Model) handleSelect() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return m, nil
	}

	node := m.visible[m.cursor]

	if node.IsDir {
		// Expand/collapse directory
		if node.Expanded {
			node.Collapse()
			m.rebuildVisible()
		} else {
			if !node.Loaded && !m.loading[node.Path] {
				m.loading[node.Path] = true
				node.Expanded = true
				m.rebuildVisible()
				return m, m.loadChildren(node.Path)
			}
			node.Expanded = true
			m.rebuildVisible()
		}
		return m, nil
	}

	// File selected - send message to open it
	return m, func() tea.Msg {
		return SelectMsg{Path: node.Path, IsDir: false}
	}
}

func (m Model) handleBack() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return m, nil
	}

	node := m.visible[m.cursor]

	// If on expanded directory, collapse it
	if node.IsDir && node.Expanded {
		node.Collapse()
		m.rebuildVisible()
		return m, nil
	}

	// Otherwise go to parent
	if node.Parent != nil && node.Parent != m.root {
		// Find parent in visible list
		for i, n := range m.visible {
			if n == node.Parent {
				m.cursor = i
				m.ensureVisible()
				break
			}
		}
	}

	return m, nil
}

func (m Model) handleToggle() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return m, nil
	}

	node := m.visible[m.cursor]
	if node.IsDir {
		node.Toggle()
		if node.Expanded && !node.Loaded && !m.loading[node.Path] {
			m.loading[node.Path] = true
			m.rebuildVisible()
			return m, m.loadChildren(node.Path)
		}
		m.rebuildVisible()
	}

	return m, nil
}

func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
}

func (m *Model) ensureVisible() {
	_, h := m.Size()
	viewportHeight := h - 3 // Account for borders and title

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

func (m *Model) rebuildVisible() {
	if m.root == nil {
		m.visible = nil
		return
	}
	m.visible = m.root.Flatten(m.showHidden)

	// Keep cursor in bounds
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m Model) loadChildren(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path)
		if err != nil {
			return LoadedMsg{Path: path, Err: err}
		}

		children := make([]*Node, 0, len(entries))
		for _, entry := range entries {
			childPath := filepath.Join(path, entry.Name())
			info, err := entry.Info()
			if err != nil {
				continue
			}

			child := &Node{
				Path:    childPath,
				Name:    entry.Name(),
				IsDir:   entry.IsDir(),
				Depth:   0, // Will be set in handleLoaded
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
			}
			children = append(children, child)
		}

		// Sort: directories first, then alphabetically
		sortNodes(children)

		return LoadedMsg{Path: path, Children: children}
	}
}

// View renders the file tree.
func (m Model) View() string {
	w, h := m.Size()
	if w == 0 || h == 0 {
		return ""
	}

	contentHeight := h - 2 // Account for borders

	var lines []string
	for i := m.offset; i < len(m.visible) && len(lines) < contentHeight; i++ {
		node := m.visible[i]
		line := m.renderNode(node, i == m.cursor, w-4) // Account for border + padding
		lines = append(lines, line)
	}

	// Pad with empty lines if needed
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	return content
}

func (m Model) renderNode(node *Node, selected bool, maxWidth int) string {
	// Build indentation
	indent := ""
	for i := 0; i < node.Depth; i++ {
		indent += theme.TreeSpace
	}

	// Get icon
	var icon string
	if node.IsDir {
		icon = m.theme.GetDirIcon(node.Name, node.Expanded)
	} else {
		icon = m.theme.GetFileIcon(node.Extension())
	}

	// Build the line
	name := node.Name
	if node.IsDir {
		name += "/"
	}

	// Get git status indicator
	gitIndicator := m.getGitIndicator(node)

	line := indent + icon + " " + name

	// Truncate if too long (leave room for git indicator)
	indicatorWidth := lipgloss.Width(gitIndicator)
	availableWidth := maxWidth - indicatorWidth - 1
	if lipgloss.Width(line) > availableWidth {
		line = line[:availableWidth-1] + "…"
	}

	// Apply styling
	var style lipgloss.Style
	if selected {
		style = theme.FileTreeSelected.Width(maxWidth - indicatorWidth - 1)
	} else if node.IsDir {
		style = theme.FileTreeDir
	} else {
		style = theme.FileTreeFile
	}

	// Add loading indicator
	if m.loading[node.Path] {
		line += " " + lipgloss.NewStyle().Foreground(theme.MagentaBlaze).Render("⠋")
	}

	result := style.Render(line)

	// Add git indicator at the end
	if gitIndicator != "" {
		result += " " + gitIndicator
	}

	return result
}

// getGitIndicator returns a styled git status indicator for the node
func (m Model) getGitIndicator(node *Node) string {
	if m.gitStatus == nil || m.workDir == "" {
		return ""
	}

	// Get relative path from workdir
	relPath, err := filepath.Rel(m.workDir, node.Path)
	if err != nil {
		return ""
	}

	// Check for exact match
	if status, ok := m.gitStatus.Files[relPath]; ok {
		return m.renderGitStatus(status)
	}

	// For directories, check status of all children
	if node.IsDir {
		dirPrefix := relPath + string(filepath.Separator)
		hasChanges := false
		allStaged := true
		hasUntracked := false

		for path, status := range m.gitStatus.Files {
			if strings.HasPrefix(path, dirPrefix) {
				hasChanges = true

				// Check if untracked
				if status.Staging == git.StatusUntracked || status.Worktree == git.StatusUntracked {
					hasUntracked = true
					allStaged = false
				} else if status.Worktree != git.StatusUnmodified && status.Worktree != ' ' {
					// Has unstaged changes
					allStaged = false
				}
			}
		}

		if hasChanges {
			if hasUntracked {
				return lipgloss.NewStyle().Foreground(theme.LaserPurple).Render("●")
			} else if allStaged {
				return lipgloss.NewStyle().Foreground(theme.MatrixGreen).Render("●")
			} else {
				return lipgloss.NewStyle().Foreground(theme.ElectricYellow).Render("●")
			}
		}
	}

	return ""
}

// renderGitStatus returns a styled indicator for a file's git status
// Shows both staging and worktree status: [staged][worktree]
func (m Model) renderGitStatus(status git.FileStatus) string {
	var result string

	// Staged status (index) - shown in green
	staged := status.Staging
	if staged != git.StatusUnmodified && staged != ' ' && staged != git.StatusUntracked {
		stagedStyle := lipgloss.NewStyle().Foreground(theme.MatrixGreen)
		switch staged {
		case git.StatusModified:
			result += stagedStyle.Render("M")
		case git.StatusAdded:
			result += stagedStyle.Render("A")
		case git.StatusDeleted:
			result += stagedStyle.Render("D")
		case git.StatusRenamed:
			result += stagedStyle.Render("R")
		case git.StatusCopied:
			result += stagedStyle.Render("C")
		}
	}

	// Worktree status (unstaged) - shown in yellow/red
	worktree := status.Worktree
	if worktree != git.StatusUnmodified && worktree != ' ' {
		switch worktree {
		case git.StatusModified:
			result += lipgloss.NewStyle().Foreground(theme.ElectricYellow).Render("M")
		case git.StatusDeleted:
			result += lipgloss.NewStyle().Foreground(theme.NeonRed).Render("D")
		case git.StatusUntracked:
			result += lipgloss.NewStyle().Foreground(theme.LaserPurple).Render("?")
		case git.StatusUnmerged:
			result += lipgloss.NewStyle().Foreground(theme.NeonRed).Bold(true).Render("!")
		}
	}

	return result
}

// SetRoot sets the root directory.
func (m *Model) SetRoot(path string) error {
	root, err := NewRootNode(path)
	if err != nil {
		return err
	}
	m.root = root
	m.cursor = 0
	m.offset = 0
	m.visible = nil
	return nil
}

// Root returns the root path.
func (m Model) Root() string {
	if m.root == nil {
		return ""
	}
	return m.root.Path
}

// SelectedPath returns the currently selected path.
func (m Model) SelectedPath() string {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return ""
	}
	return m.visible[m.cursor].Path
}

// SelectedNode returns the currently selected node.
func (m Model) SelectedNode() *Node {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return m.visible[m.cursor]
}

// SetShowHidden sets whether to show hidden files.
func (m *Model) SetShowHidden(show bool) {
	m.showHidden = show
	m.rebuildVisible()
}

// RefreshDir triggers a reload of the specified directory.
// If the path is a file, it refreshes the parent directory.
func (m Model) RefreshDir(path string) tea.Cmd {
	if m.root == nil {
		return nil
	}

	// Normalize the path
	path = filepath.Clean(path)

	// Get the directory to refresh (parent if path is a file)
	dirPath := path
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		// Path doesn't exist or is a file - use parent directory
		dirPath = filepath.Dir(path)
	}

	// Check if it's the root directory itself
	if dirPath == m.root.Path {
		if m.root.Loaded {
			return m.loadChildren(m.root.Path)
		}
		return nil
	}

	// Find the directory node in our tree
	node := m.root.FindByPath(dirPath)

	// If not found but it's within our root, try to refresh root
	if node == nil {
		// Check if the path is under our root
		if rel, err := filepath.Rel(m.root.Path, dirPath); err == nil && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..") {
			// Path is under root, refresh root
			if m.root.Loaded {
				return m.loadChildren(m.root.Path)
			}
		}
		return nil
	}

	// If node is a file, get its parent
	if !node.IsDir {
		if node.Parent != nil {
			node = node.Parent
		} else {
			return nil
		}
	}

	// Reload if the directory has been loaded before
	if node.Loaded {
		return m.loadChildren(node.Path)
	}

	return nil
}

// ShowHidden returns whether hidden files are shown.
func (m Model) ShowHidden() bool {
	return m.showHidden
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
	m.ensureVisible()
	return m
}

// SetGitStatus updates the git status for the file tree.
func (m Model) SetGitStatus(status *git.Status) Model {
	m.gitStatus = status
	return m
}

// Helper to sort nodes
func sortNodes(nodes []*Node) {
	// Simple bubble sort for small lists
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			// Directories come first
			if nodes[j].IsDir && !nodes[i].IsDir {
				nodes[i], nodes[j] = nodes[j], nodes[i]
				continue
			}
			if nodes[i].IsDir && !nodes[j].IsDir {
				continue
			}
			// Then alphabetically (case-insensitive)
			if nodes[j].Name < nodes[i].Name {
				nodes[i], nodes[j] = nodes[j], nodes[i]
			}
		}
	}
}
