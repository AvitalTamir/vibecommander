package app

import (
	"context"
	"os"
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
)

// GitStatusMsg carries updated git status
type GitStatusMsg struct {
	Status *git.Status
	IsRepo bool
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
	showHelp    bool
	showQuit    bool // Quit confirmation dialog

	// Layout
	layout layout.Layout
	theme  *theme.Theme
	keys   KeyMap

	// Git
	gitProvider *git.ShellProvider
	gitStatus   *git.Status
	isGitRepo   bool
	workDir     string

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

	return Model{
		fileTree:    ft,
		content:     content.New(),
		miniBuffer:  minibuffer.New(),
		focus:       PanelFileTree,
		miniVisible: false,
		theme:       theme.DefaultTheme(),
		keys:        DefaultKeyMap(),
		gitProvider: gitProvider,
		workDir:     workDir,
		isGitRepo:   gitProvider.IsRepo(),
	}
}

// Init initializes the application.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fileTree.Init(),
		m.content.Init(),
		m.miniBuffer.Init(),
		m.refreshGitStatus(),
	)
}

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

// tickGitRefresh returns a command that triggers git refresh periodically
func tickGitRefresh() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return gitTickMsg{}
	})
}

type gitTickMsg struct{}

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
		return m, tickGitRefresh()

	case GitStatusMsg:
		m.isGitRepo = msg.IsRepo
		m.gitStatus = msg.Status
		// Update file tree with git status
		if msg.Status != nil {
			m.fileTree = m.fileTree.SetGitStatus(msg.Status)
		}
		return m, tickGitRefresh()

	case gitTickMsg:
		return m, m.refreshGitStatus()

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

		case key.Matches(msg, m.keys.FocusNext):
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			m = m.cycleFocus(1)
			return m, nil

		case key.Matches(msg, m.keys.FocusPrev):
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			m = m.cycleFocus(-1)
			return m, nil

		case key.Matches(msg, m.keys.ToggleMini):
			m.miniVisible = !m.miniVisible
			m.layout = layout.Calculate(m.width, m.height, m.miniVisible)
			m = m.updateSizes()
			if m.miniVisible {
				// Focus the mini buffer when opening
				m = m.setFocus(PanelMiniBuffer)
			} else {
				// Return focus to previous panel when closing
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

	case minibuffer.CommandResultMsg:
		// Route to mini buffer
		var cmd tea.Cmd
		m.miniBuffer, cmd = m.miniBuffer.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case content.LaunchAIMsg:
		// Route to content pane
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case terminal.OutputMsg, terminal.ExitMsg:
		// Route to content pane (for AI terminal)
		var cmd tea.Cmd
		m.content, cmd = m.content.Update(msg)
		cmds = append(cmds, cmd)
		// Continue reading output if terminal is still running
		if m.content.Mode() == content.ModeAI {
			cmds = append(cmds, func() tea.Msg {
				return terminal.OutputMsg{} // Trigger continue reading
			})
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

	view := lipgloss.JoinVertical(lipgloss.Left, mainArea, statusBar)

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

	// Version
	version := lipgloss.NewStyle().
		Foreground(theme.LaserPurple).
		Render("Vibe Commander v0.1.0")

	// Layout the status bar
	left := branch + panelInfo + help
	right := version

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

// cycleFocus cycles focus between panels.
func (m Model) cycleFocus(direction int) Model {
	panels := []PanelID{PanelFileTree, PanelContent}
	if m.miniVisible {
		panels = append(panels, PanelMiniBuffer)
	}

	currentIdx := 0
	for i, p := range panels {
		if p == m.focus {
			currentIdx = i
			break
		}
	}

	newIdx := (currentIdx + direction + len(panels)) % len(panels)
	return m.setFocus(panels[newIdx])
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
		"║                  VIBE COMMANDER HELP                      ║",
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
		"║    Tab           Next panel                              ║",
		"║    Shift+Tab     Previous panel                          ║",
		"║    ` or Ctrl+T   Toggle terminal                         ║",
		"║                                                          ║",
		"║  ACTIONS                                                 ║",
		"║    Ctrl+A        Launch AI assistant                     ║",
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
