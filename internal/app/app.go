package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/avitaltamir/vibecommander/internal/components/content"
	"github.com/avitaltamir/vibecommander/internal/components/content/viewer"
	"github.com/avitaltamir/vibecommander/internal/components/filetree"
	"github.com/avitaltamir/vibecommander/internal/components/minibuffer"
	"github.com/avitaltamir/vibecommander/internal/components/terminal"
	"github.com/avitaltamir/vibecommander/internal/git"
	"github.com/avitaltamir/vibecommander/internal/layout"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
)

// GitStatusMsg carries updated git status
type GitStatusMsg struct {
	Status *git.Status
	IsRepo bool
}

// FileChangeMsg is sent when the file system changes
type FileChangeMsg struct {
	Path string
	Op   fsnotify.Op
}

// Model is the root application model.
type Model struct {
	// Child components
	fileTree   filetree.Model
	content    content.Model
	miniBuffer minibuffer.Model
	// statusBar  statusbar.Model

	// Focus state
	focus       PanelID
	prevFocus   PanelID
	miniVisible bool
	fullscreen  PanelID // Which panel is fullscreen (0 = none)
	showHelp    bool
	showQuit    bool // Quit confirmation dialog

	// Layout
	layout layout.Layout
	theme  *theme.Theme
	keys   KeyMap

	// Git
	gitProvider    *git.ShellProvider
	gitStatus      *git.Status
	isGitRepo      bool
	workDir        string
	gitRefreshTime time.Time // Last git refresh time for debouncing

	// File watcher
	watcher *fsnotify.Watcher

	// Window dimensions
	width  int
	height int
	ready  bool
}

// New creates a new application model.
func New() Model {
	ft := filetree.New()
	ft = ft.Focus() // File tree starts focused

	// Get current working directory
	workDir, _ := os.Getwd()
	gitProvider := git.NewShellProvider(workDir)

	// Create content pane with git provider
	contentPane := content.New()
	contentPane.SetGitProvider(gitProvider)

	// Create file watcher
	watcher, _ := fsnotify.NewWatcher()

	return Model{
		fileTree:    ft,
		content:     contentPane,
		miniBuffer:  minibuffer.New(),
		focus:       PanelFileTree,
		miniVisible: false,
		theme:       theme.DefaultTheme(),
		keys:        DefaultKeyMap(),
		gitProvider: gitProvider,
		workDir:     workDir,
		isGitRepo:   gitProvider.IsRepo(),
		watcher:     watcher,
	}
}

// Init initializes the application.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.fileTree.Init(),
		m.content.Init(),
		m.miniBuffer.Init(),
		m.refreshGitStatus(),
		gitTick(), // Start periodic git status refresh
	}

	// Start file watcher
	if m.watcher != nil && m.workDir != "" {
		// Recursively watch directories (with exclusions)
		m.addWatchRecursive(m.workDir)
		cmds = append(cmds, m.watchFilesCmd())
	}

	return tea.Batch(cmds...)
}

// addWatchRecursive adds watches for a directory and its subdirectories
func (m Model) addWatchRecursive(root string) {
	if m.watcher == nil {
		return
	}

	// Directories to skip
	skipDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		".venv":        true,
		"__pycache__":  true,
		".cache":       true,
		"dist":         true,
		"build":        true,
	}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// Check if this directory should be skipped
			name := info.Name()
			if skipDirs[name] {
				return filepath.SkipDir
			}

			// Skip hidden directories (except root)
			if path != root && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}

			// Add watch for this directory
			m.watcher.Add(path)
		}
		return nil
	})

	// Note: We don't watch .git directory to avoid lock file conflicts.
	// Git status is refreshed periodically via ticker instead.
}

// watchFilesCmd returns a command that listens for file system changes
func (m Model) watchFilesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.watcher == nil {
			// No watcher, use a ticker to retry later
			time.Sleep(500 * time.Millisecond)
			return FileChangeMsg{} // Empty msg will restart watch
		}

		select {
		case event, ok := <-m.watcher.Events:
			if !ok {
				// Channel closed, sleep and retry
				time.Sleep(500 * time.Millisecond)
				return FileChangeMsg{}
			}
			return FileChangeMsg{Path: event.Name, Op: event.Op}
		case <-m.watcher.Errors:
			// Ignore errors but continue watching
			return FileChangeMsg{}
		case <-time.After(5 * time.Second):
			// Periodic timeout to ensure watch stays alive
			return FileChangeMsg{}
		}
	}
}

