package theme

import "github.com/charmbracelet/lipgloss"

// Neon Core Colors - Primary accent colors for the cyberpunk theme
var (
	MagentaBlaze   = lipgloss.Color("#FF00FF") // Primary accent
	CyberCyan      = lipgloss.Color("#00FFFF") // Secondary accent
	HotPink        = lipgloss.Color("#FF10F0") // Selections/Focus
	MatrixGreen    = lipgloss.Color("#39FF14") // Git additions/success
	NeonRed        = lipgloss.Color("#FF3131") // Git deletions/errors
	ElectricYellow = lipgloss.Color("#FFFF00") // Warnings/modified
	LaserPurple    = lipgloss.Color("#7B68EE") // AI/special features
)

// Background Colors - Dark backgrounds for the cyberpunk aesthetic
var (
	VoidPurple = lipgloss.Color("#0D0221") // Primary background
	DeepSpace  = lipgloss.Color("#1A0A2E") // Panel backgrounds
	Twilight   = lipgloss.Color("#2D1B4E") // Active panel background
	Abyss      = lipgloss.Color("#0A0A14") // Mini buffer background
)

// Text Colors - Text hierarchy from bright to dim
var (
	PureWhite     = lipgloss.Color("#FFFFFF") // Primary text
	Silver        = lipgloss.Color("#E0E0E0") // Secondary text
	MutedLavender = lipgloss.Color("#888899") // Disabled/comments
	DimPurple     = lipgloss.Color("#4A4A6A") // Line numbers
)

// Diff Background Colors - Subtle tints for diff highlighting
var (
	BgDiffAdded   = lipgloss.Color("#0D2818") // Dark green tint
	BgDiffRemoved = lipgloss.Color("#2D0A0A") // Dark red tint
	BgDiffHunk    = lipgloss.Color("#1A1A3E") // Hunk header background
)

// Selection Colors - Text selection highlighting
var (
	BgSelection   = lipgloss.Color("#3D2D5E") // Selection background
	TextSelection = lipgloss.Color("#FFFFFF") // Selected text color
)

// Semantic Color Aliases - Use these in components for consistency
var (
	ColorPrimary   = MagentaBlaze
	ColorSecondary = CyberCyan
	ColorFocus     = HotPink
	ColorSuccess   = MatrixGreen
	ColorError     = NeonRed
	ColorWarning   = ElectricYellow
	ColorAI        = LaserPurple

	BgPrimary     = VoidPurple
	BgPanel       = DeepSpace
	BgPanelActive = Twilight
	BgMiniBuffer  = Abyss

	TextPrimary   = PureWhite
	TextSecondary = Silver
	TextMuted     = MutedLavender
	TextDim       = DimPurple
)
