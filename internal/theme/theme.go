package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds all visual configuration for the application.
type Theme struct {
	// Name of the theme
	Name string

	// Color palette
	Colors ColorPalette

	// Whether to use Nerd Font icons
	UseNerdFonts bool
}

// ColorPalette holds all color definitions.
type ColorPalette struct {
	// Accent colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Focus     lipgloss.Color
	Success   lipgloss.Color
	Error     lipgloss.Color
	Warning   lipgloss.Color
	AI        lipgloss.Color

	// Background colors
	BgPrimary     lipgloss.Color
	BgPanel       lipgloss.Color
	BgPanelActive lipgloss.Color
	BgMiniBuffer  lipgloss.Color

	// Text colors
	TextPrimary   lipgloss.Color
	TextSecondary lipgloss.Color
	TextMuted     lipgloss.Color
	TextDim       lipgloss.Color
}

// DefaultTheme returns the default cyberpunk theme.
func DefaultTheme() *Theme {
	return &Theme{
		Name:         "Cyberpunk",
		UseNerdFonts: true,
		Colors: ColorPalette{
			Primary:       MagentaBlaze,
			Secondary:     CyberCyan,
			Focus:         HotPink,
			Success:       MatrixGreen,
			Error:         NeonRed,
			Warning:       ElectricYellow,
			AI:            LaserPurple,
			BgPrimary:     VoidPurple,
			BgPanel:       DeepSpace,
			BgPanelActive: Twilight,
			BgMiniBuffer:  Abyss,
			TextPrimary:   PureWhite,
			TextSecondary: Silver,
			TextMuted:     MutedLavender,
			TextDim:       DimPurple,
		},
	}
}

// GetFileIcon returns the icon for a file, respecting the UseNerdFonts setting.
func (t *Theme) GetFileIcon(ext string) string {
	if !t.UseNerdFonts {
		return IconFile
	}
	return GetFileIcon(ext)
}

// GetDirIcon returns the icon for a directory, respecting the UseNerdFonts setting.
func (t *Theme) GetDirIcon(name string, expanded bool) string {
	if !t.UseNerdFonts {
		if expanded {
			return IconDirExpanded
		}
		return IconDirCollapsed
	}

	if icon := GetDirIcon(name); icon != "" {
		return icon
	}

	if expanded {
		return IconDirExpanded
	}
	return IconDirCollapsed
}
