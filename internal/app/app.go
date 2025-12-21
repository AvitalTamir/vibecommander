package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/avitaltamir/vibecommander/internal/components/content"
	"github.com/avitaltamir/vibecommander/internal/components/content/viewer"
	"github.com/avitaltamir/vibecommander/internal/components/filetree"
	"github.com/avitaltamir/vibecommander/internal/components/gitpanel"
	"github.com/avitaltamir/vibecommander/internal/components/minibuffer"
	"github.com/avitaltamir/vibecommander/internal/components/terminal"
	"github.com/avitaltamir/vibecommander/internal/git"
	"github.com/avitaltamir/vibecommander/internal/layout"
	"github.com/avitaltamir/vibecommander/internal/state"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"github.com/fsnotify/fsnotify"
)

// Version is the application version, set at build time via ldflags
var Version = "dev"

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

// gitCommitFinishedMsg is sent when git commit completes (after GPG, etc.)
type gitCommitFinishedMsg struct {
	err error
}

// Model is the root application model.
type Model struct {
	// Child components
	fileTree   filetree.Model
	gitPanel   gitpanel.Model
	content    content.Model
	miniBuffer minibuffer.Model
	// statusBar  statusbar.Model

	// Focus state
	focus           PanelID
	prevFocus       PanelID
	miniVisible     bool
	gitPanelVisible bool
	fullscreen      PanelID // Which panel is fullscreen (PanelNone = none)
	showHelp        bool
	showQuit        bool      // Quit confirmation dialog
	lastQuitPress   time.Time // For double-tap ctrl+q detection

	// Layout
	layout           layout.Layout
	leftPanelPercent int  // Dynamic width percentage for file tree (15-60)
	resizingPanel    bool // Whether user is dragging to resize panels
	theme            *theme.Theme
	keys             KeyMap

	// Git
	gitProvider    *git.ShellProvider
	gitStatus      *git.Status
	isGitRepo      bool
	workDir        string
	gitRefreshTime time.Time // Last git refresh time for debouncing

	// File watcher
	watcher              *fsnotify.Watcher
	lastFileChangeTime   time.Time              // Last file change time for debouncing
	pendingFileChanges   map[string]fsnotify.Op // Pending file changes to process
	fileChangeDebouncing bool                   // Whether we're waiting to process file changes

	// Window dimensions
	width  int
	height int
	ready  bool

	// State persistence
	aiLaunched      bool // Track if AI was launched this session
	restoreAI       bool // Flag to restore AI on first window size msg
	initialThemeIdx int  // Theme index to restore

	// AI assistant selection
	aiCommand       string   // Persisted AI command (e.g., "claude", "gemini")
	aiArgs          []string // Persisted AI args
	showAIDialog    bool     // Whether AI selection dialog is visible
	aiDialogIndex   int      // Current selection in dialog (0=Claude, 1=Gemini, 2=Codex, 3=Other)
	aiDialogCustom  string   // Custom command input when "Other" selected
	aiDialogEditing bool     // True when editing custom command in "Other"

	// Commit dialog
	showCommitDialog bool            // Whether commit dialog is visible
	commitInput      textinput.Model // Commit message text input
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

	// Load persisted state (global)
	savedState := state.Load()

	// Apply saved theme
	theme.SetThemeIndex(savedState.ThemeIndex)

	// Apply saved left panel percent (validate it's within bounds)
	leftPanelPercent := savedState.LeftPanelPercent
	if leftPanelPercent < layout.MinLeftPanelPercent || leftPanelPercent > layout.MaxLeftPanelPercent {
		leftPanelPercent = layout.DefaultLeftPanelPercent
	}

	// Apply saved compact indent to file tree
	ft.SetCompactIndent(savedState.CompactIndent)

	// Initialize commit message input
	commitInput := textinput.New()
	commitInput.Placeholder = ""
	commitInput.CharLimit = 200
	commitInput.Prompt = ""

	return Model{
		fileTree:           ft,
		gitPanel:           gitpanel.New(),
		content:            contentPane,
		miniBuffer:         minibuffer.New(),
		focus:              PanelFileTree,
		miniVisible:        false,
		gitPanelVisible:    false,
		leftPanelPercent:   leftPanelPercent,
		theme:              theme.DefaultTheme(),
		keys:               DefaultKeyMap(),
		gitProvider:        gitProvider,
		workDir:            workDir,
		isGitRepo:          gitProvider.IsRepo(),
		watcher:            watcher,
		pendingFileChanges: make(map[string]fsnotify.Op),
		restoreAI:          savedState.AIWindowOpen,
		initialThemeIdx:    savedState.ThemeIndex,
		aiCommand:          savedState.AICommand,
		aiArgs:             savedState.AIArgs,
		commitInput:        commitInput,
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
			return nil // No watcher, don't keep retrying
		}

		// Block until we get an actual file change event
		// Don't use timeouts - they cause unnecessary redraws
		for {
			select {
			case event, ok := <-m.watcher.Events:
				if !ok {
					return nil // Channel closed
				}
				return FileChangeMsg{Path: event.Name, Op: event.Op}
			case <-m.watcher.Errors:
				// Ignore errors but continue watching
				continue
			}
		}
	}
}

// gitDebounceInterval is the minimum time between git refreshes
const gitDebounceInterval = 2 * time.Second

// gitTickInterval is how often we poll git status
const gitTickInterval = 10 * time.Second

