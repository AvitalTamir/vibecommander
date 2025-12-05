package theme

import (
	"charm.land/lipgloss/v2"
)

// Available themes
var (
	themes       []*Theme
	currentIndex int
)

func init() {
	themes = []*Theme{
		MidnightMiamiTheme(),
		PinaColadaTheme(),
		LobsterBoyTheme(),
		FeralJungleTheme(),
		VampireWeekendTheme(),
	}
	currentIndex = 0
	ApplyTheme(themes[0])
}

// AllThemes returns all available themes.
func AllThemes() []*Theme {
	return themes
}

// CurrentTheme returns the currently active theme.
func CurrentTheme() *Theme {
	return themes[currentIndex]
}

// CurrentThemeIndex returns the index of the current theme.
func CurrentThemeIndex() int {
	return currentIndex
}

// NextTheme cycles to the next theme and applies it.
func NextTheme() *Theme {
	currentIndex = (currentIndex + 1) % len(themes)
	ApplyTheme(themes[currentIndex])
	return themes[currentIndex]
}

// SetThemeIndex sets the current theme by index and applies it.
// Returns false if index is out of bounds.
func SetThemeIndex(index int) bool {
	if index < 0 || index >= len(themes) {
		return false
	}
	currentIndex = index
	ApplyTheme(themes[currentIndex])
	return true
}

// ApplyTheme sets all the global color variables to match the theme.
func ApplyTheme(t *Theme) {
	// Update semantic color aliases
	ColorPrimary = t.Colors.Primary
	ColorSecondary = t.Colors.Secondary
	ColorFocus = t.Colors.Focus
	ColorSuccess = t.Colors.Success
	ColorError = t.Colors.Error
	ColorWarning = t.Colors.Warning
	ColorAI = t.Colors.AI

	BgPrimary = t.Colors.BgPrimary
	BgPanel = t.Colors.BgPanel
	BgPanelActive = t.Colors.BgPanelActive
	BgMiniBuffer = t.Colors.BgMiniBuffer

	TextPrimary = t.Colors.TextPrimary
	TextSecondary = t.Colors.TextSecondary
	TextMuted = t.Colors.TextMuted
	TextDim = t.Colors.TextDim

	// Also update the named colors for backwards compat
	MagentaBlaze = t.Colors.Primary
	CyberCyan = t.Colors.Secondary
	HotPink = t.Colors.Focus
	MatrixGreen = t.Colors.Success
	NeonRed = t.Colors.Error
	ElectricYellow = t.Colors.Warning
	LaserPurple = t.Colors.AI

	VoidPurple = t.Colors.BgPrimary
	DeepSpace = t.Colors.BgPanel
	Twilight = t.Colors.BgPanelActive
	Abyss = t.Colors.BgMiniBuffer

	PureWhite = t.Colors.TextPrimary
	Silver = t.Colors.TextSecondary
	MutedLavender = t.Colors.TextMuted
	DimPurple = t.Colors.TextDim

	// Regenerate styles
	regenerateStyles()
}

// MidnightMiamiTheme - Neon pink and cyan on deep purple
func MidnightMiamiTheme() *Theme {
	return &Theme{
		Name:         "Midnight Miami",
		UseNerdFonts: true,
		Colors: ColorPalette{
			Primary:       lipgloss.Color("#FF00FF"),
			Secondary:     lipgloss.Color("#00FFFF"),
			Focus:         lipgloss.Color("#FF10F0"),
			Success:       lipgloss.Color("#39FF14"),
			Error:         lipgloss.Color("#FF3131"),
			Warning:       lipgloss.Color("#FFFF00"),
			AI:            lipgloss.Color("#7B68EE"),
			BgPrimary:     lipgloss.Color("#0D0221"),
			BgPanel:       lipgloss.Color("#1A0A2E"),
			BgPanelActive: lipgloss.Color("#2D1B4E"),
			BgMiniBuffer:  lipgloss.Color("#0A0A14"),
			TextPrimary:   lipgloss.Color("#FFFFFF"),
			TextSecondary: lipgloss.Color("#E0E0E0"),
			TextMuted:     lipgloss.Color("#888899"),
			TextDim:       lipgloss.Color("#4A4A6A"),
		},
	}
}

// PinaColadaTheme - Tropical sunset vibes
func PinaColadaTheme() *Theme {
	return &Theme{
		Name:         "Piña Colada",
		UseNerdFonts: true,
		Colors: ColorPalette{
			Primary:       lipgloss.Color("#FFD700"), // Golden pineapple
			Secondary:     lipgloss.Color("#FF6B35"), // Sunset orange
			Focus:         lipgloss.Color("#F7931E"), // Mango
			Success:       lipgloss.Color("#7CB518"), // Palm leaf
			Error:         lipgloss.Color("#D62828"), // Cherry
			Warning:       lipgloss.Color("#FCBF49"), // Banana
			AI:            lipgloss.Color("#48CAE4"), // Ocean blue
			BgPrimary:     lipgloss.Color("#1A0F0A"), // Dark coconut
			BgPanel:       lipgloss.Color("#2D1810"), // Tiki wood
			BgPanelActive: lipgloss.Color("#3D2518"), // Darker tiki
			BgMiniBuffer:  lipgloss.Color("#0F0805"), // Night beach
			TextPrimary:   lipgloss.Color("#FFF8E7"), // Coconut cream
			TextSecondary: lipgloss.Color("#E8D5B7"), // Sand
			TextMuted:     lipgloss.Color("#9E8B76"), // Driftwood
			TextDim:       lipgloss.Color("#5C4A3D"), // Wet sand
		},
	}
}

