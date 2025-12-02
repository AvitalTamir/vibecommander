package theme

import "github.com/charmbracelet/lipgloss"

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
