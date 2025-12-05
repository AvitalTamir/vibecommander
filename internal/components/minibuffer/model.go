package minibuffer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/theme"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// Render throttling interval - render at most once per this duration
// Using 50ms (~20fps) to reduce flicker from animated spinners while still feeling responsive
const renderInterval = 50 * time.Millisecond

// renderTickMsg triggers a render update
type renderTickMsg struct{}

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

	// Scrollback buffer
	scrollback    []string // Lines that scrolled off the top
	scrollOffset  int      // 0 = live view, >0 = scrolled up N lines
	maxScrollback int      // Max lines to keep

	// Render throttling
	cachedView      string    // Cached rendered view
	lastRender      time.Time // Last time we rendered
	renderScheduled bool      // Whether a render tick is scheduled
	dirty           bool      // Whether the terminal has new output

	theme *theme.Theme
	ready bool
}

// New creates a new mini buffer model.
func New() Model {
	return Model{
		theme:         theme.DefaultTheme(),
		maxScrollback: 10000,
	}
}

// addToScrollback adds a line to scrollback with deduplication
func (m *Model) addToScrollback(line string) {
	// Check against recent entries to avoid duplicates
	checkCount := 20 // Check last 20 entries
	if checkCount > len(m.scrollback) {
		checkCount = len(m.scrollback)
	}
	for i := len(m.scrollback) - checkCount; i < len(m.scrollback); i++ {
		if m.scrollback[i] == line {
			return // Already exists, skip
		}
	}
	m.scrollback = append(m.scrollback, line)
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

// scheduleRenderTick returns a command to schedule a render tick if not already scheduled
func (m *Model) scheduleRenderTick() tea.Cmd {
	if m.renderScheduled {
		return nil
	}
	m.renderScheduled = true
	// Calculate time until next render (ensure we don't render too fast)
	timeSinceLastRender := time.Since(m.lastRender)
	delay := renderInterval - timeSinceLastRender
	if delay < 0 {
		delay = 0
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return renderTickMsg{}
	})
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case renderTickMsg:
		// Render tick - update cached view if dirty
		m.renderScheduled = false
		if m.dirty {
			m.dirty = false
			m.lastRender = time.Now()
			newView := m.renderVT()
			// Only update if content actually changed visually
			if newView != m.cachedView {
				m.cachedView = newView
			}
		}
		// If still running, schedule another tick to catch any missed updates
		if m.running && !m.renderScheduled {
			return m, m.scheduleRenderTick()
		}
		return m, nil

	case startShellMsg:
		if m.running {
			return m, nil
		}
		return m.startProcess()

	case OutputMsg:
		m.mu.Lock()
		// Snap to live view on new output
		m.scrollOffset = 0

		if m.vt != nil {
			cols, rows := m.vt.Size()

			// Capture all screen lines before write
			oldPlainLines := make([]string, rows)
			oldRenderedLines := make([]string, rows)
			for row := 0; row < rows; row++ {
				oldPlainLines[row] = m.getScreenLinePlain(cols, row)
				oldRenderedLines[row] = m.renderScreenLine(cols, row)
			}

			// Write data
			m.vt.Write(msg.Data)

			// Find where the new top line was in the old screen
			newTopLine := m.getScreenLinePlain(cols, 0)
			scrollAmount := 0

			if len(strings.TrimSpace(newTopLine)) > 0 {
				for i := 1; i < rows; i++ {
					if len(strings.TrimSpace(oldPlainLines[i])) > 0 && oldPlainLines[i] == newTopLine {
						scrollAmount = i
						break
					}
				}
			}

			// Add scrolled lines to scrollback
			if scrollAmount > 0 {
				// Normal scroll - add lines that scrolled off
				for i := 0; i < scrollAmount; i++ {
					if len(strings.TrimSpace(oldPlainLines[i])) > 0 {
						m.addToScrollback(oldRenderedLines[i])
					}
				}
			} else if oldPlainLines[0] != newTopLine && len(strings.TrimSpace(oldPlainLines[0])) > 0 {
				// Screen changed but couldn't detect scroll amount
				// This happens with large data chunks - save all non-empty old lines
				for i := 0; i < rows; i++ {
					if len(strings.TrimSpace(oldPlainLines[i])) > 0 {
						m.addToScrollback(oldRenderedLines[i])
					}
				}
			}

			// Trim scrollback if too large
			if len(m.scrollback) > m.maxScrollback {
				m.scrollback = m.scrollback[len(m.scrollback)-m.maxScrollback:]
			}
		}
		m.dirty = true
		m.mu.Unlock()

		// Schedule a render tick and continue reading
		if m.running && m.pty != nil {
			cmds = append(cmds, m.readOutput())
		}
		if cmd := m.scheduleRenderTick(); cmd != nil {
			cmds = append(cmds, cmd)
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

	case tea.MouseWheelMsg:
		// Handle mouse scroll for scrollback buffer
		mouse := msg.Mouse()
		switch mouse.Button {
		case tea.MouseWheelUp:
			// Scroll up (into history)
			maxScroll := len(m.scrollback)
			m.scrollOffset += 3
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}
			return m, nil
		case tea.MouseWheelDown:
			// Scroll down (toward live)
			m.scrollOffset -= 3
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil
		}
		return m, nil

	case tea.KeyPressMsg:
		if !m.Focused() {
			return m, nil
		}

		// Send input to PTY if running
		if m.running && m.pty != nil {
			var input []byte
			key := msg.Key()
			hasAlt := key.Mod&tea.ModAlt != 0
			hasCtrl := key.Mod&tea.ModCtrl != 0

			// Handle Ctrl key combinations
			if hasCtrl && key.Code >= 'a' && key.Code <= 'z' {
				// Ctrl+A=1 through Ctrl+Z=26
				input = []byte{byte(key.Code - 'a' + 1)}
			} else if hasAlt && key.Code >= 'a' && key.Code <= 'z' {
				// Alt+letter sends ESC + letter (e.g., Alt+B for backward-word)
				input = []byte{27, byte(key.Code)}
			} else {
				// Handle special keys and regular input
				switch key.Code {
				case tea.KeyEnter:
					input = []byte("\r")
				case tea.KeyBackspace:
					if hasAlt {
						input = []byte{27, 127} // ESC + DEL for Alt+Backspace (delete word)
					} else {
						input = []byte{127}
					}
				case tea.KeyTab:
					input = []byte("\t")
				case tea.KeySpace:
					input = []byte(" ")
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
				default:
					// Regular character input
					if key.Text != "" {
						// Filter out mouse/escape sequence fragments
						if looksLikeMouseSequence(key.Text) || looksLikeEscapeFragment(key.Text) {
							return m, nil
						}
						if hasAlt {
							for _, r := range key.Text {
								input = append(input, 27)
								input = append(input, byte(r))
							}
						} else {
							input = []byte(key.Text)
						}
					}
				}
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

		// Large buffer to reduce number of redraws and flickering
		buf := make([]byte, 65536)
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

	// Return cached view if available, otherwise render fresh
	if m.vt != nil {
		if m.cachedView != "" {
			return m.cachedView
		}
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

	// If scrolled up, show scrollback + partial screen
	if m.scrollOffset > 0 && len(m.scrollback) > 0 {
		return m.renderWithScrollback(cols, rows)
	}

	// Live view - render current vt screen
	return m.renderLiveScreen(cols, rows)
}

// renderLiveScreen renders the current vt10x screen (no scrollback)
func (m Model) renderLiveScreen(cols, rows int) string {
	cursor := m.vt.Cursor()
	cursorVisible := m.vt.CursorVisible() && m.Focused()

	var result strings.Builder
	result.Grow(rows * cols * 2)

	for row := 0; row < rows; row++ {
		if row > 0 {
			result.WriteByte('\n')
		}

		var currentFG, currentBG vt10x.Color
		var currentMode int16
		var currentIsCursor bool
		var batch strings.Builder
		firstCell := true

		flushBatch := func() {
			if batch.Len() == 0 {
				return
			}
			result.WriteString(buildANSI(currentFG, currentBG, currentMode, currentIsCursor))
			result.WriteString(batch.String())
			result.WriteString("\x1b[0m")
			batch.Reset()
		}

		for col := 0; col < cols; col++ {
			glyph := m.vt.Cell(col, row)
			ch := glyph.Char
			if ch == 0 {
				ch = ' '
			}

			isCursor := cursorVisible && col == cursor.X && row == cursor.Y

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

// renderWithScrollback renders view including scrollback history
func (m Model) renderWithScrollback(cols, rows int) string {
	var lines []string

	// Calculate which lines to show
	scrollbackLen := len(m.scrollback)
	startIdx := scrollbackLen - m.scrollOffset
	if startIdx < 0 {
		startIdx = 0
	}

	// Add scrollback lines
	for i := startIdx; i < scrollbackLen && len(lines) < rows; i++ {
		lines = append(lines, m.scrollback[i])
	}

	// Add current screen lines if we have room
	if len(lines) < rows {
		screenRows := rows - len(lines)
		for row := 0; row < screenRows; row++ {
			lines = append(lines, m.renderScreenLine(cols, row))
		}
	}

	// Pad with empty lines if needed
	for len(lines) < rows {
		lines = append(lines, strings.Repeat(" ", cols))
	}

	return strings.Join(lines, "\n")
}

// renderScreenLine renders a single line from the vt screen
func (m Model) renderScreenLine(cols, row int) string {
	var result strings.Builder
	var currentFG, currentBG vt10x.Color
	var currentMode int16
	var batch strings.Builder
	firstCell := true

	flushBatch := func() {
		if batch.Len() == 0 {
			return
		}
		result.WriteString(buildANSI(currentFG, currentBG, currentMode, false))
		result.WriteString(batch.String())
		result.WriteString("\x1b[0m")
		batch.Reset()
	}

	for col := 0; col < cols; col++ {
		glyph := m.vt.Cell(col, row)
		ch := glyph.Char
		if ch == 0 {
			ch = ' '
		}

		if !firstCell && (glyph.FG != currentFG || glyph.BG != currentBG || glyph.Mode != currentMode) {
			flushBatch()
		}

		currentFG = glyph.FG
		currentBG = glyph.BG
		currentMode = glyph.Mode
		firstCell = false

		batch.WriteRune(ch)
	}
	flushBatch()

	return result.String()
}

// getScreenLinePlain returns a screen line as plain text (no ANSI codes) for comparison
func (m Model) getScreenLinePlain(cols, row int) string {
	var result strings.Builder
	for col := 0; col < cols; col++ {
		ch := m.vt.Cell(col, row).Char
		if ch == 0 {
			ch = ' '
		}
		result.WriteRune(ch)
	}
	return strings.TrimRight(result.String(), " ")
}

// getTopLine returns the current top line of the terminal as plain text (no ANSI codes)
func (m Model) getTopLine() string {
	if m.vt == nil {
		return ""
	}
	cols, _ := m.vt.Size()
	if cols <= 0 {
		return ""
	}
	return m.getScreenLinePlain(cols, 0)
}

// looksLikeEscapeFragment returns true if the string looks like a fragment of an escape sequence
func looksLikeEscapeFragment(s string) bool {
	// Single [ or < often comes from split escape sequences
	if s == "[" || s == "<" || s == "[<" {
		return true
	}
	// Sequences starting with [ followed by numbers/semicolons (partial CSI)
	if len(s) > 0 && s[0] == '[' {
		for i := 1; i < len(s); i++ {
			c := s[i]
			if c != ';' && c != '<' && (c < '0' || c > '9') {
				return false
			}
		}
		return len(s) > 1
	}
	return false
}

// looksLikeMouseSequence returns true if the string looks like a partial SGR mouse sequence
// These look like "65;83;57M" or "0;45;12m" - numbers, semicolons, ending with M or m
func looksLikeMouseSequence(s string) bool {
	if len(s) < 3 {
		return false
	}
	// Check if it ends with M or m (SGR mouse)
	last := s[len(s)-1]
	if last != 'M' && last != 'm' {
		return false
	}
	// Check if the rest is numbers and semicolons
	for i := 0; i < len(s)-1; i++ {
		c := s[i]
		if c != ';' && (c < '0' || c > '9') && c != '<' {
			return false
		}
	}
	return true
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

	// Reset scroll offset on resize to show live view
	m.scrollOffset = 0

	// Invalidate cached view on resize
	m.cachedView = ""
	m.dirty = true

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

// ScrollPercent returns the scroll position as a percentage (0-100).
// Returns -1 if scrollback is not available or not scrolled.
func (m Model) ScrollPercent() float64 {
	if len(m.scrollback) == 0 || m.scrollOffset == 0 {
		return -1
	}

	// Calculate percentage based on scroll position
	// scrollOffset == len(scrollback) means we're at the top (100%)
	// scrollOffset == 0 means we're at the bottom (live view)
	percent := float64(m.scrollOffset) / float64(len(m.scrollback)) * 100
	if percent > 100 {
		percent = 100
	}
	return percent
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
