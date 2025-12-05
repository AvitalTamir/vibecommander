package theme

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	assert.NotNil(t, theme)
	assert.Equal(t, "Cyberpunk", theme.Name)
	assert.True(t, theme.UseNerdFonts)

	// Verify colors are set
	assert.NotEmpty(t, theme.Colors.Primary)
	assert.NotEmpty(t, theme.Colors.Secondary)
	assert.NotEmpty(t, theme.Colors.Success)
	assert.NotEmpty(t, theme.Colors.Error)
}

func TestGetFileIcon(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".go", "󰟓"},
		{".md", "󰍔"},
		{".json", ""},
		{".unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			icon := GetFileIcon(tt.ext)
			assert.Equal(t, tt.expected, icon)
		})
	}
}

func TestGetDirIcon(t *testing.T) {
	t.Run("known directories return non-empty icons", func(t *testing.T) {
		knownDirs := []string{".git", "node_modules", "src", "cmd"}
		for _, dir := range knownDirs {
			icon := GetDirIcon(dir)
			assert.NotEmpty(t, icon, "expected icon for %s", dir)
		}
	})

	t.Run("unknown directories return empty string", func(t *testing.T) {
		icon := GetDirIcon("random")
		assert.Empty(t, icon)
	})
}

func TestThemeGetFileIcon(t *testing.T) {
	t.Run("with nerd fonts enabled", func(t *testing.T) {
		theme := DefaultTheme()
		theme.UseNerdFonts = true

		icon := theme.GetFileIcon(".go")
		assert.Equal(t, "󰟓", icon)
	})

	t.Run("with nerd fonts disabled", func(t *testing.T) {
		theme := DefaultTheme()
		theme.UseNerdFonts = false

		icon := theme.GetFileIcon(".go")
		assert.Equal(t, IconFile, icon)
	})
}

func TestThemeGetDirIcon(t *testing.T) {
	theme := DefaultTheme()

	t.Run("expanded directory with nerd fonts", func(t *testing.T) {
		theme.UseNerdFonts = true
		icon := theme.GetDirIcon("random", true)
		assert.Equal(t, IconDirExpanded, icon)
	})

	t.Run("collapsed directory with nerd fonts", func(t *testing.T) {
		theme.UseNerdFonts = true
		icon := theme.GetDirIcon("random", false)
		assert.Equal(t, IconDirCollapsed, icon)
	})

	t.Run("special directory returns its icon", func(t *testing.T) {
		theme.UseNerdFonts = true
		// GetDirIcon returns the dir icon from DirIcons map if found
		icon := theme.GetDirIcon(".git", false)
		// .git has a special icon in DirIcons, should not be empty
		assert.NotEmpty(t, icon)
		assert.NotEqual(t, IconDirCollapsed, icon)
		assert.NotEqual(t, IconDirExpanded, icon)
	})

	t.Run("without nerd fonts uses basic icons", func(t *testing.T) {
		theme.UseNerdFonts = false
		icon := theme.GetDirIcon(".git", true)
		assert.Equal(t, IconDirExpanded, icon)
	})
}

func TestRenderTitle(t *testing.T) {
	t.Run("focused title", func(t *testing.T) {
		title := RenderTitle("FILES", true)
		assert.Contains(t, title, "FILES")
		assert.Contains(t, title, PanelDiamond)
	})

	t.Run("unfocused title", func(t *testing.T) {
		title := RenderTitle("CONTENT", false)
		assert.Contains(t, title, "CONTENT")
		assert.Contains(t, title, PanelDiamond)
	})
}

func TestGetPanelStyle(t *testing.T) {
	t.Run("focused returns PanelFocused", func(t *testing.T) {
		style := GetPanelStyle(true)
		// Just verify it returns a style without panicking
		_ = style.Render("test")
	})

	t.Run("unfocused returns PanelInactive", func(t *testing.T) {
		style := GetPanelStyle(false)
		_ = style.Render("test")
	})
}

