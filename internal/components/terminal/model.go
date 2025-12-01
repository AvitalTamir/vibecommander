package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// Messages
type (
	// OutputMsg contains output from the terminal.
	OutputMsg struct {
		Data []byte
	}

	// ExitMsg is sent when the terminal process exits.
	ExitMsg struct {
		Err error
	}

	// StartMsg requests starting a command.
	StartMsg struct {
		Cmd  string
		Args []string
	}
)

// Model is the terminal component for running interactive programs.
type Model struct {
	components.Base

	vt      vt10x.Terminal
	cmd     *exec.Cmd
	pty     *os.File
	mu      sync.Mutex
	running bool
	exitErr error

	theme *theme.Theme
	ready bool
}

// New creates a new terminal model.
func New() Model {
	return Model{
		theme: theme.DefaultTheme(),
	}
}

// Init initializes the terminal.
func (m Model) Init() tea.Cmd {
	return nil
}

// Start starts a command in the terminal.
func Start(cmd string, args ...string) tea.Cmd {
	return func() tea.Msg {
		return StartMsg{Cmd: cmd, Args: args}
	}
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case StartMsg:
		if m.running {
			return m, nil
		}
		return m.startProcess(msg.Cmd, msg.Args)

	case OutputMsg:
		m.mu.Lock()
		if m.vt != nil {
			m.vt.Write(msg.Data)
		}
		m.mu.Unlock()
		return m, nil

	case ExitMsg:
		m.mu.Lock()
		m.running = false
		m.exitErr = msg.Err
		if m.pty != nil {
			m.pty.Close()
			m.pty = nil
		}
		m.cmd = nil
		m.mu.Unlock()
		return m, nil

	case tea.KeyMsg:
		if !m.Focused() {
			return m, nil
		}

		// Send input to PTY if running
		if m.running && m.pty != nil {
			var input []byte

			switch msg.Type {
			case tea.KeyEnter:
				input = []byte("\r")
			case tea.KeyBackspace:
				input = []byte{127}
			case tea.KeyTab:
				input = []byte("\t")
			case tea.KeySpace:
				input = []byte(" ")
			// Ctrl keys (Ctrl+A=1 through Ctrl+Z=26)
			case tea.KeyCtrlA:
				input = []byte{1}
			case tea.KeyCtrlB:
				input = []byte{2}
			case tea.KeyCtrlC:
				input = []byte{3}
			case tea.KeyCtrlD:
				input = []byte{4}
			case tea.KeyCtrlE:
				input = []byte{5}
			case tea.KeyCtrlF:
				input = []byte{6}
			case tea.KeyCtrlG:
				input = []byte{7}
			case tea.KeyCtrlJ:
				input = []byte{10}
			case tea.KeyCtrlK:
				input = []byte{11}
			case tea.KeyCtrlL:
				input = []byte{12}
			case tea.KeyCtrlN:
				input = []byte{14}
			case tea.KeyCtrlO:
				input = []byte{15}
			case tea.KeyCtrlP:
				input = []byte{16}
			case tea.KeyCtrlR:
				input = []byte{18}
			case tea.KeyCtrlS:
				input = []byte{19}
			case tea.KeyCtrlT:
				input = []byte{20}
			case tea.KeyCtrlU:
				input = []byte{21}
			case tea.KeyCtrlV:
				input = []byte{22}
			case tea.KeyCtrlW:
				input = []byte{23}
			case tea.KeyCtrlX:
				input = []byte{24}
			case tea.KeyCtrlY:
				input = []byte{25}
			case tea.KeyCtrlZ:
				input = []byte{26}
			case tea.KeyUp:
				input = []byte("\x1b[A")
			case tea.KeyDown:
				input = []byte("\x1b[B")
			case tea.KeyRight:
				input = []byte("\x1b[C")
			case tea.KeyLeft:
				input = []byte("\x1b[D")
			case tea.KeyEscape:
				input = []byte{27}
			case tea.KeyRunes:
				// Check for Alt modifier (sends ESC + char)
				if msg.Alt {
					for _, r := range msg.Runes {
						input = append(input, 27) // ESC
						input = append(input, byte(r))
					}
				} else {
					input = []byte(string(msg.Runes))
				}
			case tea.KeyHome:
				input = []byte("\x1b[H")
			case tea.KeyEnd:
				input = []byte("\x1b[F")
			case tea.KeyPgUp:
				input = []byte("\x1b[5~")
			case tea.KeyPgDown:
				input = []byte("\x1b[6~")
			case tea.KeyDelete:
				input = []byte("\x1b[3~")
			}

			if len(input) > 0 {
				m.pty.Write(input)
			}
			return m, nil
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) startProcess(cmd string, args []string) (Model, tea.Cmd) {
	w, h := m.Size()
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	// Create virtual terminal with current size
	m.vt = vt10x.New(vt10x.WithSize(w, h-1)) // -1 for status line

	m.cmd = exec.Command(cmd, args...)
	m.cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start PTY
	ptmx, err := pty.Start(m.cmd)
	if err != nil {
		m.vt.Write([]byte("\x1b[31mError starting process: " + err.Error() + "\x1b[0m\n"))
		return m, nil
	}

	m.pty = ptmx
	m.running = true
	m.exitErr = nil

	// Set PTY size
	pty.Setsize(m.pty, &pty.Winsize{
		Rows: uint16(h - 1),
		Cols: uint16(w),
	})

	// Start reading output
	return m, m.readOutput()
}

func (m Model) readOutput() tea.Cmd {
	return func() tea.Msg {
		if m.pty == nil {
			return nil
		}

		buf := make([]byte, 4096)
		n, err := m.pty.Read(buf)
		if err != nil {
			if err == io.EOF {
				// Wait for process to exit
				exitErr := m.cmd.Wait()
				return ExitMsg{Err: exitErr}
			}
			return ExitMsg{Err: err}
		}

		return OutputMsg{Data: buf[:n]}
	}
}

// View renders the terminal.
func (m Model) View() string {
	w, h := m.Size()
	if !m.ready || w <= 0 || h <= 0 {
		return lipgloss.NewStyle().
			Foreground(theme.MutedLavender).
			Render("Initializing terminal...")
	}

	// Render the virtual terminal screen
	var content string
	if m.vt != nil {
		content = m.renderVT()
	} else {
		content = lipgloss.NewStyle().
			Foreground(theme.MutedLavender).
			Italic(true).
			Render("Terminal ready...")
	}

	// Status line
	var status string
	if m.running {
		status = lipgloss.NewStyle().
			Foreground(theme.MatrixGreen).
			Render(" ● Running")
	} else if m.exitErr != nil {
		status = lipgloss.NewStyle().
			Foreground(theme.NeonRed).
			Render(" ● Exited with error")
	} else if m.vt != nil {
		status = lipgloss.NewStyle().
			Foreground(theme.CyberCyan).
			Render(" ○ Process ended")
	} else {
		status = lipgloss.NewStyle().
			Foreground(theme.MutedLavender).
			Render(" ○ Idle")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		content,
		status,
	)
}

// renderVT renders the virtual terminal screen buffer with colors
func (m Model) renderVT() string {
	if m.vt == nil {
		return ""
	}

	m.vt.Lock()
	defer m.vt.Unlock()

	cols, rows := m.vt.Size()
	if cols <= 0 || rows <= 0 {
		return ""
	}

	var lines []string
	for row := 0; row < rows; row++ {
		var line strings.Builder
		for col := 0; col < cols; col++ {
			glyph := m.vt.Cell(col, row)
			ch := glyph.Char
			if ch == 0 {
				ch = ' '
			}

			// Build style with foreground and background colors
			style := lipgloss.NewStyle()

			// Apply foreground color
			if fg := colorToLipgloss(glyph.FG); fg != "" {
				style = style.Foreground(lipgloss.Color(fg))
			}

			// Apply background color
			if bg := colorToLipgloss(glyph.BG); bg != "" {
				style = style.Background(lipgloss.Color(bg))
			}

			// Render the character with style
			line.WriteString(style.Render(string(ch)))
		}
		lines = append(lines, line.String())
	}

	return strings.Join(lines, "\n")
}

// colorToLipgloss converts a vt10x color to a lipgloss color string
func colorToLipgloss(c vt10x.Color) string {
	// Check for default colors (>= 0x01000000)
	if c >= 0x01000000 {
		return "" // Use default terminal colors
	}

	// ANSI 256-color palette (0-255)
	if c < 256 {
		return fmt.Sprintf("%d", c)
	}

	// True color RGB (encoded as 0x00RRGGBB)
	r := (c >> 16) & 0xFF
	g := (c >> 8) & 0xFF
	b := c & 0xFF
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
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

	// Account for status line
	termHeight := height - 1
	if termHeight < 0 {
		termHeight = 0
	}

	if !m.ready {
		m.ready = true
	}

	// Resize virtual terminal if it exists
	if m.vt != nil && width > 0 && termHeight > 0 {
		m.vt.Resize(width, termHeight)
	}

	// Update PTY size if running
	if m.running && m.pty != nil && width > 0 && termHeight > 0 {
		pty.Setsize(m.pty, &pty.Winsize{
			Rows: uint16(termHeight),
			Cols: uint16(width),
		})
	}

	return m
}

// Running returns whether a process is running.
func (m Model) Running() bool {
	return m.running
}

// Stop stops the running process.
func (m *Model) Stop() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
	if m.pty != nil {
		m.pty.Close()
		m.pty = nil
	}
	m.running = false
}

// ContinueReading returns a command to continue reading output.
func (m Model) ContinueReading() tea.Cmd {
	if !m.running || m.pty == nil {
		return nil
	}
	return m.readOutput()
}
