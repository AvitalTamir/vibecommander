package minibuffer

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandResultMsg is sent when a command finishes executing.
type CommandResultMsg struct {
	Output string
	Err    error
}

// Model is the mini buffer component for running shell commands.
type Model struct {
	components.Base

	input    textinput.Model
	output   viewport.Model
	history  []string
	histIdx  int
	lastCmd  string
	running  bool
	cwd      string

	theme *theme.Theme
	ready bool
}

// New creates a new mini buffer model.
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Prompt = "❯ "
	ti.CharLimit = 256
	ti.Width = 80

	return Model{
		input:   ti,
		history: make([]string, 0),
		histIdx: -1,
		theme:   theme.DefaultTheme(),
	}
}

// Init initializes the mini buffer.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Focused() {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			if m.running {
				return m, nil
			}
			cmd := strings.TrimSpace(m.input.Value())
			if cmd != "" {
				m.lastCmd = cmd
				m.history = append(m.history, cmd)
				m.histIdx = len(m.history)
				m.input.SetValue("")
				m.running = true
				return m, m.executeCommand(cmd)
			}

		case tea.KeyUp:
			// Navigate history up
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			}
			return m, nil

		case tea.KeyDown:
			// Navigate history down
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			} else if m.histIdx == len(m.history)-1 {
				m.histIdx = len(m.history)
				m.input.SetValue("")
			}
			return m, nil

		case tea.KeyCtrlC:
			// Clear input
			m.input.SetValue("")
			return m, nil

		case tea.KeyCtrlL:
			// Clear output
			m.output.SetContent("")
			return m, nil
		}

		// Pass to text input
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

	case CommandResultMsg:
		m.running = false
		var content string
		if msg.Err != nil {
			content = lipgloss.NewStyle().
				Foreground(theme.NeonRed).
				Render("Error: " + msg.Err.Error())
			if msg.Output != "" {
				content += "\n" + msg.Output
			}
		} else {
			content = msg.Output
		}

		// Append to existing output
		existing := m.output.View()
		if existing != "" {
			content = existing + "\n" + m.formatCommand(m.lastCmd) + "\n" + content
		} else {
			content = m.formatCommand(m.lastCmd) + "\n" + content
		}
		m.output.SetContent(content)
		m.output.GotoBottom()

	case tea.MouseMsg:
		// Handle mouse for output scrolling
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) formatCommand(cmd string) string {
	return lipgloss.NewStyle().
		Foreground(theme.MatrixGreen).
		Bold(true).
		Render("❯ " + cmd)
}

func (m Model) executeCommand(cmd string) tea.Cmd {
	return func() tea.Msg {
		// Parse command
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			return CommandResultMsg{}
		}

		name := parts[0]
		args := parts[1:]

		// Handle built-in commands
		if name == "clear" {
			return CommandResultMsg{Output: "\033[H\033[2J"}
		}

		// Execute external command
		c := exec.Command(name, args...)
		var stdout, stderr bytes.Buffer
		c.Stdout = &stdout
		c.Stderr = &stderr

		err := c.Run()

		output := stdout.String()
		if stderr.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}

		// Trim trailing newline
		output = strings.TrimSuffix(output, "\n")

		return CommandResultMsg{
			Output: output,
			Err:    err,
		}
	}
}

// View renders the mini buffer.
func (m Model) View() string {
	w, h := m.Size()
	if w == 0 || h == 0 {
		return ""
	}

	// Input line at the bottom
	inputStyle := lipgloss.NewStyle().
		Foreground(theme.CyberCyan).
		Width(w)

	// Output area above input
	outputHeight := h - 2 // 1 for input, 1 for separator
	if outputHeight < 0 {
		outputHeight = 0
	}

	outputStyle := lipgloss.NewStyle().
		Width(w).
		Height(outputHeight).
		Foreground(theme.PureWhite)

	// Separator
	separator := lipgloss.NewStyle().
		Foreground(theme.DimPurple).
		Render(strings.Repeat("─", w))

	// Status indicator
	var status string
	if m.running {
		status = lipgloss.NewStyle().
			Foreground(theme.ElectricYellow).
			Render(" ⟳ running...")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		outputStyle.Render(m.output.View()),
		separator,
		inputStyle.Render(m.input.View()+status),
	)
}

// Focus gives focus to this component.
func (m Model) Focus() Model {
	m.Base.Focus()
	m.input.Focus()
	return m
}

// Blur removes focus from this component.
func (m Model) Blur() Model {
	m.Base.Blur()
	m.input.Blur()
	return m
}

// SetSize updates the component's dimensions.
func (m Model) SetSize(width, height int) Model {
	m.Base.SetSize(width, height)

	m.input.Width = width - 4 // Account for prompt

	// Initialize or resize viewport
	outputHeight := height - 2
	if outputHeight < 0 {
		outputHeight = 0
	}

	if !m.ready {
		m.output = viewport.New(width, outputHeight)
		m.output.MouseWheelEnabled = true
		m.ready = true
	} else {
		m.output.Width = width
		m.output.Height = outputHeight
	}

	return m
}

// Clear clears the output.
func (m *Model) Clear() {
	m.output.SetContent("")
}

// SetCwd sets the current working directory for commands.
func (m *Model) SetCwd(cwd string) {
	m.cwd = cwd
}
