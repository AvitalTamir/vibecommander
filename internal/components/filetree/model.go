package filetree

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/git"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	// StageToggleMsg is sent when user wants to toggle staging for a file.
	StageToggleMsg struct {
		Path     string
		IsStaged bool // Current state (will be toggled)
	}
)

// KeyMap defines the key bindings for the file tree.
type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	Enter         key.Binding
	PageUp        key.Binding
	PageDown      key.Binding
	Home          key.Binding
	End           key.Binding
	Toggle        key.Binding
	CompactIndent key.Binding
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
		CompactIndent: key.NewBinding(
			key.WithKeys("alt+i", "ˆ"), // ˆ = Option+i on Mac
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

	gitStatus        *git.Status // Current git status
	gitStatusVersion uint64      // Version counter for cache invalidation
	workDir          string      // Working directory for relative paths

	// Cached directory git status (computed when git status updates)
	dirGitStatusCache map[string]string // path -> cached indicator

	// Cached view for dirty checking
	cachedView string

	// Search/filter functionality
	searching   bool            // Whether search mode is active
	searchInput textinput.Model // Text input for search query
	searchQuery string          // Current search filter (persists after exiting search mode)
	matchCount  int             // Number of matching files

	// Display options
	compactIndent bool // Use 2-space indentation instead of 4-space

	keys  KeyMap
	theme *theme.Theme
}

