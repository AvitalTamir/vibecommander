package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/selection"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"github.com/hinshun/vt10x"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
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

	// Scrollback buffer
	scrollback    []string // Lines that scrolled off the top
	scrollOffset  int      // 0 = live view, >0 = scrolled up N lines
	maxScrollback int      // Max lines to keep

	// Text selection
	selection selection.Model

	theme *theme.Theme
	ready bool
}

// New creates a new terminal model.
func New() Model {
	return Model{
		theme:         theme.DefaultTheme(),
		maxScrollback: 10000,
		selection:     selection.New(),
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
		m.mu.Unlock()
		return m, m.ContinueReading()

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

	case tea.MouseMsg:
		// Handle text selection with left button
		switch msg.Button {
		case tea.MouseButtonLeft:
			switch msg.Action {
			case tea.MouseActionPress:
				// Start selection
				line, col := m.screenToTextPosition(msg.X, msg.Y)
				m.selection.StartSelection(line, col)
				m.updateSelectionContent()
				return m, nil
			case tea.MouseActionMotion:
				// Update selection during drag
				if m.selection.Selection.Active {
					line, col := m.screenToTextPosition(msg.X, msg.Y)
					m.selection.UpdateSelection(line, col)
					return m, nil
				}
			case tea.MouseActionRelease:
				// End selection
				if m.selection.Selection.Active {
					line, col := m.screenToTextPosition(msg.X, msg.Y)
					m.selection.UpdateSelection(line, col)
					m.selection.EndSelection()
					return m, nil
				}
			}
		case tea.MouseButtonWheelUp:
			// Scroll up (into history)
			maxScroll := len(m.scrollback)
			m.scrollOffset += 3
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}
			return m, nil
		case tea.MouseButtonWheelDown:
			// Scroll down (toward live)
			m.scrollOffset -= 3
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil
		}
		return m, nil

	case tea.KeyMsg:
		if !m.Focused() {
			return m, nil
		}

		// Handle copy (Ctrl+C) when text is selected - copy instead of SIGINT
		if selection.IsCopyKey(msg.String()) && m.selection.HasSelection() {
			_ = m.selection.CopyToClipboard()
			m.selection.ClearSelection()
			return m, nil
		}

		// Clear selection on Escape
		if msg.Type == tea.KeyEscape && m.selection.HasSelection() {
			m.selection.ClearSelection()
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
				// Filter out mouse/escape sequence fragments
				runeStr := string(msg.Runes)
				if looksLikeMouseSequence(runeStr) || looksLikeEscapeFragment(runeStr) {
					return m, nil
				}
				// Check for Alt modifier (sends ESC + char)
				if msg.Alt {
					for _, r := range msg.Runes {
						input = append(input, 27) // ESC
						input = append(input, byte(r))
					}
				} else {
					input = []byte(runeStr)
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

		// Large buffer to reduce number of redraws and flickering
		buf := make([]byte, 65536)
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
	if m.vt != nil {
		return m.renderVT()
	}

	return lipgloss.NewStyle().
		Foreground(theme.MutedLavender).
		Italic(true).
		Render("Terminal ready...")
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
	hasSelection := m.selection.Selection.Active || m.selection.Selection.Complete

	var result strings.Builder
	result.Grow(rows * cols * 2)

	for row := 0; row < rows; row++ {
		if row > 0 {
			result.WriteByte('\n')
		}

		var currentFG, currentBG vt10x.Color
		var currentMode int16
		var currentIsCursor bool
		var currentIsSelected bool
		var batch strings.Builder
		firstCell := true

		flushBatch := func() {
			if batch.Len() == 0 {
				return
			}
			result.WriteString(buildANSI(currentFG, currentBG, currentMode, currentIsCursor, currentIsSelected))
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
			isSelected := hasSelection && m.selection.IsSelected(row, col)

			if !firstCell && (glyph.FG != currentFG || glyph.BG != currentBG || glyph.Mode != currentMode || isCursor != currentIsCursor || isSelected != currentIsSelected) {
				flushBatch()
			}

			currentFG = glyph.FG
			currentBG = glyph.BG
			currentMode = glyph.Mode
			currentIsCursor = isCursor
			currentIsSelected = isSelected
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
		result.WriteString(buildANSI(currentFG, currentBG, currentMode, false, false))
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
func buildANSI(fg, bg vt10x.Color, mode int16, isCursor, isSelected bool) string {
	var codes []string

	if isCursor {
		codes = append(codes, "7") // Reverse video
	} else if isSelected {
		// Selection styling - reverse video (swap fg/bg)
		codes = append(codes, "7")
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

	// Account for status line
	termHeight := height - 1
	if termHeight < 0 {
		termHeight = 0
	}

	if !m.ready {
		m.ready = true
	}

	// Reset scroll offset on resize to show live view
	m.scrollOffset = 0

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

// screenToTextPosition converts screen coordinates to text line and column.
// For terminal, coordinates map directly to the visible buffer position.
func (m Model) screenToTextPosition(x, y int) (line, col int) {
	// Y coordinate: adjust for scroll offset (if viewing scrollback)
	line = y
	if m.scrollOffset > 0 {
		// When scrolled up, the visible lines are from scrollback
		scrollbackLen := len(m.scrollback)
		startIdx := scrollbackLen - m.scrollOffset
		if startIdx < 0 {
			startIdx = 0
		}
		line = startIdx + y
	}

	// X coordinate maps directly
	col = x

	return line, col
}

// updateSelectionContent updates the selection model with all visible text content.
func (m *Model) updateSelectionContent() {
	if m.vt == nil {
		m.selection.SetContent(nil)
		return
	}

	m.vt.Lock()
	defer m.vt.Unlock()

	cols, rows := m.vt.Size()
	if cols <= 0 || rows <= 0 {
		m.selection.SetContent(nil)
		return
	}

	var lines []string

	// If scrolled up, include scrollback
	if m.scrollOffset > 0 && len(m.scrollback) > 0 {
		scrollbackLen := len(m.scrollback)
		startIdx := scrollbackLen - m.scrollOffset
		if startIdx < 0 {
			startIdx = 0
		}
		// Add scrollback lines (these contain ANSI codes, so extract plain text)
		for i := startIdx; i < scrollbackLen && len(lines) < rows; i++ {
			lines = append(lines, stripANSI(m.scrollback[i]))
		}
		// Add current screen lines if we have room
		if len(lines) < rows {
			screenRows := rows - len(lines)
			for row := 0; row < screenRows; row++ {
				lines = append(lines, m.getScreenLinePlain(cols, row))
			}
		}
	} else {
		// Live view - get current screen content
		for row := 0; row < rows; row++ {
			lines = append(lines, m.getScreenLinePlain(cols, row))
		}
	}

	m.selection.SetContent(lines)
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	// Simple ANSI stripper - removes escape sequences
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find a letter (end of sequence)
			i += 2
			for i < len(s) && !((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
				i++
			}
			if i < len(s) {
				i++ // Skip the final letter
			}
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// HasSelection returns true if there is an active text selection.
func (m Model) HasSelection() bool {
	return m.selection.HasSelection()
}

// GetSelectedText returns the currently selected text.
func (m Model) GetSelectedText() string {
	return m.selection.GetSelectedText()
}