// LobsterBoyTheme - Fresh from the seafood shack
func LobsterBoyTheme() *Theme {
	return &Theme{
		Name:         "Lobster Boy",
		UseNerdFonts: true,
		Colors: ColorPalette{
			Primary:       lipgloss.Color("#E63946"), // Cooked lobster
			Secondary:     lipgloss.Color("#5CC8E4"), // Bright ocean
			Focus:         lipgloss.Color("#F4A261"), // Melted butter
			Success:       lipgloss.Color("#2A9D8F"), // Seaweed
			Error:         lipgloss.Color("#9B2226"), // Old bay stain
			Warning:       lipgloss.Color("#E9C46A"), // Lemon wedge
			AI:            lipgloss.Color("#7EC8E3"), // Seafoam
			BgPrimary:     lipgloss.Color("#0A1628"), // Midnight ocean
			BgPanel:       lipgloss.Color("#132238"), // Deep sea
			BgPanelActive: lipgloss.Color("#1D3048"), // Wave crest
			BgMiniBuffer:  lipgloss.Color("#050D18"), // Abyss
			TextPrimary:   lipgloss.Color("#F1FAEE"), // Sea foam white
			TextSecondary: lipgloss.Color("#A8DADC"), // Pale aqua
			TextMuted:     lipgloss.Color("#6B8E9F"), // Foggy coast
			TextDim:       lipgloss.Color("#3D5A6C"), // Stormy sea
		},
	}
}

// FeralJungleTheme - Deep in the rainforest
func FeralJungleTheme() *Theme {
	return &Theme{
		Name:         "Feral Jungle",
		UseNerdFonts: true,
		Colors: ColorPalette{
			Primary:       lipgloss.Color("#A7C957"), // Fresh leaf
			Secondary:     lipgloss.Color("#F2CC8F"), // Jaguar spots
			Focus:         lipgloss.Color("#E07A5F"), // Exotic flower
			Success:       lipgloss.Color("#81B29A"), // Moss
			Error:         lipgloss.Color("#BC4749"), // Poison dart frog
			Warning:       lipgloss.Color("#F4D35E"), // Toucan beak
			AI:            lipgloss.Color("#EE6C4D"), // Macaw orange
			BgPrimary:     lipgloss.Color("#0B1A0F"), // Forest floor
			BgPanel:       lipgloss.Color("#132A18"), // Dense canopy
			BgPanelActive: lipgloss.Color("#1D3A22"), // Sunlit clearing
			BgMiniBuffer:  lipgloss.Color("#050F08"), // Undergrowth
			TextPrimary:   lipgloss.Color("#E8F5E9"), // Misty morning
			TextSecondary: lipgloss.Color("#B8D4BA"), // Filtered light
			TextMuted:     lipgloss.Color("#7A9E7E"), // Fern shadow
			TextDim:       lipgloss.Color("#4A6B4E"), // Deep shade
		},
	}
}

// VampireWeekendTheme - Gothic but make it indie
func VampireWeekendTheme() *Theme {
	return &Theme{
		Name:         "Vampire Weekend",
		UseNerdFonts: true,
		Colors: ColorPalette{
			Primary:       lipgloss.Color("#8B0000"), // Fresh blood
			Secondary:     lipgloss.Color("#C0C0C0"), // Moonlight silver
			Focus:         lipgloss.Color("#DC143C"), // Crimson kiss
			Success:       lipgloss.Color("#228B22"), // Graveyard moss
			Error:         lipgloss.Color("#FF0000"), // Arterial spray
			Warning:       lipgloss.Color("#FFD700"), // Candlelight
			AI:            lipgloss.Color("#9932CC"), // Dark orchid
			BgPrimary:     lipgloss.Color("#0D0D0D"), // Coffin interior
			BgPanel:       lipgloss.Color("#1A1A1A"), // Castle stone
			BgPanelActive: lipgloss.Color("#2D2D2D"), // Crypt
			BgMiniBuffer:  lipgloss.Color("#080808"), // Eternal night
			TextPrimary:   lipgloss.Color("#F5F5F5"), // Pale complexion
			TextSecondary: lipgloss.Color("#B8B8B8"), // Aged parchment
			TextMuted:     lipgloss.Color("#6E6E6E"), // Dusty tome
			TextDim:       lipgloss.Color("#3D3D3D"), // Shadow
		},
	}
}