// New creates a new file tree model rooted at the current directory.
func New() Model {
	cwd, _ := os.Getwd()

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Search files..."
	ti.CharLimit = 100

	m := Model{
		loading:           make(map[string]bool),
		dirGitStatusCache: make(map[string]string),
		keys:              DefaultKeyMap(),
		theme:             theme.DefaultTheme(),
		workDir:           cwd,
		showHidden:        true,
		searchInput:       ti,
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

	case tea.KeyPressMsg:
		if !m.Focused() {
			return m, nil
		}
		// When searching, route most keys to the search input
		if m.searching {
			return m.handleSearchKey(msg)
		}
		return m.handleKey(msg)

	case tea.MouseWheelMsg:
		// Always handle mouse wheel - app.go handles focus management
		return m.handleMouseWheel(msg)

	case tea.MouseClickMsg:
		// Always handle mouse clicks - app.go handles focus management
		// This allows click-to-select even when the panel wasn't focused
		return m.handleMouseClick(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	// Check for search activation
	if msg.String() == "/" {
		m.searching = true
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		return m, textinput.Blink
	}

	// Check for Escape to clear search filter (when not in search mode)
	if msg.String() == "esc" && m.searchQuery != "" {
		m.searchQuery = ""
		m.rebuildVisible()
		return m, nil
	}

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

	case key.Matches(msg, m.keys.CompactIndent):
		m.compactIndent = !m.compactIndent
		m.MarkDirty()
		return m, nil
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Confirm search and exit search mode
		m.searching = false
		m.searchQuery = m.searchInput.Value()
		m.searchInput.Blur()
		m.rebuildVisible()
		// Jump to first match if there is one
		if len(m.visible) > 0 {
			m.cursor = 0
			m.offset = 0
		}
		return m, nil

	case "esc":
		// Cancel search, clear filter, exit search mode
		m.searching = false
		m.searchQuery = ""
		m.searchInput.Blur()
		m.rebuildVisible()
		return m, nil
	}

	// Update the search input
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)

	// Apply filter as user types
	m.searchQuery = m.searchInput.Value()
	m.rebuildVisible()

	return m, cmd
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
		// Calculate which item was clicked
		clickedIdx := m.offset + mouse.Y - 2 // Account for border and title
		if clickedIdx >= 0 && clickedIdx < len(m.visible) {
			m.cursor = clickedIdx
			return m.handleSelect()
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
		// Directory: expand/collapse
		node.Toggle()
		if node.Expanded && !node.Loaded && !m.loading[node.Path] {
			m.loading[node.Path] = true
			m.rebuildVisible()
			return m, m.loadChildren(node.Path)
		}
		m.rebuildVisible()
		return m, nil
	}

	// File: check for git changes and emit staging toggle
	if m.gitStatus != nil && m.workDir != "" {
		relPath, err := filepath.Rel(m.workDir, node.Path)
		if err == nil {
			if status, ok := m.gitStatus.Files[relPath]; ok && status.HasChanges() {
				isStaged := status.IsStaged()
				return m, func() tea.Msg {
					return StageToggleMsg{
						Path:     node.Path,
						IsStaged: isStaged,
					}
				}
			}
		}
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

	allNodes := m.root.Flatten(m.showHidden)

	// Apply search filter if active
	if m.searchQuery != "" {
		m.visible, m.matchCount = m.filterNodes(allNodes, m.searchQuery)
	} else {
		m.visible = allNodes
		m.matchCount = 0
	}

	// Keep cursor in bounds
	if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Mark dirty since visible nodes changed
	m.MarkDirty()
}

// filterNodes filters nodes based on search query.
// Returns matching nodes (files that match + their parent directories) and match count.
func (m *Model) filterNodes(nodes []*Node, query string) ([]*Node, int) {
	if query == "" {
		return nodes, 0
	}

	query = strings.ToLower(query)
	matchingPaths := make(map[string]bool)
	matchCount := 0

	// First pass: find all matching files and their ancestor paths
	for _, node := range nodes {
		if !node.IsDir && strings.Contains(strings.ToLower(node.Name), query) {
			matchCount++
			// Mark this node and all its ancestors as matching
			matchingPaths[node.Path] = true
			parent := node.Parent
			for parent != nil {
				matchingPaths[parent.Path] = true
				// Auto-expand parent directories to show matches
				parent.Expanded = true
				parent = parent.Parent
			}
		}
	}

	// If no matches, show all nodes (don't filter)
	if matchCount == 0 {
		return nodes, 0
	}

	// Second pass: collect all matching nodes
	var result []*Node
	for _, node := range nodes {
		if matchingPaths[node.Path] {
			result = append(result, node)
		}
	}

	return result, matchCount
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

	// Reserve space for search bar if searching or filtered
	searchBarHeight := 0
	if m.searching || m.searchQuery != "" {
		searchBarHeight = 1
	}

	contentHeight := h - 2 - searchBarHeight // Account for borders and search bar

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

	// Add search bar at the bottom if searching or filtered
	if m.searching || m.searchQuery != "" {
		searchBar := m.renderSearchBar(w - 4)
		content = lipgloss.JoinVertical(lipgloss.Left, content, searchBar)
	}

	return content
}

// renderSearchBar renders the search input bar.
func (m Model) renderSearchBar(maxWidth int) string {
	var bar string

	if m.searching {
		// Active search mode - show input
		bar = "/" + m.searchInput.View()
	} else if m.searchQuery != "" {
		// Filter active - show filter info
		filterStyle := lipgloss.NewStyle().Foreground(theme.CyberCyan)
		bar = filterStyle.Render("/ " + m.searchQuery)
		if m.matchCount > 0 {
			countStyle := lipgloss.NewStyle().Foreground(theme.MutedLavender)
			bar += countStyle.Render(" (" + itoa(m.matchCount) + " matches)")
		}
	}

	return bar
}

// itoa converts an int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var result []byte
	for n > 0 {
		result = append([]byte{byte('0' + n%10)}, result...)
		n /= 10
	}
	return string(result)
}

// ViewCached returns the cached view if not dirty, otherwise renders fresh.
func (m *Model) ViewCached() string {
	if !m.IsDirty() && m.cachedView != "" {
		return m.cachedView
	}
	m.cachedView = m.View()
	m.ClearDirty()
	return m.cachedView
}

