package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Neon Core Colors - Primary accent colors for the cyberpunk theme
var (
	MagentaBlaze   color.Color = lipgloss.Color("#FF00FF") // Primary accent
	CyberCyan      color.Color = lipgloss.Color("#00FFFF") // Secondary accent
	HotPink        color.Color = lipgloss.Color("#FF10F0") // Selections/Focus
	MatrixGreen    color.Color = lipgloss.Color("#39FF14") // Git additions/success
	NeonRed        color.Color = lipgloss.Color("#FF3131") // Git deletions/errors
	ElectricYellow color.Color = lipgloss.Color("#FFFF00") // Warnings/modified
	LaserPurple    color.Color = lipgloss.Color("#7B68EE") // AI/special features
)

// Background Colors - Dark backgrounds for the cyberpunk aesthetic
var (
	VoidPurple color.Color = lipgloss.Color("#0D0221") // Primary background
	DeepSpace  color.Color = lipgloss.Color("#1A0A2E") // Panel backgrounds
	Twilight   color.Color = lipgloss.Color("#2D1B4E") // Active panel background
	Abyss      color.Color = lipgloss.Color("#0A0A14") // Mini buffer background
)

// Text Colors - Text hierarchy from bright to dim
var (
	PureWhite     color.Color = lipgloss.Color("#FFFFFF") // Primary text
	Silver        color.Color = lipgloss.Color("#E0E0E0") // Secondary text
	MutedLavender color.Color = lipgloss.Color("#888899") // Disabled/comments
	DimPurple     color.Color = lipgloss.Color("#4A4A6A") // Line numbers
)

// Diff Background Colors - Subtle tints for diff highlighting
var (
	BgDiffAdded   color.Color = lipgloss.Color("#0D2818") // Dark green tint
	BgDiffRemoved color.Color = lipgloss.Color("#2D0A0A") // Dark red tint
	BgDiffHunk    color.Color = lipgloss.Color("#1A1A3E") // Hunk header background
)

// Selection Colors - Text selection highlighting
var (
	BgSelection   = lipgloss.Color("#3D2D5E") // Selection background
	TextSelection = lipgloss.Color("#FFFFFF") // Selected text color
)

// Semantic Color Aliases - Use these in components for consistency
var (
	ColorPrimary   color.Color = MagentaBlaze
	ColorSecondary color.Color = CyberCyan
	ColorFocus     color.Color = HotPink
	ColorSuccess   color.Color = MatrixGreen
	ColorError     color.Color = NeonRed
	ColorWarning   color.Color = ElectricYellow
	ColorAI        color.Color = LaserPurple

	BgPrimary     color.Color = VoidPurple
	BgPanel       color.Color = DeepSpace
	BgPanelActive color.Color = Twilight
	BgMiniBuffer  color.Color = Abyss

	TextPrimary   color.Color = PureWhite
	TextSecondary color.Color = Silver
	TextMuted     color.Color = MutedLavender
	TextDim       color.Color = DimPurple
)