func TestGetGitStatusStyle(t *testing.T) {
	tests := []struct {
		code rune
		name string
	}{
		{'M', "modified"},
		{'A', "added"},
		{'D', "deleted"},
		{'?', "untracked"},
		{'U', "unmerged"},
		{' ', "unmodified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := GetGitStatusStyle(tt.code)
			// Verify style can render without panic
			_ = style.Render("test")
		})
	}
}

func TestSetThemeIndex(t *testing.T) {
	// Save original index to restore after test
	originalIdx := CurrentThemeIndex()
	defer SetThemeIndex(originalIdx)

	t.Run("valid index sets theme", func(t *testing.T) {
		ok := SetThemeIndex(2)
		assert.True(t, ok)
		assert.Equal(t, 2, CurrentThemeIndex())
	})

	t.Run("negative index returns false", func(t *testing.T) {
		SetThemeIndex(0) // Reset first
		ok := SetThemeIndex(-1)
		assert.False(t, ok)
		assert.Equal(t, 0, CurrentThemeIndex(), "index should not change")
	})

	t.Run("out of bounds index returns false", func(t *testing.T) {
		SetThemeIndex(0) // Reset first
		ok := SetThemeIndex(100)
		assert.False(t, ok)
		assert.Equal(t, 0, CurrentThemeIndex(), "index should not change")
	})
}

func TestCalculateTitleRegions(t *testing.T) {
	t.Run("single title without status", func(t *testing.T) {
		opts := PanelTitleOptions{
			Title:         "file.txt",
			PrimaryActive: true,
		}

		primary, secondary := CalculateTitleRegions(opts)

		// Primary should start at position 3 (corner + 2 filler)
		assert.Equal(t, 3, primary.StartX)
		// Length: "[ " (2) + "file.txt" (8) + " ]" (2) = 12
		assert.Equal(t, 3+12, primary.EndX)
		assert.True(t, primary.IsActive)
		assert.Equal(t, "file.txt", primary.Title)

		// No secondary
		assert.Empty(t, secondary.Title)
	})

	t.Run("single title with status", func(t *testing.T) {
		opts := PanelTitleOptions{
			Title:         "Claude",
			ShowStatus:    true,
			PrimaryActive: true,
		}

		primary, _ := CalculateTitleRegions(opts)

		// Length: "[ " (2) + "Claude" (6) + " ]" (2) + " ●" (2) = 12
		assert.Equal(t, 3, primary.StartX)
		assert.Equal(t, 3+12, primary.EndX)
	})

	t.Run("dual titles", func(t *testing.T) {
		opts := PanelTitleOptions{
			Title:          "file.txt",
			PrimaryActive:  true,
			SecondaryTitle: "Claude",
			SecondaryActive: false,
			SecondaryShowStatus: true,
		}

		primary, secondary := CalculateTitleRegions(opts)

		// Primary: "[ file.txt ]" = 12 chars
		assert.Equal(t, 3, primary.StartX)
		assert.Equal(t, 15, primary.EndX) // 3 + 12
		assert.True(t, primary.IsActive)

		// Secondary starts after primary + 2 space separator
		// "[ Claude ● ]" = 4 + 6 + 2 = 12 chars
		assert.Equal(t, 17, secondary.StartX) // 15 + 2
		assert.Equal(t, 29, secondary.EndX) // 17 + 12
		assert.False(t, secondary.IsActive)
	})

	t.Run("primary inactive when secondary active", func(t *testing.T) {
		opts := PanelTitleOptions{
			Title:           "file.txt",
			PrimaryActive:   false,
			SecondaryTitle:  "Claude",
			SecondaryActive: true,
		}

		primary, secondary := CalculateTitleRegions(opts)

		assert.False(t, primary.IsActive)
		assert.True(t, secondary.IsActive)
	})
}