func (m Model) renderNode(node *Node, selected bool, maxWidth int) string {
	// Build indentation (use compact 2-space or normal 4-space)
	indent := ""
	indentStr := theme.TreeSpace // 4 spaces
	if m.compactIndent {
		indentStr = "  " // 2 spaces
	}
	for i := 0; i < node.Depth; i++ {
		indent += indentStr
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
	// Use cached indicator if it's up to date
	if node.GitStatusVersion == m.gitStatusVersion {
		return node.CachedGitIndicator
	}

	// Cache is stale, update it
	m.updateNodeGitIndicator(node)
	return node.CachedGitIndicator
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

// SetCompactIndent sets whether to use compact (2-space) indentation.
func (m *Model) SetCompactIndent(compact bool) {
	m.compactIndent = compact
	m.MarkDirty()
}

// CompactIndent returns whether compact indentation is enabled.
func (m Model) CompactIndent() bool {
	return m.compactIndent
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
	m.gitStatusVersion++

	// Rebuild directory git status cache
	m.dirGitStatusCache = make(map[string]string)
	if status != nil && m.workDir != "" {
		m.buildDirGitStatusCache(status)
	}

	// Update cached indicators on visible nodes
	for _, node := range m.visible {
		m.updateNodeGitIndicator(node)
	}

	// Mark dirty since git status changed
	m.MarkDirty()

	return m
}

// buildDirGitStatusCache precomputes git status indicators for directories
func (m *Model) buildDirGitStatusCache(status *git.Status) {
	// Group files by directory
	dirStatus := make(map[string]struct {
		hasChanges   bool
		allStaged    bool
		hasUntracked bool
	})

	for path, fileStatus := range status.Files {
		// Walk up the directory tree
		dir := filepath.Dir(path)
		for dir != "." && dir != "" {
			ds := dirStatus[dir]
			ds.hasChanges = true

			// Check if untracked
			if fileStatus.Staging == git.StatusUntracked || fileStatus.Worktree == git.StatusUntracked {
				ds.hasUntracked = true
				ds.allStaged = false
			} else if fileStatus.Worktree != git.StatusUnmodified && fileStatus.Worktree != ' ' {
				// Has unstaged changes
				ds.allStaged = false
			} else if !ds.hasUntracked && ds.allStaged {
				ds.allStaged = true
			}

			dirStatus[dir] = ds
			dir = filepath.Dir(dir)
		}
	}

	// Pre-render indicators for all affected directories
	for dir, ds := range dirStatus {
		fullPath := filepath.Join(m.workDir, dir)
		if ds.hasUntracked {
			m.dirGitStatusCache[fullPath] = lipgloss.NewStyle().Foreground(theme.LaserPurple).Render("●")
		} else if ds.allStaged {
			m.dirGitStatusCache[fullPath] = lipgloss.NewStyle().Foreground(theme.MatrixGreen).Render("●")
		} else {
			m.dirGitStatusCache[fullPath] = lipgloss.NewStyle().Foreground(theme.ElectricYellow).Render("●")
		}
	}
}

// updateNodeGitIndicator computes and caches the git indicator for a node
func (m *Model) updateNodeGitIndicator(node *Node) {
	if m.gitStatus == nil || m.workDir == "" {
		node.CachedGitIndicator = ""
		node.GitStatusVersion = m.gitStatusVersion
		return
	}

	// Get relative path from workdir
	relPath, err := filepath.Rel(m.workDir, node.Path)
	if err != nil {
		node.CachedGitIndicator = ""
		node.GitStatusVersion = m.gitStatusVersion
		return
	}

	// Check for exact match (files)
	if status, ok := m.gitStatus.Files[relPath]; ok {
		node.CachedGitIndicator = m.renderGitStatus(status)
		node.GitStatusVersion = m.gitStatusVersion
		return
	}

	// For directories, use cached directory status
	if node.IsDir {
		if cached, ok := m.dirGitStatusCache[node.Path]; ok {
			node.CachedGitIndicator = cached
			node.GitStatusVersion = m.gitStatusVersion
			return
		}
	}

	node.CachedGitIndicator = ""
	node.GitStatusVersion = m.gitStatusVersion
}

// ScrollPercent returns the current scroll position as a percentage (0-100).
func (m Model) ScrollPercent() float64 {
	if len(m.visible) == 0 {
		return 100
	}
	_, h := m.Size()
	viewportHeight := h - 3
	if viewportHeight <= 0 {
		return 100
	}
	maxOffset := len(m.visible) - viewportHeight
	if maxOffset <= 0 {
		return 100
	}
	return float64(m.offset) / float64(maxOffset) * 100
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
