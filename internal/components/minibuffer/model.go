package minibuffer

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
)

// Model is the mini buffer component with a proper PTY shell.
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

// New creates a new mini buffer model.
func New() Model {
	return Model{
		theme: theme.DefaultTheme(),
	}
}

// Init initializes the mini buffer.
func (m Model) Init() tea.Cmd {
	return nil
}

// StartShell starts a shell in the terminal.
func (m Model) StartShell() tea.Cmd {
	return func() tea.Msg {
		return startShellMsg{}
	}
}

type startShellMsg struct{}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case startShellMsg:
		if m.running {
			return m, nil
		}
		return m.startProcess()

	case OutputMsg:
		m.mu.Lock()
		if m.vt != nil {
			m.vt.Write(msg.Data)
		}
		m.mu.Unlock()
		// Continue reading
		if m.running && m.pty != nil {
			cmds = append(cmds, m.readOutput())
		}
		return m, tea.Batch(cmds...)

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
		// Restart shell
		return m, m.StartShell()

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
				if msg.Alt {
					input = []byte{27, 127} // ESC + DEL for Alt+Backspace (delete word)
				} else {
					input = []byte{127}
				}
			case tea.KeyTab:
				input = []byte("\t")
			case tea.KeySpace:
				input = []byte(" ")
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
				if msg.Alt {
					for _, r := range msg.Runes {
						input = append(input, 27)
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

func (m Model) startProcess() (Model, tea.Cmd) {
	w, h := m.Size()
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	// Create virtual terminal with current size
	m.vt = vt10x.New(vt10x.WithSize(w, h))

	// Get user's shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	m.cmd = exec.Command(shell)
	m.cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start PTY
	ptmx, err := pty.Start(m.cmd)
	if err != nil {
		m.vt.Write([]byte("\x1b[31mError starting shell: " + err.Error() + "\x1b[0m\n"))
		return m, nil
	}

	m.pty = ptmx
	m.running = true
	m.exitErr = nil

	// Set PTY size
	pty.Setsize(m.pty, &pty.Winsize{
		Rows: uint16(h),
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
				exitErr := m.cmd.Wait()
				return ExitMsg{Err: exitErr}
			}
			return ExitMsg{Err: err}
		}

		return OutputMsg{Data: buf[:n]}
	}
}

// View renders the mini buffer.
func (m Model) View() string {
	w, h := m.Size()
	if !m.ready || w <= 0 || h <= 0 {
		return lipgloss.NewStyle().
			Foreground(theme.MutedLavender).
			Render("Initializing shell...")
	}

	// Render the virtual terminal screen
	if m.vt != nil {
		return m.renderVT()
	}

	return lipgloss.NewStyle().
		Foreground(theme.MutedLavender).
		Italic(true).
		Render("Shell ready...")
}

// renderVT renders the virtual terminal screen buffer with colors
// Optimized to use raw ANSI codes and batch consecutive characters with same style
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

	// Get cursor position and visibility
	cursor := m.vt.Cursor()
	cursorVisible := m.vt.CursorVisible() && m.Focused()

	var result strings.Builder
	result.Grow(rows * cols * 2) // Pre-allocate

	for row := 0; row < rows; row++ {
		if row > 0 {
			result.WriteByte('\n')
		}

		// Track current style to batch characters
		var currentFG, currentBG vt10x.Color
		var currentMode int16
		var currentIsCursor bool
		var batch strings.Builder
		firstCell := true

		flushBatch := func() {
			if batch.Len() == 0 {
				return
			}
			// Write ANSI codes for the batch
			result.WriteString(buildANSI(currentFG, currentBG, currentMode, currentIsCursor))
			result.WriteString(batch.String())
			result.WriteString("\x1b[0m") // Reset
			batch.Reset()
		}

		for col := 0; col < cols; col++ {
			glyph := m.vt.Cell(col, row)
			ch := glyph.Char
			if ch == 0 {
				ch = ' '
			}

			isCursor := cursorVisible && col == cursor.X && row == cursor.Y

			// Check if style changed
			if !firstCell && (glyph.FG != currentFG || glyph.BG != currentBG || glyph.Mode != currentMode || isCursor != currentIsCursor) {
				flushBatch()
			}

			currentFG = glyph.FG
			currentBG = glyph.BG
			currentMode = glyph.Mode
			currentIsCursor = isCursor
			firstCell = false

			batch.WriteRune(ch)
		}
		flushBatch()
	}

	return result.String()
}

// buildANSI builds ANSI escape sequence for the given style
func buildANSI(fg, bg vt10x.Color, mode int16, isCursor bool) string {
	var codes []string

	if isCursor {
		codes = append(codes, "7") // Reverse video
	} else {
		// Mode attributes
		if mode&0x01 != 0 { // Reverse
			codes = append(codes, "7")
		}
		if mode&0x02 != 0 { // Underline
			codes = append(codes, "4")
		}
		if mode&0x04 != 0 { // Bold
			codes = append(codes, "1")
		}
		if mode&0x10 != 0 { // Italic
			codes = append(codes, "3")
		}

		// Foreground color
		if fgCode := colorToANSI(fg, true); fgCode != "" {
			codes = append(codes, fgCode)
		}

		// Background color
		if bgCode := colorToANSI(bg, false); bgCode != "" {
			codes = append(codes, bgCode)
		}
	}

	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

// colorToANSI converts a vt10x color to ANSI escape code
func colorToANSI(c vt10x.Color, isFG bool) string {
	// Default color
	if c >= 0x01000000 {
		return ""
	}

	base := 38 // Foreground
	if !isFG {
		base = 48 // Background
	}

	// ANSI 256-color palette
	if c < 256 {
		return fmt.Sprintf("%d;5;%d", base, c)
	}

	// True color RGB
	r := (c >> 16) & 0xFF
	g := (c >> 8) & 0xFF
	b := c & 0xFF
	return fmt.Sprintf("%d;2;%d;%d;%d", base, r, g, b)
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

	if !m.ready {
		m.ready = true
	}

	// Resize virtual terminal if it exists
	if m.vt != nil && width > 0 && height > 0 {
		m.vt.Resize(width, height)
	}

	// Update PTY size if running
	if m.running && m.pty != nil && width > 0 && height > 0 {
		pty.Setsize(m.pty, &pty.Winsize{
			Rows: uint16(height),
			Cols: uint16(width),
		})
	}

	return m
}

// Running returns whether the shell is running.
func (m Model) Running() bool {
	return m.running
}

// Stop stops the running shell.
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
