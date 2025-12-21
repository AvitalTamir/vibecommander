package selection

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/avitaltamir/vibecommander/internal/theme"
)

// Position represents a character position in the text.
type Position struct {
	Line   int // 0-indexed line number
	Column int // 0-indexed column position
}

// Selection represents a text selection range.
type Selection struct {
	Active   bool     // Whether selection is in progress
	Start    Position // Where selection started (anchor)
	End      Position // Current selection end
	Complete bool     // Whether selection is complete (mouse released)
}

// Model holds the selection state for a component.
type Model struct {
	Selection Selection
	content   []string // The text content being displayed
}

// New creates a new selection model.
func New() Model {
	return Model{}
}

// SetContent sets the text content for selection.
func (m *Model) SetContent(lines []string) {
	m.content = lines
}

// StartSelection begins a new selection at the given position.
func (m *Model) StartSelection(line, col int) {
	m.Selection = Selection{
		Active:   true,
		Start:    Position{Line: line, Column: col},
		End:      Position{Line: line, Column: col},
		Complete: false,
	}
}

// UpdateSelection updates the selection end position during drag.
func (m *Model) UpdateSelection(line, col int) {
	if !m.Selection.Active {
		return
	}
	m.Selection.End = Position{Line: line, Column: col}
}

// EndSelection marks the selection as complete.
func (m *Model) EndSelection() {
	if !m.Selection.Active {
		return
	}
	m.Selection.Complete = true
	m.Selection.Active = false
}

// ClearSelection clears any active selection.
func (m *Model) ClearSelection() {
	m.Selection = Selection{}
}

// HasSelection returns true if there's a valid completed selection.
func (m Model) HasSelection() bool {
	return m.Selection.Complete && !m.positionsEqual(m.Selection.Start, m.Selection.End)
}

// HasVisibleSelection returns true if there's a visible selection (active or complete with different start/end).
func (m Model) HasVisibleSelection() bool {
	if !m.Selection.Active && !m.Selection.Complete {
		return false
	}
	return !m.positionsEqual(m.Selection.Start, m.Selection.End)
}

// GetSelectedText returns the selected text from the content.
func (m Model) GetSelectedText() string {
	if !m.HasSelection() {
		return ""
	}

	start, end := m.normalizeRange()
	if len(m.content) == 0 {
		return ""
	}

	// Clamp to valid range
	if start.Line < 0 {
		start.Line = 0
	}
	if start.Line >= len(m.content) {
		return ""
	}
	if end.Line >= len(m.content) {
		end.Line = len(m.content) - 1
		end.Column = len(m.content[end.Line])
	}

	// Single line selection
	if start.Line == end.Line {
		line := m.content[start.Line]
		startCol := clamp(start.Column, 0, len(line))
		endCol := clamp(end.Column, 0, len(line))
		if startCol > endCol {
			startCol, endCol = endCol, startCol
		}
		return line[startCol:endCol]
	}

	// Multi-line selection
	var result strings.Builder

	// First line (from start column to end)
	firstLine := m.content[start.Line]
	startCol := clamp(start.Column, 0, len(firstLine))
	result.WriteString(firstLine[startCol:])
	result.WriteString("\n")

	// Middle lines (complete lines)
	for i := start.Line + 1; i < end.Line; i++ {
		result.WriteString(m.content[i])
		result.WriteString("\n")
	}

	// Last line (from start to end column)
	lastLine := m.content[end.Line]
	endCol := clamp(end.Column, 0, len(lastLine))
	result.WriteString(lastLine[:endCol])

	return result.String()
}

// CopyToClipboard copies the selected text to the system clipboard.
func (m Model) CopyToClipboard() error {
	text := m.GetSelectedText()
	if text == "" {
		return nil
	}
	return clipboard.WriteAll(text)
}

// IsSelected returns true if the character at (line, col) is within the selection.
func (m Model) IsSelected(line, col int) bool {
	if !m.Selection.Complete && !m.Selection.Active {
		return false
	}

	start, end := m.normalizeRange()

	// Before selection start
	if line < start.Line {
		return false
	}
	if line == start.Line && col < start.Column {
		return false
	}

	// After selection end
	if line > end.Line {
		return false
	}
	if line == end.Line && col >= end.Column {
		return false
	}

	return true
}

// normalizeRange returns start and end positions in order (start < end).
func (m Model) normalizeRange() (Position, Position) {
	start := m.Selection.Start
	end := m.Selection.End

	// Ensure start comes before end
	if start.Line > end.Line || (start.Line == end.Line && start.Column > end.Column) {
		start, end = end, start
	}

	return start, end
}

// positionsEqual returns true if two positions are the same.
func (m Model) positionsEqual(a, b Position) bool {
	return a.Line == b.Line && a.Column == b.Column
}

// IsCopyKey returns true if the key message is a copy command.
// On macOS, Cmd+C is intercepted by the terminal emulator before reaching the app,
// so we support multiple key bindings for copy:
// - Ctrl+C: Standard terminal copy (when text is selected; otherwise SIGINT)
// - y: Vim-style yank
// - Ctrl+Y: Alternative copy binding
func IsCopyKey(key string) bool {
	switch key {
	case "ctrl+c", "y", "ctrl+y":
		return true
	default:
		return false
	}
}

// SelectionStyle returns the lipgloss style for selected text.
func SelectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(theme.BgSelection).
		Foreground(theme.TextSelection)
}

// RenderWithSelection renders a line of text with selection highlighting.
// The offset parameter adjusts for line numbers or other prefixes.
func RenderWithSelection(line string, lineNum int, sel *Model, offset int) string {
	if sel == nil || (!sel.Selection.Active && !sel.Selection.Complete) {
		return line
	}

	// Check if any part of this line is selected
	start, end := sel.normalizeRange()
	if lineNum < start.Line || lineNum > end.Line {
		return line
	}

	// Determine selection bounds for this line
	selStart := 0
	selEnd := len(line)

	if lineNum == start.Line {
		selStart = start.Column - offset
	}
	if lineNum == end.Line {
		selEnd = end.Column - offset
	}

	// Clamp to valid range
	if selStart < 0 {
		selStart = 0
	}
	if selEnd > len(line) {
		selEnd = len(line)
	}
	if selStart >= selEnd {
		return line
	}

	// Build result with selection highlighting
	style := SelectionStyle()
	var result strings.Builder

	if selStart > 0 {
		result.WriteString(line[:selStart])
	}
	result.WriteString(style.Render(line[selStart:selEnd]))
	if selEnd < len(line) {
		result.WriteString(line[selEnd:])
	}

	return result.String()
}

// clamp restricts v to the range [min, max].
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