// gitDebounceInterval is the minimum time between git refreshes
const gitDebounceInterval = 500 * time.Millisecond

// gitTickInterval is how often we poll git status
const gitTickInterval = 1 * time.Second

// refreshGitStatus fetches the current git status
func (m Model) refreshGitStatus() tea.Cmd {
	return func() tea.Msg {
		if !m.gitProvider.IsRepo() {
			return GitStatusMsg{IsRepo: false}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		status, _ := m.gitProvider.GetStatus(ctx)
		return GitStatusMsg{Status: status, IsRepo: true}
	}
}

// refreshGitStatusDebounced only refreshes if enough time has passed since last refresh
func (m *Model) refreshGitStatusDebounced() tea.Cmd {
	now := time.Now()
	if now.Sub(m.gitRefreshTime) < gitDebounceInterval {
		return nil
	}
	m.gitRefreshTime = now
	return m.refreshGitStatus()
}

type gitTickMsg struct{}

// gitTick returns a command that sends a gitTickMsg after the tick interval
func gitTick() tea.Cmd {
	return tea.Tick(gitTickInterval, func(t time.Time) tea.Msg {
		return gitTickMsg{}
	})
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = layout.Calculate(msg.Width, msg.Height, m.miniVisible)
		m.ready = true
		// Update child component sizes
		m = m.updateSizes()
		return m, nil

	case GitStatusMsg:
		m.isGitRepo = msg.IsRepo
		m.gitStatus = msg.Status
		// Update file tree with git status
		if msg.Status != nil {
			m.fileTree = m.fileTree.SetGitStatus(msg.Status)
		}
		return m, nil

	case gitTickMsg:
		// Periodic git status refresh with debouncing
		var cmds []tea.Cmd
		if cmd := m.refreshGitStatusDebounced(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, gitTick()) // Schedule next tick
		return m, tea.Batch(cmds...)

	case FileChangeMsg:
		// Always continue watching for more events
		cmds = append(cmds, m.watchFilesCmd())

		// Skip empty messages (from error handling)
		if msg.Path == "" {
			return m, tea.Batch(cmds...)
		}

		// If a new directory was created, add a watch for it
		if msg.Op&fsnotify.Create != 0 {
			if info, err := os.Stat(msg.Path); err == nil && info.IsDir() {
				name := filepath.Base(msg.Path)
				if !strings.HasPrefix(name, ".") && name != "node_modules" && name != "vendor" {
					m.watcher.Add(msg.Path)
				}
			}
		}

		// Refresh the directory containing the changed file
		dirPath := filepath.Dir(msg.Path)
		if cmd := m.fileTree.RefreshDir(dirPath); cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Refresh git status (debounced to avoid excessive calls)
		if cmd := m.refreshGitStatusDebounced(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Handle quit dialog first
		if m.showQuit {
			switch msg.String() {
			case "y", "Y", "enter":
				return m, tea.Quit
			case "n", "N", "esc", "q":
				m.showQuit = false
				return m, nil
			}
			return m, nil
		}

		// Handle global keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.showQuit = true
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.FocusTree):
			// Exit fullscreen if in fullscreen mode
			if m.fullscreen != 0 {
				m.fullscreen = 0
				m = m.updateSizes()
			}
			// Close mini buffer if open
			if m.miniVisible {
				m.miniVisible = false
				m.layout = layout.Calculate(m.width, m.height, m.miniVisible)
				m = m.updateSizes()
			}
			m = m.setFocus(PanelFileTree)
			return m, nil

		case key.Matches(msg, m.keys.FocusContent):
			// If content is already fullscreen, just exit fullscreen
			if m.fullscreen == PanelContent {
				m.fullscreen = 0
				m = m.updateSizes()
				return m, nil
			}
			// Close mini buffer if open
			if m.miniVisible {
				m.miniVisible = false
				m.layout = layout.Calculate(m.width, m.height, m.miniVisible)
				m = m.updateSizes()
			}
			// Enter fullscreen if already focused
			if m.focus == PanelContent {
				m.fullscreen = PanelContent
				m = m.updateSizes()
			}
			m = m.setFocus(PanelContent)
			return m, nil

		case key.Matches(msg, m.keys.ToggleMini):
			// Exit fullscreen if in fullscreen mode
			if m.fullscreen != 0 {
				m.fullscreen = 0
				m = m.updateSizes()
			}
			// Always toggle terminal on/off
			m.miniVisible = !m.miniVisible
			m.layout = layout.Calculate(m.width, m.height, m.miniVisible)
			m = m.updateSizes()
			if m.miniVisible {
				m = m.setFocus(PanelMiniBuffer)
				// Start shell if not running
				if !m.miniBuffer.Running() {
					return m, m.miniBuffer.StartShell()
				}
			} else {
				m = m.setFocus(m.prevFocus)
			}
			return m, nil

		case key.Matches(msg, m.keys.LaunchAI):
			// If terminal is running in content pane, pass Ctrl+A to it instead
			if m.focus == PanelContent && m.content.IsTerminalRunning() {
				break // Let it fall through to be routed to focused component
			}
			// Launch AI assistant in content pane
			m = m.setFocus(PanelContent)
			return m, func() tea.Msg {
				return content.LaunchAIMsg{
					Command: "claude",
					Args:    []string{},
				}
			}

		case key.Matches(msg, m.keys.CycleTheme):
			// Cycle to next theme
			theme.NextTheme()
			return m, nil
		}

		// Any other key closes help
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

	case FocusMsg:
		m = m.setFocus(msg.Target)
		return m, nil

	case ToggleMiniBufferMsg:
		m.miniVisible = !m.miniVisible
		m.layout = layout.Calculate(m.width, m.height, m.miniVisible)
		m = m.updateSizes()
		return m, nil

	case filetree.SelectMsg:
		// File selected in file tree - open it in content pane
		if !msg.IsDir {
			return m, func() tea.Msg {
				return content.OpenFileMsg{Path: msg.Path}
			}
		}
		return m, nil

	case filetree.LoadedMsg:
		// Route to file tree
		var cmd tea.Cmd
		m.fileTree, cmd = m.fileTree.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case content.OpenFileMsg:
		// Route to content pane
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case viewer.FileLoadedMsg:
		// Route to content pane
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case content.FileWithDiffMsg:
		// Route to content pane (file loaded with diff check)
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case minibuffer.OutputMsg:
		// Route to mini buffer (for shell terminal)
		var cmd tea.Cmd
		m.miniBuffer, cmd = m.miniBuffer.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case minibuffer.ExitMsg:
		// Route to mini buffer and refresh git status (command completed)
		var cmd tea.Cmd
		m.miniBuffer, cmd = m.miniBuffer.Update(msg)
		cmds = append(cmds, cmd)
		// Refresh git status in case a git command was run
		cmds = append(cmds, m.refreshGitStatus())
		return m, tea.Batch(cmds...)

	case content.LaunchAIMsg:
		// Route to content pane
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case terminal.OutputMsg, terminal.ExitMsg:
		// Route to content pane (for AI terminal)
		// Note: content.Update already calls ContinueReading() when needed
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// Route messages to focused component
	cmd := m.routeToFocused(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updateSizes updates the size of all child components.
func (m Model) updateSizes() Model {
	// Handle fullscreen mode (content only)
	if m.fullscreen == PanelContent {
		fullWidth := m.width - 2
		fullHeight := m.height - 1 - 2 // minus status bar and borders
		if fullWidth < 0 {
			fullWidth = 0
		}
		if fullHeight < 0 {
			fullHeight = 0
		}
		m.content = m.content.SetSize(fullWidth, fullHeight)
		return m
	}

	// Account for borders
	leftWidth := m.layout.LeftWidth - 2
	rightWidth := m.layout.RightWidth - 2
	mainHeight := m.layout.MainHeight - 2

	if leftWidth < 0 {
		leftWidth = 0
	}
	if rightWidth < 0 {
		rightWidth = 0
	}
	if mainHeight < 0 {
		mainHeight = 0
	}

	m.fileTree = m.fileTree.SetSize(leftWidth, mainHeight)
	m.content = m.content.SetSize(rightWidth, mainHeight)

	// Size mini buffer if visible
	if m.miniVisible {
		miniWidth := m.layout.TotalWidth - 2
		miniHeight := m.layout.MiniHeight - 2
		if miniWidth < 0 {
			miniWidth = 0
		}
		if miniHeight < 0 {
			miniHeight = 0
		}
		m.miniBuffer = m.miniBuffer.SetSize(miniWidth, miniHeight)
	}

	return m
}

// routeToFocused routes a message to the focused component.
func (m *Model) routeToFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch m.focus {
	case PanelFileTree:
		m.fileTree, cmd = m.fileTree.Update(msg)
	case PanelContent:
		m.content, cmd = m.content.Update(msg)
	case PanelMiniBuffer:
		m.miniBuffer, cmd = m.miniBuffer.Update(msg)
	}

	return cmd
}

// View renders the application.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var view string

	// Handle fullscreen mode (content only)
	if m.fullscreen == PanelContent {
		statusBar := m.renderStatusBar()
		panel := m.renderFullscreenPanel(m.content.View(), "", true)
		view = lipgloss.JoinVertical(lipgloss.Left, panel, statusBar)
	} else {
		// Render panels
		leftPanel := m.renderLeftPanel()
		rightPanel := m.renderRightPanel()

		// Join horizontally
		mainArea := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

		// Add mini buffer if visible
		if m.miniVisible {
			miniPanel := m.renderMiniBuffer()
			mainArea = lipgloss.JoinVertical(lipgloss.Left, mainArea, miniPanel)
		}

		// Add status bar
		statusBar := m.renderStatusBar()

		view = lipgloss.JoinVertical(lipgloss.Left, mainArea, statusBar)
	}

	// Show help overlay if active
	if m.showHelp {
		return m.renderHelpOverlay(view)
	}

	// Show quit confirmation dialog
	if m.showQuit {
		return m.renderQuitDialog(view)
	}

	return view
}