// fileChangeDebounceInterval is the minimum time between file change processing
const fileChangeDebounceInterval = 500 * time.Millisecond

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
type gitRefreshMsg struct{} // Immediate git refresh request

// gitTick returns a command that sends a gitTickMsg after the tick interval
func gitTick() tea.Cmd {
	return tea.Tick(gitTickInterval, func(t time.Time) tea.Msg {
		return gitTickMsg{}
	})
}

// fileChangeDebounceMsg is sent after the debounce interval to process pending file changes
type fileChangeDebounceMsg struct{}

// scheduleFileChangeDebounce schedules processing of pending file changes
func (m *Model) scheduleFileChangeDebounce() tea.Cmd {
	if m.fileChangeDebouncing {
		return nil
	}
	m.fileChangeDebouncing = true
	return tea.Tick(fileChangeDebounceInterval, func(t time.Time) tea.Msg {
		return fileChangeDebounceMsg{}
	})
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = layout.Calculate(msg.Width, msg.Height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
		wasReady := m.ready
		m.ready = true
		// Update child component sizes
		m = m.updateSizes()

		// Restore AI window on first ready (if it was open before)
		if !wasReady && m.restoreAI {
			m.restoreAI = false // Only restore once
			m.aiLaunched = true
			m = m.setFocus(PanelContent)
			return m, func() tea.Msg {
				return content.LaunchAIMsg{
					Command: "claude",
					Args:    []string{},
				}
			}
		}
		return m, nil

	case GitStatusMsg:
		// Skip update if nothing changed (reduces flickering)
		if m.isGitRepo == msg.IsRepo && gitStatusEqual(m.gitStatus, msg.Status) {
			return m, nil
		}
		m.isGitRepo = msg.IsRepo
		m.gitStatus = msg.Status
		// Update file tree and git panel with git status
		if msg.Status != nil {
			m.fileTree = m.fileTree.SetGitStatus(msg.Status)
			m.gitPanel = m.gitPanel.SetGitStatus(msg.Status)
		}
		return m, nil

	case gitTickMsg:
		// Schedule next tick - always do this
		nextTick := gitTick()
		// Only refresh if debounce allows
		if cmd := m.refreshGitStatusDebounced(); cmd != nil {
			return m, tea.Batch(cmd, nextTick)
		}
		// Just reschedule, no state change needed
		return m, nextTick

	case gitRefreshMsg:
		// Immediate git refresh (after staging/unstaging)
		return m, m.refreshGitStatus()

	case gitCommitFinishedMsg:
		// Git commit finished (after GPG passphrase entry, etc.)
		if msg.err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{Err: msg.err}
			}
		}
		// Refresh git status after successful commit
		return m, m.refreshGitStatus()

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

		// Add to pending file changes (debounce multiple rapid changes)
		m.pendingFileChanges[msg.Path] = msg.Op

		// Schedule debounced processing
		if cmd := m.scheduleFileChangeDebounce(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case fileChangeDebounceMsg:
		// Process all pending file changes
		m.fileChangeDebouncing = false

		// Collect unique directories to refresh
		dirsToRefresh := make(map[string]bool)
		for path := range m.pendingFileChanges {
			dirPath := filepath.Dir(path)
			dirsToRefresh[dirPath] = true
		}

		// Clear pending changes
		m.pendingFileChanges = make(map[string]fsnotify.Op)

		// Refresh each affected directory
		for dirPath := range dirsToRefresh {
			if cmd := m.fileTree.RefreshDir(dirPath); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Refresh git status (debounced to avoid excessive calls)
		if cmd := m.refreshGitStatusDebounced(); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		// Handle quit dialog first
		if m.showQuit {
			switch msg.String() {
			case "y", "Y", "enter", "ctrl+q":
				// Save state before quitting
				m.saveState()
				return m, tea.Quit
			case "n", "N", "esc":
				m.showQuit = false
				return m, nil
			}
			return m, nil
		}

		// Handle commit dialog
		if m.showCommitDialog {
			return m.handleCommitDialog(msg)
		}

		// Handle AI dialog
		if m.showAIDialog {
			return m.handleAIDialog(msg)
		}

		// Handle global keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			// Check for double-tap ctrl+q (within 400ms) for immediate quit
			now := time.Now()
			if now.Sub(m.lastQuitPress) < 400*time.Millisecond {
				m.saveState()
				return m, tea.Quit
			}
			m.lastQuitPress = now
			m.showQuit = true
			return m, nil

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.FocusTree):
			// Exit fullscreen if in fullscreen mode
			if m.fullscreen != PanelNone {
				m.fullscreen = PanelNone
				m = m.updateSizes()
			}
			// Close mini buffer if open
			if m.miniVisible {
				m.miniVisible = false
				m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
				m = m.updateSizes()
			}
			m = m.setFocus(PanelFileTree)
			return m, nil

		case key.Matches(msg, m.keys.FocusContent):
			// If content is already fullscreen, just exit fullscreen
			if m.fullscreen == PanelContent {
				m.fullscreen = PanelNone
				m = m.updateSizes()
				return m, nil
			}
			// Close mini buffer if open
			if m.miniVisible {
				m.miniVisible = false
				m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
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
			if m.fullscreen != PanelNone {
				m.fullscreen = PanelNone
				m = m.updateSizes()
			}
			// If terminal is visible but not focused, just focus it
			if m.miniVisible && m.focus != PanelMiniBuffer {
				m = m.setFocus(PanelMiniBuffer)
				return m, nil
			}
			// Otherwise toggle terminal on/off
			m.miniVisible = !m.miniVisible
			m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
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

		case key.Matches(msg, m.keys.ToggleGitPanel):
			// If git panel is visible but not focused, just focus it
			if m.gitPanelVisible && m.focus != PanelGitPanel {
				m = m.setFocus(PanelGitPanel)
				return m, nil
			}
			// Otherwise toggle git panel visibility
			m.gitPanelVisible = !m.gitPanelVisible
			m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
			m = m.updateSizes()
			if m.gitPanelVisible {
				// Focus git panel when opened
				m = m.setFocus(PanelGitPanel)
			} else {
				// Return focus to file tree when closed
				if m.focus == PanelGitPanel {
					m = m.setFocus(PanelFileTree)
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.SelectAI):
			// Show AI selection dialog
			m.showAIDialog = true
			m.aiDialogIndex = 0
			return m, nil

		case key.Matches(msg, m.keys.LaunchAI):
			// If terminal is running in content pane, pass Ctrl+A to it instead
			if m.focus == PanelContent && m.content.IsTerminalRunning() {
				break // Let it fall through to be routed to focused component
			}
			// If no AI is configured, show selection dialog
			if m.aiCommand == "" {
				m.showAIDialog = true
				m.aiDialogIndex = 0
				return m, nil
			}
			// Launch AI assistant in content pane
			m.aiLaunched = true
			m = m.setFocus(PanelContent)
			return m, func() tea.Msg {
				return content.LaunchAIMsg{
					Command: m.aiCommand,
					Args:    m.aiArgs,
				}
			}

		case key.Matches(msg, m.keys.CycleTheme):
			// Cycle to next theme
			theme.NextTheme()
			return m, nil

		case key.Matches(msg, m.keys.ShrinkTree):
			// Shrink file tree by 5%
			m.leftPanelPercent -= 5
			if m.leftPanelPercent < layout.MinLeftPanelPercent {
				m.leftPanelPercent = layout.MinLeftPanelPercent
			}
			m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
			m = m.updateSizes()
			return m, nil

		case key.Matches(msg, m.keys.WidenTree):
			// Widen file tree by 5%
			m.leftPanelPercent += 5
			if m.leftPanelPercent > layout.MaxLeftPanelPercent {
				m.leftPanelPercent = layout.MaxLeftPanelPercent
			}
			m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
			m = m.updateSizes()
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
		m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
		m = m.updateSizes()
		return m, nil

	case filetree.SelectMsg:
		// File selected in file tree - open it in content pane and focus viewer
		if !msg.IsDir {
			m = m.setFocus(PanelContent)
			return m, func() tea.Msg {
				return content.OpenFileMsg{Path: msg.Path}
			}
		}
		return m, nil

	case filetree.StageToggleMsg:
		// Toggle staging for a file from file tree
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var err error
			if msg.IsStaged {
				err = m.gitProvider.Unstage(ctx, msg.Path)
			} else {
				err = m.gitProvider.Stage(ctx, msg.Path)
			}
			if err != nil {
				return ErrorMsg{Err: err}
			}
			// Trigger git status refresh
			return gitRefreshMsg{}
		}

	case gitpanel.StageToggleMsg:
		// Toggle staging for a file
		return m, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var err error
			if msg.IsStaged {
				err = m.gitProvider.Unstage(ctx, msg.Path)
			} else {
				err = m.gitProvider.Stage(ctx, msg.Path)
			}
			if err != nil {
				return ErrorMsg{Err: err}
			}
			// Trigger git status refresh
			return gitRefreshMsg{}
		}

	case gitpanel.OpenCommitMsg:
		// Open commit dialog
		m.showCommitDialog = true
		m.commitInput.SetValue("")
		m.commitInput.Focus()
		return m, textinput.Blink

	case gitpanel.OpenFileMsg:
		// Open file from git panel - same as file tree behavior
		fullPath := filepath.Join(m.workDir, msg.Path)
		m = m.setFocus(PanelContent)
		return m, func() tea.Msg {
			return content.OpenFileMsg{Path: fullPath}
		}

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

	case tea.MouseClickMsg:
		// Handle mouse clicks to focus panes and interact with content
		mouse := msg.Mouse()
		if mouse.Button == tea.MouseLeft {
			// Check if clicking on the border between file tree and content for resize
			borderX := m.layout.LeftWidth
			if mouse.X >= borderX-1 && mouse.X <= borderX+1 && m.fullscreen == PanelNone {
				m.resizingPanel = true
				return m, nil
			}

			targetPanel := m.panelAtPosition(mouse.X, mouse.Y)

			// Check for header click on content panel to switch sources
			if targetPanel == PanelContent && m.content.HasMultipleSources() {
				if source := m.detectHeaderClick(mouse.X, mouse.Y); source != content.SourceNone {
					// Switch to the clicked source
					var cmd tea.Cmd
					m.content, cmd = m.content.Update(content.SwitchSourceMsg{Source: source})
					if cmd != nil {
						cmds = append(cmds, cmd)
					}
					// Also focus the content panel if not already focused
					if m.focus != PanelContent {
						m = m.setFocus(PanelContent)
					}
					return m, tea.Batch(cmds...)
				}
			}

			if targetPanel != PanelNone && targetPanel != m.focus {
				m = m.setFocus(targetPanel)
			}
		}
		// Route click to the appropriate panel
		cmd := m.routeMouseClickToPanel(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.MouseMotionMsg:
		// Handle panel resize dragging
		if m.resizingPanel {
			mouse := msg.Mouse()
			// Calculate new percentage based on mouse X position
			if m.width > 0 {
				newPercent := mouse.X * 100 / m.width
				if newPercent < layout.MinLeftPanelPercent {
					newPercent = layout.MinLeftPanelPercent
				}
				if newPercent > layout.MaxLeftPanelPercent {
					newPercent = layout.MaxLeftPanelPercent
				}
				m.leftPanelPercent = newPercent
				m.layout = layout.Calculate(m.width, m.height, m.miniVisible, m.leftPanelPercent, m.gitPanelVisible)
				m = m.updateSizes()
			}
			return m, nil
		}
		// Route to appropriate panel
		cmd := m.routeMouseToPanel(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.MouseReleaseMsg:
		// Stop panel resize dragging
		if m.resizingPanel {
			m.resizingPanel = false
			return m, nil
		}
		// Route to appropriate panel
		cmd := m.routeMouseToPanel(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.MouseWheelMsg:
		// Route wheel events to the appropriate panel for scrolling
		cmd := m.routeMouseToPanel(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
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

	// Calculate left panel heights (file tree and git panel)
	fileTreeHeight := m.layout.FileTreeHeight - 2
	gitPanelHeight := m.layout.GitPanelHeight - 2
	if fileTreeHeight < 0 {
		fileTreeHeight = 0
	}
	if gitPanelHeight < 0 {
		gitPanelHeight = 0
	}

	m.fileTree = m.fileTree.SetSize(leftWidth, fileTreeHeight)
	if m.gitPanelVisible {
		m.gitPanel = m.gitPanel.SetSize(leftWidth, gitPanelHeight)
	}
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
	case PanelGitPanel:
		m.gitPanel, cmd = m.gitPanel.Update(msg)
	case PanelContent:
		m.content, cmd = m.content.Update(msg)
	case PanelMiniBuffer:
		m.miniBuffer, cmd = m.miniBuffer.Update(msg)
	}

	return cmd
}

// View renders the application.
func (m Model) View() tea.View {
	if !m.ready {
		v := tea.NewView("Initializing...")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	var view string

	// Handle fullscreen mode (content only)
	if m.fullscreen == PanelContent {
		statusBar := m.renderStatusBar()
		title, scrollPercent := m.content.TitleInfo()
		isRunning := m.content.IsTerminalRunning()
		mode := m.content.Mode()

		opts := theme.PanelTitleOptions{
			Title:         title,
			StatusRunning: isRunning,
			ShowStatus:    mode == content.ModeAI || mode == content.ModeTerminal,
			ScrollPercent: scrollPercent,
		}

		panel := theme.RenderPanelWithTitle(
			m.content.ContentView(),
			opts,
			m.width,
			m.height-1,
			true,
		)
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
		v := tea.NewView(m.renderHelpOverlay(view))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	// Show quit confirmation dialog
	if m.showQuit {
		v := tea.NewView(m.renderQuitDialog(view))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	// Show AI selection dialog
	if m.showAIDialog {
		v := tea.NewView(m.renderAIDialog(view))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	// Show commit dialog
	if m.showCommitDialog {
		v := tea.NewView(m.renderCommitDialog(view))
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	v := tea.NewView(view)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// renderLeftPanel renders the file tree panel (and git panel if visible).
func (m Model) renderLeftPanel() string {
	fileTreeFocused := m.focus == PanelFileTree

	var fileTreeHints string
	if fileTreeFocused {
		bottomHints := "↑↓:nav  enter:open"
		if m.isGitRepo {
			bottomHints += "  space:stage"
		}
		fileTreeHints = bottomHints
	}

	fileTreeOpts := theme.PanelTitleOptions{
		Title:         "FILES",
		ScrollPercent: -1, // Don't show scroll indicator
		BottomHints:   fileTreeHints,
	}

	fileTreePanel := theme.RenderPanelWithTitle(
		m.fileTree.View(),
		fileTreeOpts,
		m.layout.LeftWidth,
		m.layout.FileTreeHeight,
		fileTreeFocused,
	)

	// If git panel is not visible, just return file tree
	if !m.gitPanelVisible {
		return fileTreePanel
	}

	// Render git panel
	gitPanelFocused := m.focus == PanelGitPanel

	var gitPanelHints string
	if gitPanelFocused {
		gitPanelHints = "space:stage  c:commit"
	}

	// Build git panel title with counts
	stagedCount := m.gitPanel.StagedCount()
	unstagedCount := m.gitPanel.UnstagedCount()
	gitTitle := "GIT"
	if stagedCount > 0 || unstagedCount > 0 {
		gitTitle = "GIT " +
			theme.GitStatusAdded.Render("●"+string(rune('0'+stagedCount))) + " " +
			theme.GitStatusModified.Render("○"+string(rune('0'+unstagedCount)))
	}

	gitPanelOpts := theme.PanelTitleOptions{
		Title:         gitTitle,
		ScrollPercent: -1,
		BottomHints:   gitPanelHints,
	}

	gitPanel := theme.RenderPanelWithTitle(
		m.gitPanel.View(),
		gitPanelOpts,
		m.layout.LeftWidth,
		m.layout.GitPanelHeight,
		gitPanelFocused,
	)

	// Join file tree and git panel vertically
	return lipgloss.JoinVertical(lipgloss.Left, fileTreePanel, gitPanel)
}

// renderRightPanel renders the content panel.
func (m Model) renderRightPanel() string {
	focused := m.focus == PanelContent

	mode := m.content.Mode()

	// Determine bottom hints based on mode (only when focused)
	var bottomHints string
	if focused {
		switch mode {
		case content.ModeViewer:
			if m.content.HasActiveSearch() {
				bottomHints = "n/p:search  esc:clear"
			} else {
				bottomHints = "↑↓:scroll  /:search"
			}
		case content.ModeDiff:
			bottomHints = "↑↓:scroll"
		}
	}

	// Build panel options using sources info for dual-header support
	sources := m.content.SourcesInfo()
	opts := m.buildContentPanelOpts(sources, bottomHints)

	return theme.RenderPanelWithTitle(
		m.content.ContentView(),
		opts,
		m.layout.RightWidth,
		m.layout.MainHeight,
		focused,
	)
}

// buildContentPanelOpts builds PanelTitleOptions from content sources.
// This supports the dual-header display with file and AI titles.
func (m Model) buildContentPanelOpts(sources []content.SourceInfo, bottomHints string) theme.PanelTitleOptions {
	opts := theme.PanelTitleOptions{
		BottomHints: bottomHints,
	}

	if len(sources) == 0 {
		// No content loaded yet
		opts.Title = "VIEWER"
		opts.ScrollPercent = -1
		return opts
	}

	// Single source - simple case
	if len(sources) == 1 {
		src := sources[0]
		opts.Title = src.Title
		opts.ScrollPercent = src.ScrollPercent
		opts.StatusRunning = src.IsRunning
		opts.ShowStatus = src.Source == content.SourceAI
		opts.PrimaryActive = true
		return opts
	}

	// Dual sources - file (primary) and AI (secondary)
	var fileSource, aiSource *content.SourceInfo
	for i := range sources {
		if sources[i].Source == content.SourceFile {
			fileSource = &sources[i]
		} else if sources[i].Source == content.SourceAI {
			aiSource = &sources[i]
		}
	}

	// File is always primary (left), AI is secondary (right)
	if fileSource != nil {
		opts.Title = fileSource.Title
		opts.PrimaryActive = fileSource.IsActive
		if fileSource.IsActive {
			opts.ScrollPercent = fileSource.ScrollPercent
		} else {
			opts.ScrollPercent = -1
		}
	}

	if aiSource != nil {
		opts.SecondaryTitle = aiSource.Title
		opts.SecondaryActive = aiSource.IsActive
		opts.SecondaryShowStatus = true
		opts.SecondaryStatusRunning = aiSource.IsRunning
	}

	return opts
}

// renderMiniBuffer renders the mini buffer panel.
func (m Model) renderMiniBuffer() string {
	focused := m.focus == PanelMiniBuffer

	opts := theme.PanelTitleOptions{
		Title:         "TERMINAL",
		StatusRunning: m.miniBuffer.Running(),
		ShowStatus:    true,
		ScrollPercent: -1, // Don't show scroll for terminal
	}

	return theme.RenderPanelWithTitle(
		m.miniBuffer.View(),
		opts,
		m.layout.TotalWidth,
		m.layout.MiniHeight,
		focused,
	)
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
		Render(Version)

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

// gitStatusEqual compares two git statuses for equality
func gitStatusEqual(a, b *git.Status) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Branch != b.Branch || a.IsDirty != b.IsDirty || a.Ahead != b.Ahead || a.Behind != b.Behind {
		return false
	}
	if len(a.Files) != len(b.Files) || len(a.Untracked) != len(b.Untracked) {
		return false
	}
	for path, statusA := range a.Files {
		statusB, ok := b.Files[path]
		if !ok || statusA != statusB {
			return false
		}
	}
	return true
}

// setFocus changes focus to the specified panel.
func (m Model) setFocus(target PanelID) Model {
	// Blur previously focused component
	switch m.focus {
	case PanelFileTree:
		m.fileTree = m.fileTree.Blur()
	case PanelGitPanel:
		m.gitPanel = m.gitPanel.Blur()
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
	case PanelGitPanel:
		m.gitPanel = m.gitPanel.Focus()
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
	// Two-column help layout - use fixed-width strings
	// Title row spans full width, then column divider starts below
	helpLines := []string{
		"╔═════════════════════════════════════════════════════════╗",
		"║                   VIBE COMMANDER HELP                   ║",
		"╠════════════════════════════╤════════════════════════════╣",
		"║ NAVIGATION                 │ GIT                        ║",
		"║   Up/k Down/j  Move        │   Space   Stage/Unstage    ║",
		"║   Left/h Right/l Collapse  │   c       Commit (panel)   ║",
		"║   Enter       Select/Open  │                            ║",
		"║   PgUp/PgDn   Page scroll  │ VIEWER                     ║",
		"║   Home/g End/G Top/Bottom  │   /       Search (regex)   ║",
		"║                            │   n/p     Next/Prev match  ║",
		"║ PANELS                     │   Esc     Cancel search    ║",
		"║   Alt+1   Focus file tree  │                            ║",
		"║   Alt+2   Focus content    │ ACTIONS                    ║",
		"║   Alt+3   Toggle terminal  │   Alt+A   Launch AI        ║",
		"║   Alt+G   Toggle git panel │   Alt+S   Select AI        ║",
		"║   Alt+[/] Resize panels    │   Alt+T   Cycle theme      ║",
		"║                            │   Ctrl+H  Toggle help      ║",
		"║ FILE TREE                  │   Ctrl+Q  Quit             ║",
		"║   /       Search files     │                            ║",
		"║   Esc     Clear filter     │                            ║",
		"║   Alt+I   Compact indent   │   Press any key to close   ║",
		"╚════════════════════════════╧════════════════════════════╝",
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
		"║     [Y]es    [N]o    [^Q]uit       ║",
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

// checkCommandAvailable checks if a command is available in PATH.
func checkCommandAvailable(cmd string) bool {
	if cmd == "" {
		return false
	}
	// Handle command with args - only check the first word
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	_, err := exec.LookPath(parts[0])
	return err == nil
}

// aiDialogOptions returns the list of AI assistant options.
type aiDialogOption struct {
	name      string
	command   string
	available bool
}

func (m Model) getAIDialogOptions() []aiDialogOption {
	return []aiDialogOption{
		{name: "Claude", command: "claude", available: checkCommandAvailable("claude")},
		{name: "Gemini", command: "gemini", available: checkCommandAvailable("gemini")},
		{name: "Codex", command: "codex", available: checkCommandAvailable("codex")},
		{name: "Other", command: m.aiDialogCustom, available: m.aiDialogCustom == "" || checkCommandAvailable(m.aiDialogCustom)},
	}
}

// renderAIDialog renders the AI assistant selection dialog.
func (m Model) renderAIDialog(_ string) string {
	options := m.getAIDialogOptions()

	// Box inner width is 50 characters (between the two ║ borders)
	const boxInnerWidth = 50

	// Helper to pad a string to exact width
	padRight := func(s string, width int) string {
		// Count runes for proper Unicode handling
		runeCount := len([]rune(s))
		if runeCount >= width {
			return s
		}
		return s + strings.Repeat(" ", width-runeCount)
	}

	// Build dialog lines
	dialogLines := []string{
		"╔══════════════════════════════════════════════════╗",
		"║           SELECT AI ASSISTANT                    ║",
		"╠══════════════════════════════════════════════════╣",
		"║                                                  ║",
	}

	for i, opt := range options {
		var selector string
		if i == m.aiDialogIndex {
			selector = ">"
		} else {
			selector = " "
		}

		var content string
		if opt.name == "Other" {
			// For "Other" option, show the input field
			var status string
			if m.aiDialogEditing {
				// Show cursor when editing
				customDisplay := m.aiDialogCustom + "_"
				if len(customDisplay) > 20 {
					customDisplay = customDisplay[len(customDisplay)-20:]
				}
				status = customDisplay
			} else if m.aiDialogCustom != "" {
				customDisplay := m.aiDialogCustom
				if len(customDisplay) > 20 {
					customDisplay = customDisplay[:20]
				}
				if opt.available {
					status = customDisplay + " [ok]"
				} else {
					status = customDisplay + " [not found]"
				}
			} else {
				status = "(press Enter to set)"
			}
			content = "    [ " + selector + " ] " + opt.name + ": " + status
		} else {
			var status string
			if opt.available {
				status = "[ok]"
			} else {
				status = "[not found]"
			}
			nameWithSelector := "    [ " + selector + " ] " + opt.name
			// Pad name to align status column
			nameWithSelector = padRight(nameWithSelector, 22)
			content = nameWithSelector + status
		}

		line := "║" + padRight(content, boxInnerWidth) + "║"
		dialogLines = append(dialogLines, line)
	}

	dialogLines = append(dialogLines,
		"║                                                  ║",
		"║     [Enter] Select    [Esc] Cancel               ║",
		"╚══════════════════════════════════════════════════╝",
	)

	dialogContent := lipgloss.JoinVertical(lipgloss.Left, dialogLines...)

	dialogStyle := lipgloss.NewStyle().
		Foreground(theme.CyberCyan).
		Bold(true).
		Padding(1, 2)

	dialogBox := dialogStyle.Render(dialogContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialogBox,
	)
}

// handleAIDialog handles keyboard input for the AI selection dialog.
func (m Model) handleAIDialog(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	options := m.getAIDialogOptions()

	// If editing custom command
	if m.aiDialogEditing {
		switch msg.String() {
		case "esc":
			// Cancel editing
			m.aiDialogEditing = false
			return m, nil
		case "enter":
			// Finish editing
			m.aiDialogEditing = false
			// If custom command is valid, select it
			if m.aiDialogCustom != "" && checkCommandAvailable(m.aiDialogCustom) {
				m.showAIDialog = false
				parts := strings.Fields(m.aiDialogCustom)
				m.aiCommand = parts[0]
				if len(parts) > 1 {
					m.aiArgs = parts[1:]
				} else {
					m.aiArgs = nil
				}
				m.saveState()
				// Launch the AI
				m.aiLaunched = true
				m = m.setFocus(PanelContent)
				return m, func() tea.Msg {
					return content.LaunchAIMsg{
						Command: m.aiCommand,
						Args:    m.aiArgs,
					}
				}
			}
			return m, nil
		case "backspace":
			if len(m.aiDialogCustom) > 0 {
				m.aiDialogCustom = m.aiDialogCustom[:len(m.aiDialogCustom)-1]
			}
			return m, nil
		default:
			// Add character to custom command (printable characters only)
			if len(msg.String()) == 1 {
				r := rune(msg.String()[0])
				if r >= 32 && r < 127 {
					m.aiDialogCustom += string(r)
				}
			}
			return m, nil
		}
	}

	// Normal dialog navigation
	switch msg.String() {
	case "esc":
		m.showAIDialog = false
		return m, nil
	case "up", "k":
		if m.aiDialogIndex > 0 {
			m.aiDialogIndex--
		}
		return m, nil
	case "down", "j":
		if m.aiDialogIndex < len(options)-1 {
			m.aiDialogIndex++
		}
		return m, nil
	case "enter":
		opt := options[m.aiDialogIndex]
		// If "Other" is selected, start editing
		if opt.name == "Other" {
			if m.aiDialogCustom == "" {
				m.aiDialogEditing = true
				return m, nil
			}
			// If custom command is set but not available, allow editing
			if !opt.available {
				m.aiDialogEditing = true
				return m, nil
			}
		}
		// Check if the selected option is available
		if !opt.available {
			// Can't select unavailable option
			return m, nil
		}
		// Select the option
		m.showAIDialog = false
		if opt.name == "Other" {
			parts := strings.Fields(m.aiDialogCustom)
			m.aiCommand = parts[0]
			if len(parts) > 1 {
				m.aiArgs = parts[1:]
			} else {
				m.aiArgs = nil
			}
		} else {
			m.aiCommand = opt.command
			m.aiArgs = nil
		}
		m.saveState()
		// Launch the AI
		m.aiLaunched = true
		m = m.setFocus(PanelContent)
		return m, func() tea.Msg {
			return content.LaunchAIMsg{
				Command: m.aiCommand,
				Args:    m.aiArgs,
			}
		}
	}
	return m, nil
}

// saveState persists the current application state globally.
func (m Model) saveState() {
	s := state.State{
		AIWindowOpen:     m.aiLaunched,
		ThemeIndex:       theme.CurrentThemeIndex(),
		LeftPanelPercent: m.leftPanelPercent,
		CompactIndent:    m.fileTree.CompactIndent(),
		AICommand:        m.aiCommand,
		AIArgs:           m.aiArgs,
	}
	// Ignore errors - state persistence is best-effort
	_ = state.Save(s)
}

// panelAtPosition returns which panel contains the given screen coordinates.
func (m Model) panelAtPosition(x, y int) PanelID {
	// Handle fullscreen content mode
	if m.fullscreen == PanelContent {
		return PanelContent
	}

	// Check mini buffer first (if visible)
	if m.miniVisible {
		_, miniY, _, miniH := m.layout.MiniBufferBounds()
		if y >= miniY && y < miniY+miniH {
			return PanelMiniBuffer
		}
	}

	// Check left panel area (file tree and git panel)
	_, _, leftW, fileTreeH := m.layout.LeftPanelBounds()
	if x < leftW {
		// Check if git panel is visible and click is in git panel area
		if m.gitPanelVisible {
			_, gitY, _, gitH := m.layout.GitPanelBounds()
			if y >= gitY && y < gitY+gitH {
				return PanelGitPanel
			}
		}
		// Otherwise it's the file tree (if within its bounds)
		if y < fileTreeH {
			return PanelFileTree
		}
	}

	// Check content panel
	if y < m.layout.MainHeight {
		if x >= leftW {
			return PanelContent
		}
	}

	return PanelNone // Status bar or outside panels
}

// detectHeaderClick checks if a mouse click is on a title in the content panel header.
// Returns the ContentSource that was clicked, or SourceNone if not on a title.
func (m Model) detectHeaderClick(x, y int) content.ContentSource {
	// Only the top border (y == 0 relative to panel) contains clickable titles
	// Content panel starts at x = LeftWidth
	panelX := x - m.layout.LeftWidth
	if panelX < 0 {
		return content.SourceNone
	}

	// Only check first row (the header)
	if y != 0 {
		return content.SourceNone
	}

	// Build the same opts we use for rendering to get accurate regions
	sources := m.content.SourcesInfo()
	opts := m.buildContentPanelOpts(sources, "")

	// Calculate title regions
	primary, secondary := theme.CalculateTitleRegions(opts)

	// Check if click is in primary title region (file)
	if panelX >= primary.StartX && panelX < primary.EndX {
		return content.SourceFile
	}

	// Check if click is in secondary title region (AI)
	if opts.SecondaryTitle != "" && panelX >= secondary.StartX && panelX < secondary.EndX {
		return content.SourceAI
	}

	return content.SourceNone
}

// routeMouseClickToPanel routes mouse click events to the panel at the mouse position.
func (m *Model) routeMouseClickToPanel(msg tea.MouseClickMsg) tea.Cmd {
	var cmd tea.Cmd
	mouse := msg.Mouse()

	targetPanel := m.panelAtPosition(mouse.X, mouse.Y)

	switch targetPanel {
	case PanelFileTree:
		m.fileTree, cmd = m.fileTree.Update(msg)
	case PanelGitPanel:
		if m.gitPanelVisible {
			// Adjust Y coordinate relative to git panel
			_, gitY, _, _ := m.layout.GitPanelBounds()
			// Create adjusted mouse event with Y relative to git panel
			adjustedMouse := mouse
			adjustedMouse.Y = mouse.Y - gitY
			m.gitPanel, cmd = m.gitPanel.Update(tea.MouseClickMsg(adjustedMouse))
		}
	case PanelContent:
		// Adjust X coordinate relative to content panel (starts at LeftWidth)
		contentX, _, _, _ := m.layout.RightPanelBounds()
		adjustedMouse := mouse
		adjustedMouse.X = mouse.X - contentX
		m.content, cmd = m.content.Update(tea.MouseClickMsg(adjustedMouse))
	case PanelMiniBuffer:
		if m.miniVisible {
			m.miniBuffer, cmd = m.miniBuffer.Update(msg)
		}
	}

	return cmd
}

// routeMouseToPanel routes mouse events to the panel at the mouse position.
// This allows scrolling in non-focused panels.
func (m *Model) routeMouseToPanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	// Get mouse coordinates from the message
	var mouseX, mouseY int
	switch mm := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := mm.Mouse()
		mouseX, mouseY = mouse.X, mouse.Y
	case tea.MouseMotionMsg:
		mouse := mm.Mouse()
		mouseX, mouseY = mouse.X, mouse.Y
	case tea.MouseReleaseMsg:
		mouse := mm.Mouse()
		mouseX, mouseY = mouse.X, mouse.Y
	default:
		return nil
	}

	targetPanel := m.panelAtPosition(mouseX, mouseY)

	switch targetPanel {
	case PanelFileTree:
		m.fileTree, cmd = m.fileTree.Update(msg)
	case PanelContent:
		// Adjust X coordinate relative to content panel (starts at LeftWidth)
		contentX, _, _, _ := m.layout.RightPanelBounds()
		switch mm := msg.(type) {
		case tea.MouseMotionMsg:
			mouse := mm.Mouse()
			adjustedMouse := mouse
			adjustedMouse.X = mouse.X - contentX
			m.content, cmd = m.content.Update(tea.MouseMotionMsg(adjustedMouse))
		case tea.MouseReleaseMsg:
			mouse := mm.Mouse()
			adjustedMouse := mouse
			adjustedMouse.X = mouse.X - contentX
			m.content, cmd = m.content.Update(tea.MouseReleaseMsg(adjustedMouse))
		default:
			m.content, cmd = m.content.Update(msg)
		}
	case PanelMiniBuffer:
		if m.miniVisible {
			m.miniBuffer, cmd = m.miniBuffer.Update(msg)
		}
	}

	return cmd
}

// renderCommitDialog renders the commit message dialog.
func (m Model) renderCommitDialog(_ string) string {
	// Show staged files count
	stagedCount := m.gitPanel.StagedCount()

	// Get the raw commit message value and add cursor
	inputValue := m.commitInput.Value()
	cursorPos := m.commitInput.Position()

	// Build display string with cursor
	var displayMsg string
	if cursorPos >= len(inputValue) {
		displayMsg = inputValue + "█"
	} else {
		displayMsg = inputValue[:cursorPos] + "█" + inputValue[cursorPos:]
	}

	// Truncate if too long (keep last 52 chars to show cursor)
	if len(displayMsg) > 52 {
		displayMsg = displayMsg[len(displayMsg)-52:]
	}

	// Pad to fixed width
	for len(displayMsg) < 52 {
		displayMsg += " "
	}

	// Build the dialog content
	var content strings.Builder
	content.WriteString("                       COMMIT CHANGES                         \n")
	content.WriteString("──────────────────────────────────────────────────────────────\n")
	content.WriteString("                                                              \n")
	content.WriteString("  Staged files: " + itoa(stagedCount) + strings.Repeat(" ", 45-len(itoa(stagedCount))) + "\n")
	content.WriteString("                                                              \n")
	content.WriteString("  Commit message:                                             \n")
	content.WriteString("  ┌────────────────────────────────────────────────────┐      \n")
	content.WriteString("  │ " + displayMsg + " │      \n")
	content.WriteString("  └────────────────────────────────────────────────────┘      \n")
	content.WriteString("                                                              \n")
	content.WriteString("      [Enter] Commit    [Esc] Cancel                          ")

	dialogStyle := lipgloss.NewStyle().
		Foreground(theme.CyberCyan).
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.CyberCyan).
		Padding(1, 2)

	dialogBox := dialogStyle.Render(content.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialogBox,
	)
}

// handleCommitDialog handles keyboard input for the commit dialog.
func (m Model) handleCommitDialog(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel commit
		m.showCommitDialog = false
		m.commitInput.Blur()
		return m, nil

	case "enter":
		// Execute commit if message is not empty
		commitMsg := strings.TrimSpace(m.commitInput.Value())
		if commitMsg == "" {
			return m, nil
		}
		m.showCommitDialog = false
		m.commitInput.Blur()
		// Use tea.ExecProcess to suspend TUI and allow GPG passphrase input
		cmd := exec.Command("git", "commit", "-m", commitMsg)
		cmd.Dir = m.workDir
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
			return gitCommitFinishedMsg{err: err}
		})
	}

	// Pass all other keys to textinput
	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return m, cmd
}
