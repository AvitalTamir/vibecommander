package viewer

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/avitaltamir/vibecommander/internal/components"
	"github.com/avitaltamir/vibecommander/internal/selection"
	"github.com/avitaltamir/vibecommander/internal/theme"
	"github.com/charmbracelet/x/ansi"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Messages
type (
	// FileLoadedMsg is sent when a file has been loaded.
	FileLoadedMsg struct {
		Path    string
		Content string
		Err     error
	}
)

// Model is the content viewer component.
type Model struct {
	components.Base

	viewport viewport.Model
	path     string
	content  string
	ready    bool
	err      error

	// Search
	searching    bool
	searchInput  textinput.Model
	searchQuery  string
	searchRegex  *regexp.Regexp
	matchLines   []int // Line numbers (0-indexed) with matches
	currentMatch int   // Current match index (-1 if none)

	// Text selection
	selection selection.Model

	theme *theme.Theme
}

// New creates a new content viewer model.
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "regex pattern..."
	ti.CharLimit = 256
	ti.SetWidth(30)

	return Model{
		theme:        theme.DefaultTheme(),
		searchInput:  ti,
		currentMatch: -1,
		selection:    selection.New(),
	}
}

// Init initializes the viewer.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// This is handled by SetSize from parent
		return m, nil

	case tea.MouseClickMsg:
		// Handle text selection start - MouseClickMsg is only for left button
		mouse := msg.Mouse()
		// Start selection - convert screen coordinates to text position
		line, col := m.screenToTextPosition(mouse.X, mouse.Y)
		m.selection.StartSelection(line, col)
		m.updateSelectionContent()
		m.viewport.SetContent(m.renderContent())
		return m, nil

	case tea.MouseMotionMsg:
		// Update selection during drag
		mouse := msg.Mouse()
		if m.selection.Selection.Active {
			line, col := m.screenToTextPosition(mouse.X, mouse.Y)
			m.selection.UpdateSelection(line, col)
			m.viewport.SetContent(m.renderContent())
			return m, nil
		}

	case tea.MouseReleaseMsg:
		// End selection
		mouse := msg.Mouse()
		if m.selection.Selection.Active {
			line, col := m.screenToTextPosition(mouse.X, mouse.Y)
			m.selection.UpdateSelection(line, col)
			m.selection.EndSelection()
			m.viewport.SetContent(m.renderContent())
			return m, nil
		}

	case tea.MouseWheelMsg:
		// Always handle mouse wheel for scrolling, even when not focused
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case FileLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.content = ""
			m.viewport.SetContent(m.renderError(msg.Err))
		} else {
			m.path = msg.Path
			m.content = msg.Content
			m.err = nil
			// Clear search when loading new file
			m.clearSearch()
			m.viewport.SetContent(m.renderContent())
			m.viewport.GotoTop()
		}
		return m, nil

	case tea.KeyPressMsg:
		if !m.Focused() {
			return m, nil
		}

		key := msg.Key()

		// Handle copy (Ctrl+C) when text is selected
		if selection.IsCopyKey(msg.String()) && m.selection.HasSelection() {
			_ = m.selection.CopyToClipboard()
			m.selection.ClearSelection()
			m.viewport.SetContent(m.renderContent())
			return m, nil
		}

		// Clear selection on Escape
		if key.Code == tea.KeyEscape && m.selection.HasSelection() {
			m.selection.ClearSelection()
			m.viewport.SetContent(m.renderContent())
			return m, nil
		}

		// Handle search mode
		if m.searching {
			switch key.Code {
			case tea.KeyEscape:
				// Just close input, keep existing search highlights
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			case tea.KeyEnter:
				// Perform search or go to next match
				query := m.searchInput.Value()
				if query != m.searchQuery {
					// New search
					m.performSearch(query)
				} else if len(m.matchLines) > 0 {
					// Go to next match
					m.currentMatch = (m.currentMatch + 1) % len(m.matchLines)
					m.scrollToCurrentMatch()
				}
				// Close search input and re-render
				m.searching = false
				m.searchInput.Blur()
				m.viewport.SetContent(m.renderContent())
				return m, nil
			default:
				// Pass to text input
				m.searchInput, cmd = m.searchInput.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
		}

		// Normal mode - check for Esc to clear search
		if key.Code == tea.KeyEscape && len(m.matchLines) > 0 {
			m.clearSearch()
			m.viewport.SetContent(m.renderContent())
			return m, nil
		}

		// Normal mode - check for search trigger
		if key.Text == "/" {
			m.searching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		}

		// Check for 'n' to go to next match (when not searching)
		if key.Text == "n" && len(m.matchLines) > 0 {
			m.currentMatch = (m.currentMatch + 1) % len(m.matchLines)
			m.scrollToCurrentMatch()
			m.viewport.SetContent(m.renderContent())
			return m, nil
		}

		// Check for 'p' to go to previous match
		if key.Text == "p" && len(m.matchLines) > 0 {
			m.currentMatch--
			if m.currentMatch < 0 {
				m.currentMatch = len(m.matchLines) - 1
			}
			m.scrollToCurrentMatch()
			m.viewport.SetContent(m.renderContent())
			return m, nil
		}

		// Pass other keys to viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// Handle keyboard only when focused
	if m.Focused() {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the viewer.
func (m Model) View() string {
	if !m.ready {
		return m.renderPlaceholder()
	}

	// If searching, show search bar at bottom
	if m.searching {
		w, h := m.Size()
		viewportHeight := h - 1 // Reserve 1 line for search bar

		// Temporarily resize viewport for display
		oldHeight := m.viewport.Height()
		m.viewport.SetHeight(viewportHeight)
		content := m.viewport.View()
		m.viewport.SetHeight(oldHeight)

		// Render search bar
		searchBar := m.renderSearchBar(w)

		return lipgloss.JoinVertical(lipgloss.Left, content, searchBar)
	}

	return m.viewport.View()
}

func (m Model) renderPlaceholder() string {
	w, h := m.Size()
	style := lipgloss.NewStyle().
		Width(w).
		Height(h).
		Foreground(theme.MutedLavender).
		Align(lipgloss.Center, lipgloss.Center)

	return style.Render("Select a file to view its contents...")
}

func (m Model) renderError(err error) string {
	style := lipgloss.NewStyle().
		Foreground(theme.NeonRed).
		Bold(true)

	return style.Render("Error: " + err.Error())
}

func (m Model) renderContent() string {
	if m.content == "" {
		return lipgloss.NewStyle().
			Foreground(theme.MutedLavender).
			Italic(true).
			Render("(empty file)")
	}

	// Get syntax highlighted content
	highlighted := m.highlightSyntax()

	// Build set of match lines for quick lookup
	matchSet := make(map[int]bool)
	for _, ln := range m.matchLines {
		matchSet[ln] = true
	}
	currentMatchLine := -1
	if m.currentMatch >= 0 && m.currentMatch < len(m.matchLines) {
		currentMatchLine = m.matchLines[m.currentMatch]
	}

	// Check if we have a visible selection (with actual range, not just a click)
	hasSelection := m.selection.HasVisibleSelection()

	// Split both raw and highlighted content
	rawLines := strings.Split(m.content, "\n")
	highlightedLines := strings.Split(highlighted, "\n")

	var result strings.Builder

	lineNumStyle := theme.DiffLineNumberStyle
	sepStyle := lipgloss.NewStyle().Foreground(theme.DimPurple)
	matchLineNumStyle := lipgloss.NewStyle().Foreground(theme.ElectricYellow).Bold(true)
	currentMatchLineNumStyle := lipgloss.NewStyle().Foreground(theme.MatrixGreen).Bold(true)

	for i := 0; i < len(highlightedLines); i++ {
		var lineNum string
		var lineContent string
		sep := sepStyle.Render(" │ ")

		if i == currentMatchLine {
			// Current match - highlight in green, show matched text
			lineNum = currentMatchLineNumStyle.Render(padLeft(i+1, 4))
			sep = lipgloss.NewStyle().Foreground(theme.MatrixGreen).Render(" │ ")
			if i < len(rawLines) {
				lineContent = m.highlightMatchesInLine(rawLines[i], true)
			} else {
				lineContent = highlightedLines[i]
			}
		} else if matchSet[i] {
			// Other match - highlight in yellow, show matched text
			lineNum = matchLineNumStyle.Render(padLeft(i+1, 4))
			sep = lipgloss.NewStyle().Foreground(theme.ElectricYellow).Render(" │ ")
			if i < len(rawLines) {
				lineContent = m.highlightMatchesInLine(rawLines[i], false)
			} else {
				lineContent = highlightedLines[i]
			}
		} else if hasSelection && i < len(rawLines) {
			// Render with selection highlighting on top of syntax highlighting
			lineNum = lineNumStyle.Render(padLeft(i+1, 4))
			lineContent = m.renderLineWithSelection(highlightedLines[i], i)
		} else {
			lineNum = lineNumStyle.Render(padLeft(i+1, 4))
			lineContent = highlightedLines[i]
		}

		result.WriteString(lineNum)
		result.WriteString(sep)
		result.WriteString(lineContent)
		if i < len(highlightedLines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderLineWithSelection renders a line with selection highlighting on top of syntax highlighting.
// Uses ANSI-aware string operations to preserve existing syntax highlighting colors.
func (m Model) renderLineWithSelection(line string, lineNum int) string {
	if !m.selection.Selection.Active && !m.selection.Selection.Complete {
		return line
	}

	// Get normalized selection range
	start, end := m.selection.Selection.Start, m.selection.Selection.End
	// Ensure start comes before end
	if start.Line > end.Line || (start.Line == end.Line && start.Column > end.Column) {
		start, end = end, start
	}

	// Check if this line is within selection
	if lineNum < start.Line || lineNum > end.Line {
		return line
	}

	// Get visible width of the line (excluding ANSI codes)
	lineWidth := ansi.StringWidth(line)

	// Determine selection bounds for this line
	selStart := 0
	selEnd := lineWidth

	if lineNum == start.Line {
		selStart = start.Column
	}
	if lineNum == end.Line {
		selEnd = end.Column
	}

	// Clamp to valid range
	if selStart < 0 {
		selStart = 0
	}
	if selEnd > lineWidth {
		selEnd = lineWidth
	}
	if selStart >= selEnd {
		return line
	}

	// Use ANSI-aware cutting to preserve syntax highlighting
	// ansi.Cut(s, left, right) extracts characters from position left to right
	selStyle := selection.SelectionStyle()
	var result strings.Builder

	// Part before selection (preserves syntax highlighting)
	if selStart > 0 {
		result.WriteString(ansi.Cut(line, 0, selStart))
	}

	// Selected part - apply selection background on top of syntax colors
	// We need to handle ANSI reset sequences inside the selected part,
	// as they would clear our background. Replace resets with reset+background.
	selectedPart := ansi.Cut(line, selStart, selEnd)

	// Selection background color from theme: #3D2D5E = rgb(61, 45, 94)
	// ANSI 24-bit background: \x1b[48;2;R;G;Bm
	selBg := "\x1b[48;2;61;45;94m"
	// Replace SGR reset sequences with reset + re-apply background
	selectedPart = strings.ReplaceAll(selectedPart, "\x1b[0m", "\x1b[0m"+selBg)
	selectedPart = strings.ReplaceAll(selectedPart, "\x1b[m", "\x1b[m"+selBg)

	result.WriteString(selStyle.Render(selectedPart))

	// Part after selection (preserves syntax highlighting)
	if selEnd < lineWidth {
		result.WriteString(ansi.Cut(line, selEnd, lineWidth))
	}

	return result.String()
}

// highlightMatchesInLine highlights all regex matches within a line.
func (m Model) highlightMatchesInLine(line string, isCurrent bool) string {
	if m.searchRegex == nil {
		return line
	}

	matches := m.searchRegex.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return line
	}

	// Define highlight styles
	var matchStyle lipgloss.Style
	if isCurrent {
		matchStyle = lipgloss.NewStyle().
			Background(theme.MatrixGreen).
			Foreground(lipgloss.Color("0"))
	} else {
		matchStyle = lipgloss.NewStyle().
			Background(theme.ElectricYellow).
			Foreground(lipgloss.Color("0"))
	}

	// Build result with highlighted matches
	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		start, end := match[0], match[1]
		// Add text before match
		if start > lastEnd {
			result.WriteString(line[lastEnd:start])
		}
		// Add highlighted match
		result.WriteString(matchStyle.Render(line[start:end]))
		lastEnd = end
	}
	// Add remaining text
	if lastEnd < len(line) {
		result.WriteString(line[lastEnd:])
	}

	return result.String()
}

// highlightSyntax returns syntax-highlighted content
func (m Model) highlightSyntax() string {
	// Try to get lexer by filename
	var lexer chroma.Lexer
	if m.path != "" {
		lexer = lexers.Match(filepath.Base(m.path))
	}

	// Fallback: try to analyze content
	if lexer == nil {
		lexer = lexers.Analyse(m.content)
	}

	// Final fallback: plain text
	if lexer == nil {
		lexer = lexers.Fallback
	}

	lexer = chroma.Coalesce(lexer)

	// Use a dark theme that works well with terminal
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	// Use terminal256 formatter for colored output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize and format
	iterator, err := lexer.Tokenise(nil, m.content)
	if err != nil {
		return m.content
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return m.content
	}

	return buf.String()
}

// LoadFile loads a file into the viewer.
func LoadFile(path string) tea.Cmd {
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		if err != nil {
			return FileLoadedMsg{Path: path, Err: err}
		}
		return FileLoadedMsg{Path: path, Content: string(content)}
	}
}

// SetContent sets the content directly (for non-file content).
func (m *Model) SetContent(content string) {
	m.content = content
	m.path = ""
	m.err = nil
	m.viewport.SetContent(m.renderContent())
	m.viewport.GotoTop()
}

// Path returns the current file path.
func (m Model) Path() string {
	return m.path
}

// Content returns the current content.
func (m Model) Content() string {
	return m.content
}

// Clear clears the viewer.
func (m *Model) Clear() {
	m.path = ""
	m.content = ""
	m.err = nil
	m.ready = false
	m.viewport.SetContent("")
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

	// Initialize or resize viewport
	if !m.ready {
		m.viewport = viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
		m.viewport.MouseWheelEnabled = true
		m.viewport.MouseWheelDelta = 3
		m.ready = true
	} else {
		m.viewport.SetWidth(width)
		m.viewport.SetHeight(height)
	}

	// Re-render content to fit new width
	if m.content != "" {
		m.viewport.SetContent(m.renderContent())
	}

	return m
}

// ScrollPercent returns the current scroll position as a percentage (0-100).
func (m Model) ScrollPercent() float64 {
	return m.viewport.ScrollPercent() * 100
}

// IsSearching returns whether the viewer is in search mode.
func (m Model) IsSearching() bool {
	return m.searching
}

// HasActiveSearch returns whether there's an active search query (for n/p navigation).
func (m Model) HasActiveSearch() bool {
	return m.searchQuery != ""
}

// renderSearchBar renders the search input bar.
func (m Model) renderSearchBar(width int) string {
	prefix := lipgloss.NewStyle().
		Foreground(theme.CyberCyan).
		Bold(true).
		Render("/")

	// Match info
	var matchInfo string
	if m.searchQuery != "" {
		if len(m.matchLines) == 0 {
			matchInfo = lipgloss.NewStyle().
				Foreground(theme.NeonRed).
				Render(" [no matches]")
		} else {
			matchInfo = lipgloss.NewStyle().
				Foreground(theme.MatrixGreen).
				Render(" [" + itoa(m.currentMatch+1) + "/" + itoa(len(m.matchLines)) + "]")
		}
	}

	input := m.searchInput.View()

	// Combine and pad to width
	bar := prefix + input + matchInfo
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(width)

	return style.Render(bar)
}

// performSearch searches for the regex pattern in the content.
func (m *Model) performSearch(query string) {
	m.searchQuery = query
	m.matchLines = nil
	m.currentMatch = -1
	m.searchRegex = nil

	if query == "" {
		return
	}

	// Compile regex (case-insensitive by default)
	re, err := regexp.Compile("(?i)" + query)
	if err != nil {
		// Invalid regex, try as literal
		re, err = regexp.Compile(regexp.QuoteMeta(query))
		if err != nil {
			return
		}
	}
	m.searchRegex = re

	// Find all matching lines
	lines := strings.Split(m.content, "\n")
	for i, line := range lines {
		if re.MatchString(line) {
			m.matchLines = append(m.matchLines, i)
		}
	}

	// Go to first match if found
	if len(m.matchLines) > 0 {
		m.currentMatch = 0
		m.scrollToCurrentMatch()
	}
}

// scrollToCurrentMatch scrolls the viewport to show the current match.
func (m *Model) scrollToCurrentMatch() {
	if m.currentMatch < 0 || m.currentMatch >= len(m.matchLines) {
		return
	}

	line := m.matchLines[m.currentMatch]
	// Scroll so the match is roughly centered
	targetLine := line - m.viewport.Height()/2
	if targetLine < 0 {
		targetLine = 0
	}
	m.viewport.SetYOffset(targetLine)
}

// clearSearch clears the search state.
func (m *Model) clearSearch() {
	m.searching = false
	m.searchQuery = ""
	m.searchRegex = nil
	m.matchLines = nil
	m.currentMatch = -1
	m.searchInput.SetValue("")
	m.searchInput.Blur()
}

// itoa converts int to string without importing strconv
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var s string
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// Helper to pad line numbers
func padLeft(n, width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s += " "
	}
	ns := strings.TrimLeft(s+string(rune('0'+n%10)), " ")
	if n >= 10 {
		ns = string(rune('0'+n/10%10)) + string(rune('0'+n%10))
	}
	if n >= 100 {
		ns = string(rune('0'+n/100%10)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
	}
	if n >= 1000 {
		ns = string(rune('0'+n/1000%10)) + string(rune('0'+n/100%10)) + string(rune('0'+n/10%10)) + string(rune('0'+n%10))
	}

	// Pad to width
	for len(ns) < width {
		ns = " " + ns
	}
	return ns
}

// lineNumberWidth is the width of line numbers plus separator (" 123 │ ")
const lineNumberWidth = 7 // 4 digits + " │ "

// screenToTextPosition converts screen coordinates to text line and column.
// Takes into account the viewport scroll offset, line number prefix, and panel border.
func (m Model) screenToTextPosition(x, y int) (line, col int) {
	// Y coordinate: subtract 1 for top border, then add viewport scroll offset
	line = y - 1 + m.viewport.YOffset()
	if line < 0 {
		line = 0
	}

	// X coordinate: subtract 1 for left border, then subtract line number prefix width
	col = x - 1 - lineNumberWidth
	if col < 0 {
		col = 0
	}

	return line, col
}

// updateSelectionContent updates the selection model with the current content lines.
func (m *Model) updateSelectionContent() {
	if m.content == "" {
		m.selection.SetContent(nil)
		return
	}
	lines := strings.Split(m.content, "\n")
	m.selection.SetContent(lines)
}

// HasSelection returns true if there is an active text selection.
func (m Model) HasSelection() bool {
	return m.selection.HasSelection()
}

// GetSelectedText returns the currently selected text.
func (m Model) GetSelectedText() string {
	return m.selection.GetSelectedText()
}