// renderLeftPanel renders the file tree panel.
func (m Model) renderLeftPanel() string {
	style := theme.GetPanelStyle(m.focus == PanelFileTree).
		Width(m.layout.LeftWidth - 2).
		Height(m.layout.MainHeight - 2)

	title := theme.RenderTitle("FILES", m.focus == PanelFileTree)

	return style.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.fileTree.View(),
	))
}

// renderRightPanel renders the content panel.
func (m Model) renderRightPanel() string {
	style := theme.GetPanelStyle(m.focus == PanelContent).
		Width(m.layout.RightWidth - 2).
		Height(m.layout.MainHeight - 2)

	return style.Render(m.content.View())
}

// renderMiniBuffer renders the mini buffer panel.
func (m Model) renderMiniBuffer() string {
	style := theme.GetPanelStyle(m.focus == PanelMiniBuffer).
		Width(m.layout.TotalWidth - 2).
		Height(m.layout.MiniHeight - 2)

	title := theme.RenderTitle("TERMINAL", m.focus == PanelMiniBuffer)

	return style.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.miniBuffer.View(),
	))
}

// renderFullscreenPanel renders a panel in fullscreen mode.
func (m Model) renderFullscreenPanel(content string, title string, focused bool) string {
	// Use full width and height minus status bar
	fullHeight := m.height - 1
	style := theme.GetPanelStyle(focused).
		Width(m.width - 2).
		Height(fullHeight - 2)

	if title != "" {
		titleBar := theme.RenderTitle(title, focused)
		return style.Render(lipgloss.JoinVertical(lipgloss.Left,
			titleBar,
			content,
		))
	}
	return style.Render(content)
}

