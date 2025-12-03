package theme

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// Border definitions
var (
	// NeonBorder uses heavy lines for a bold look
	NeonBorder = lipgloss.Border{
		Top:         "━",
		Bottom:      "━",
		Left:        "┃",
		Right:       "┃",
		TopLeft:     "┏",
		TopRight:    "┓",
		BottomLeft:  "┗",
		BottomRight: "┛",
	}

	// GlowBorder uses rounded corners for a softer look
	GlowBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "╰",
		BottomRight: "╯",
	}

	// DoubleBorder for important content
	DoubleBorder = lipgloss.Border{
		Top:         "═",
		Bottom:      "═",
		Left:        "║",
		Right:       "║",
		TopLeft:     "╔",
		TopRight:    "╗",
		BottomLeft:  "╚",
		BottomRight: "╝",
	}
)

// Panel styles
var (
	PanelInactive lipgloss.Style
	PanelActive   lipgloss.Style
	PanelFocused  lipgloss.Style
)

// Text styles - hierarchy from most to least prominent
var (
	TextH1             lipgloss.Style
	TextH2             lipgloss.Style
	TextBody           lipgloss.Style
	TextSecondaryStyle lipgloss.Style
	TextMutedStyle     lipgloss.Style
	TextDimStyle       lipgloss.Style
)

// File tree styles
var (
	FileTreeDir      lipgloss.Style
	FileTreeFile     lipgloss.Style
	FileTreeSelected lipgloss.Style
)

// Git status styles
var (
	GitStatusModified  lipgloss.Style
	GitStatusAdded     lipgloss.Style
	GitStatusDeleted   lipgloss.Style
	GitStatusUntracked lipgloss.Style
	GitStatusConflict  lipgloss.Style
	GitBranchStyle     lipgloss.Style
	GitAheadStyle      lipgloss.Style
	GitBehindStyle     lipgloss.Style
)

// Diff styles
var (
	DiffAddedStyle      lipgloss.Style
	DiffRemovedStyle    lipgloss.Style
	DiffContextStyle    lipgloss.Style
	DiffHunkStyle       lipgloss.Style
	DiffLineNumberStyle lipgloss.Style
)

// Status bar styles
var (
	StatusBarStyle     lipgloss.Style
	StatusBarSection   lipgloss.Style
	StatusBarHighlight lipgloss.Style
)

// Spinner style
var SpinnerStyle lipgloss.Style

// regenerateStyles rebuilds all style variables based on current color values.
// Called when theme changes.
func regenerateStyles() {
	// Panel styles
	PanelInactive = lipgloss.NewStyle().
		Border(GlowBorder).
		BorderForeground(DimPurple)

	PanelActive = lipgloss.NewStyle().
		Border(GlowBorder).
		BorderForeground(CyberCyan)

	PanelFocused = lipgloss.NewStyle().
		Border(NeonBorder).
		BorderForeground(MagentaBlaze)

	// Text styles
	TextH1 = lipgloss.NewStyle().
		Bold(true).
		Foreground(CyberCyan)

	TextH2 = lipgloss.NewStyle().
		Bold(true).
		Foreground(MagentaBlaze)

	TextBody = lipgloss.NewStyle().
		Foreground(PureWhite)

	TextSecondaryStyle = lipgloss.NewStyle().
		Foreground(Silver)

	TextMutedStyle = lipgloss.NewStyle().
		Foreground(MutedLavender).
		Italic(true)

	TextDimStyle = lipgloss.NewStyle().
		Foreground(DimPurple).
		Faint(true)

	// File tree styles
	FileTreeDir = lipgloss.NewStyle().
		Foreground(CyberCyan).
		Bold(true)

	FileTreeFile = lipgloss.NewStyle().
		Foreground(PureWhite)

	FileTreeSelected = lipgloss.NewStyle().
		Foreground(MagentaBlaze).
		Bold(true)

	// Git status styles
	GitStatusModified = lipgloss.NewStyle().
		Foreground(ElectricYellow)

	GitStatusAdded = lipgloss.NewStyle().
		Foreground(MatrixGreen)

	GitStatusDeleted = lipgloss.NewStyle().
		Foreground(NeonRed)

	GitStatusUntracked = lipgloss.NewStyle().
		Foreground(LaserPurple)

	GitStatusConflict = lipgloss.NewStyle().
		Foreground(NeonRed).
		Bold(true)

	GitBranchStyle = lipgloss.NewStyle().
		Foreground(CyberCyan).
		Bold(true)

	GitAheadStyle = lipgloss.NewStyle().
		Foreground(MatrixGreen)

	GitBehindStyle = lipgloss.NewStyle().
		Foreground(NeonRed)

	// Diff styles
	DiffAddedStyle = lipgloss.NewStyle().
		Foreground(MatrixGreen)

	DiffRemovedStyle = lipgloss.NewStyle().
		Foreground(NeonRed)

	DiffContextStyle = lipgloss.NewStyle().
		Foreground(Silver)

	DiffHunkStyle = lipgloss.NewStyle().
		Foreground(LaserPurple).
		Bold(true)

	DiffLineNumberStyle = lipgloss.NewStyle().
		Foreground(DimPurple).
		Width(4).
		Align(lipgloss.Right)

	// Status bar styles
	StatusBarStyle = lipgloss.NewStyle().
		Foreground(Silver).
		Padding(0, 1)

	StatusBarSection = lipgloss.NewStyle().
		Foreground(MutedLavender).
		Padding(0, 1)

	StatusBarHighlight = lipgloss.NewStyle().
		Foreground(CyberCyan).
		Bold(true)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
		Foreground(MagentaBlaze)
}