// renderStatusBar renders the status bar.
func (m Model) renderStatusBar() string {
	style := theme.StatusBarStyle.Width(m.layout.TotalWidth)

	// Branch info
	var branch string
	if m.isGitRepo && m.gitStatus != nil && m.gitStatus.Branch != "" {
		branchIcon := lipgloss.NewStyle().
			Foreground(theme.MagentaBlaze).
			Render(theme.GitBranchIcon)
		branchName := lipgloss.NewStyle().
			Foreground(theme.CyberCyan).
			Render(" " + m.gitStatus.Branch)

		// Add dirty indicator
		var dirty string
		if m.gitStatus.IsDirty {
			dirty = lipgloss.NewStyle().
				Foreground(theme.ElectricYellow).
				Render(" ●")
		}

		// Add ahead/behind
		var aheadBehind string
		if m.gitStatus.Ahead > 0 {
			aheadBehind += lipgloss.NewStyle().
				Foreground(theme.MatrixGreen).
				Render(" ↑" + itoa(m.gitStatus.Ahead))
		}
		if m.gitStatus.Behind > 0 {
			aheadBehind += lipgloss.NewStyle().
				Foreground(theme.NeonRed).
				Render(" ↓" + itoa(m.gitStatus.Behind))
		}

		branch = " " + branchIcon + branchName + dirty + aheadBehind
	}

	// Panel indicator
	panelInfo := lipgloss.NewStyle().
		Foreground(theme.MutedLavender).
		Render(" │ " + m.focus.String())

	// Help hint
	help := lipgloss.NewStyle().
		Foreground(theme.DimPurple).
		Render(" │ ^H help │ ^Q quit")

	// Theme name
	themeName := lipgloss.NewStyle().
		Foreground(theme.LaserPurple).
		Render(theme.CurrentTheme().Name)

	// Version
	version := lipgloss.NewStyle().
		Foreground(theme.DimPurple).
		Render("v0.1.0")

	// Layout the status bar
	left := branch + panelInfo + help
	right := themeName + " │ " + version

	gap := m.layout.TotalWidth - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 0 {
		gap = 0
	}

	return style.Render(left + lipgloss.NewStyle().Width(gap).Render("") + right)
}