// GetPanelStyle returns the appropriate panel style based on focus state.
func GetPanelStyle(focused bool) lipgloss.Style {
	if focused {
		return PanelFocused
	}
	return PanelInactive
}

// GetGitStatusStyle returns the style for a git status code.
func GetGitStatusStyle(code rune) lipgloss.Style {
	switch code {
	case 'M':
		return GitStatusModified
	case 'A', '+':
		return GitStatusAdded
	case 'D':
		return GitStatusDeleted
	case '?':
		return GitStatusUntracked
	case 'U', '!':
		return GitStatusConflict
	default:
		return TextBody
	}
}

// RenderTitle renders a panel title with decorations.
func RenderTitle(title string, focused bool) string {
	accent := DimPurple
	if focused {
		accent = CyberCyan
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(accent).
		Bold(true)

	diamond := lipgloss.NewStyle().
		Foreground(MagentaBlaze).
		Render(PanelDiamond)

	return diamond + "─[ " + titleStyle.Render(title) + " ]─"
}

// FormatScrollIndicator returns a formatted scroll percentage indicator.
// Returns empty string if percent is 100 (at bottom) or invalid.
func FormatScrollIndicator(percent float64) string {
	if percent >= 99.9 || percent < 0 {
		return ""
	}
	return fmt.Sprintf("%d%%", int(percent))
}

// FormatStatusIndicator returns a running/idle status indicator.
func FormatStatusIndicator(running bool) string {
	if running {
		return StatusRunning
	}
	return StatusIdle
}

// PanelTitleOptions configures what to show in panel borders.
type PanelTitleOptions struct {
	Title         string  // Main title text (e.g., "FILES", "Claude")
	StatusRunning bool    // Show running indicator (●) vs idle (○)
	ShowStatus    bool    // Whether to show status at all
	ScrollPercent float64 // Scroll position (0-100), negative to hide
	BottomHints   string  // Key hints for bottom border (e.g., "↑↓:scroll  q:quit")
}

// RenderPanelWithTitle renders content in a panel with title embedded in the border.
func RenderPanelWithTitle(content string, opts PanelTitleOptions, width, height int, focused bool) string {
	if width < 4 || height < 2 {
		return ""
	}

	// Choose border style and colors based on focus
	var border lipgloss.Border
	var borderColor lipgloss.Color
	var titleColor lipgloss.Color

	if focused {
		border = NeonBorder
		borderColor = MagentaBlaze
		titleColor = CyberCyan
	} else {
		border = GlowBorder
		borderColor = DimPurple
		titleColor = DimPurple
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(MutedLavender)
	scrollStyle := lipgloss.NewStyle().Foreground(DimPurple)
	statusStyle := lipgloss.NewStyle().Foreground(MatrixGreen)
	if !opts.StatusRunning {
		statusStyle = lipgloss.NewStyle().Foreground(DimPurple)
	}

	// Calculate inner width (minus 2 for side borders)
	innerWidth := width - 2

	// Build top border with title
	topBorder := buildTopBorder(border, borderStyle, titleStyle, scrollStyle, statusStyle, opts, innerWidth)

	// Build bottom border with hints
	bottomBorder := buildBottomBorder(border, borderStyle, hintStyle, opts.BottomHints, innerWidth)

	// Build content area
	contentHeight := height - 2 // Account for top and bottom borders
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Split content into lines and pad/truncate to fit
	contentLines := strings.Split(content, "\n")
	renderedLines := make([]string, contentHeight)

	// Style for truncating lines with ANSI codes
	lineStyle := lipgloss.NewStyle().MaxWidth(innerWidth)

	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(contentLines) {
			line = contentLines[i]
		}
		// Truncate line (handles ANSI codes properly)
		line = lineStyle.Render(line)
		// Pad to fill width
		lineLen := lipgloss.Width(line)
		if lineLen < innerWidth {
			line = line + strings.Repeat(" ", innerWidth-lineLen)
		}
		renderedLines[i] = borderStyle.Render(border.Left) + line + borderStyle.Render(border.Right)
	}

	// Join all parts
	var result strings.Builder
	result.WriteString(topBorder)
	result.WriteString("\n")
	result.WriteString(strings.Join(renderedLines, "\n"))
	result.WriteString("\n")
	result.WriteString(bottomBorder)

	return result.String()
}

// buildTopBorder creates the top border with title and optional scroll/status indicators.
func buildTopBorder(border lipgloss.Border, borderStyle, titleStyle, scrollStyle, statusStyle lipgloss.Style, opts PanelTitleOptions, innerWidth int) string {
	// Format the title segment: "[ Title ● ]" or "[ Title ]"
	titleText := opts.Title
	if opts.ShowStatus {
		status := FormatStatusIndicator(opts.StatusRunning)
		titleText = titleText + " " + statusStyle.Render(status)
	}
	titleSegment := "[ " + titleStyle.Render(opts.Title)
	if opts.ShowStatus {
		titleSegment += " " + statusStyle.Render(FormatStatusIndicator(opts.StatusRunning))
	}
	titleSegment += " ]"

	// Format scroll indicator if applicable
	var scrollSegment string
	if opts.ScrollPercent >= 0 && opts.ScrollPercent < 99.9 {
		scrollText := FormatScrollIndicator(opts.ScrollPercent)
		if scrollText != "" {
			scrollSegment = "[ " + scrollStyle.Render(scrollText) + " ]"
		}
	}

	// Calculate visible widths (without ANSI codes)
	titleWidth := utf8.RuneCountInString(stripAnsi(titleSegment))
	scrollWidth := 0
	if scrollSegment != "" {
		scrollWidth = utf8.RuneCountInString(stripAnsi(scrollSegment))
	}

	// Calculate filler lengths
	leftFiller := 2 // Small gap after corner
	rightFiller := innerWidth - leftFiller - titleWidth - scrollWidth
	if rightFiller < 0 {
		rightFiller = 0
	}

	// Build the border
	var result strings.Builder
	result.WriteString(borderStyle.Render(border.TopLeft))
	result.WriteString(borderStyle.Render(strings.Repeat(border.Top, leftFiller)))
	result.WriteString(titleSegment)
	if scrollSegment != "" {
		result.WriteString(borderStyle.Render(strings.Repeat(border.Top, rightFiller-scrollWidth)))
		result.WriteString(scrollSegment)
		result.WriteString(borderStyle.Render(strings.Repeat(border.Top, scrollWidth)))
	} else {
		result.WriteString(borderStyle.Render(strings.Repeat(border.Top, rightFiller)))
	}
	result.WriteString(borderStyle.Render(border.TopRight))

	return result.String()
}

// buildBottomBorder creates the bottom border with optional key hints.
func buildBottomBorder(border lipgloss.Border, borderStyle, hintStyle lipgloss.Style, hints string, innerWidth int) string {
	if hints == "" {
		// Simple border without hints
		return borderStyle.Render(border.BottomLeft) +
			borderStyle.Render(strings.Repeat(border.Bottom, innerWidth)) +
			borderStyle.Render(border.BottomRight)
	}

	// Format hint segment
	hintSegment := "[ " + hintStyle.Render(hints) + " ]"
	hintWidth := utf8.RuneCountInString(stripAnsi(hintSegment))

	// Calculate filler lengths
	leftFiller := 2
	rightFiller := innerWidth - leftFiller - hintWidth
	if rightFiller < 0 {
		rightFiller = 0
	}

	var result strings.Builder
	result.WriteString(borderStyle.Render(border.BottomLeft))
	result.WriteString(borderStyle.Render(strings.Repeat(border.Bottom, leftFiller)))
	result.WriteString(hintSegment)
	result.WriteString(borderStyle.Render(strings.Repeat(border.Bottom, rightFiller)))
	result.WriteString(borderStyle.Render(border.BottomRight))

	return result.String()
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