// itoa converts int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var s string
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// setFocus changes focus to the specified panel.
func (m Model) setFocus(target PanelID) Model {
	// Blur previously focused component
	switch m.focus {
	case PanelFileTree:
		m.fileTree = m.fileTree.Blur()
	case PanelContent:
		m.content = m.content.Blur()
	case PanelMiniBuffer:
		m.miniBuffer = m.miniBuffer.Blur()
	}

	m.prevFocus = m.focus
	m.focus = target

	// Focus new component
	switch target {
	case PanelFileTree:
		m.fileTree = m.fileTree.Focus()
	case PanelContent:
		m.content = m.content.Focus()
	case PanelMiniBuffer:
		m.miniBuffer = m.miniBuffer.Focus()
	}

	return m
}

// Focus returns the currently focused panel.
func (m Model) Focus() PanelID {
	return m.focus
}

// MiniVisible returns whether the mini buffer is visible.
func (m Model) MiniVisible() bool {
	return m.miniVisible
}

// renderHelpOverlay renders the help overlay on top of the existing view.
func (m Model) renderHelpOverlay(_ string) string {
	// Help content
	helpLines := []string{
		"╔══════════════════════════════════════════════════════════╗",
		"║                  VIBE COMMANDER HELP                     ║",
		"╠══════════════════════════════════════════════════════════╣",
		"║  NAVIGATION                                              ║",
		"║    ↑/k, ↓/j      Move up/down                            ║",
		"║    ←/h, →/l      Collapse/Expand or navigate             ║",
		"║    Enter         Select file or toggle directory         ║",
		"║    PgUp/PgDn     Page up/down                            ║",
		"║    Home/g        Go to top                               ║",
		"║    End/G         Go to bottom                            ║",
		"║                                                          ║",
		"║  PANELS                                                  ║",
		"║    Alt+1         Focus file tree                         ║",
		"║    Alt+2         Focus content (2x = fullscreen)         ║",
		"║    Alt+3         Toggle terminal                         ║",
		"║                                                          ║",
		"║  VIEWER (when viewing a file)                            ║",
		"║    /             Search (regex)                          ║",
		"║    Enter         Search / Next match                     ║",
		"║    n/p           Next / Previous match                   ║",
		"║    Esc           Cancel search                           ║",
		"║                                                          ║",
		"║  ACTIONS                                                 ║",
		"║    Alt+A         Launch AI assistant                     ║",
		"║    Alt+T         Cycle theme                             ║",
		"║    Ctrl+H        Toggle this help                        ║",
		"║    Ctrl+Q        Quit                                    ║",
		"║                                                          ║",
		"║              Press any key to close                      ║",
		"╚══════════════════════════════════════════════════════════╝",
	}

	helpContent := lipgloss.JoinVertical(lipgloss.Left, helpLines...)

	helpStyle := lipgloss.NewStyle().
		Foreground(theme.CyberCyan).
		Bold(true).
		Padding(1, 2)

	helpBox := helpStyle.Render(helpContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)
}

// renderQuitDialog renders the quit confirmation dialog.
func (m Model) renderQuitDialog(_ string) string {
	quitLines := []string{
		"╔════════════════════════════════════╗",
		"║       QUIT VIBE COMMANDER?         ║",
		"╠════════════════════════════════════╣",
		"║                                    ║",
		"║    Are you sure you want to quit?  ║",
		"║                                    ║",
		"║       [Y]es          [N]o          ║",
		"║                                    ║",
		"╚════════════════════════════════════╝",
	}

	quitContent := lipgloss.JoinVertical(lipgloss.Left, quitLines...)

	quitStyle := lipgloss.NewStyle().
		Foreground(theme.CyberCyan).
		Bold(true).
		Padding(1, 2)

	quitBox := quitStyle.Render(quitContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		quitBox,
	)
}
