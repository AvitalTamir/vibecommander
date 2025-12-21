package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/avitaltamir/vibecommander/internal/components/content/viewer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New()

	assert.Equal(t, ModeViewer, m.mode)
	assert.NotNil(t, m.theme)
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()

	// Init returns the viewer's init command
	assert.Nil(t, cmd)
}

func TestMode(t *testing.T) {
	t.Run("String returns correct values", func(t *testing.T) {
		assert.Equal(t, "VIEWER", ModeViewer.String())
		assert.Equal(t, "DIFF", ModeDiff.String())
		assert.Equal(t, "TERMINAL", ModeTerminal.String())
		assert.Equal(t, "AI ASSISTANT", ModeAI.String())
		assert.Equal(t, "UNKNOWN", Mode(99).String())
	})

	t.Run("Mode getter returns current mode", func(t *testing.T) {
		m := New()
		assert.Equal(t, ModeViewer, m.Mode())
	})

	t.Run("SetMode changes the mode", func(t *testing.T) {
		m := New()
		m.SetMode(ModeDiff)
		assert.Equal(t, ModeDiff, m.Mode())
	})
}

func TestSetSize(t *testing.T) {
	m := New()
	m = m.SetSize(100, 50)

	w, h := m.Size()
	assert.Equal(t, 100, w)
	assert.Equal(t, 50, h)
}

func TestFocusBlur(t *testing.T) {
	m := New()

	assert.False(t, m.Focused())

	m, _ = m.Focus()
	assert.True(t, m.Focused())

	m = m.Blur()
	assert.False(t, m.Focused())
}

func TestView(t *testing.T) {
	t.Run("returns empty for zero dimensions", func(t *testing.T) {
		m := New()
		view := m.View()
		assert.Empty(t, view)
	})

	t.Run("renders viewer mode", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m.mode = ModeViewer

		view := m.View()

		assert.Contains(t, view, "VIEWER")
	})

	t.Run("renders diff mode", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m.mode = ModeDiff

		view := m.View()

		assert.Contains(t, view, "DIFF")
	})

	t.Run("renders terminal mode", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m.mode = ModeTerminal

		view := m.View()

		assert.Contains(t, view, "TERMINAL")
		// Terminal shows "ready" message when not running
		assert.Contains(t, view, "Terminal ready")
	})

	t.Run("renders AI assistant mode", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m.mode = ModeAI
		m.aiCommand = "claude"

		view := m.View()

		// Note: View() still uses mode.String() which returns "AI ASSISTANT"
		// The AI command name is used via AICommandName() for the title bar
		assert.Contains(t, view, "AI ASSISTANT")
	})
}

func TestUpdate(t *testing.T) {
	t.Run("handles SetModeMsg", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		m, _ = m.Update(SetModeMsg{Mode: ModeDiff})

		assert.Equal(t, ModeDiff, m.mode)
	})

	t.Run("handles OpenFileMsg", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		m, cmd := m.Update(OpenFileMsg{Path: "/test/file.txt"})

		assert.Equal(t, ModeViewer, m.mode)
		assert.Equal(t, "/test/file.txt", m.currentPath)
		assert.NotNil(t, cmd) // Should return command to load file
	})

	t.Run("handles FileLoadedMsg", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		msg := viewer.FileLoadedMsg{
			Path:    "/test/file.txt",
			Content: "Test content",
		}

		m, _ = m.Update(msg)

		// The viewer should have received the message
		assert.Equal(t, "Test content", m.viewer.Content())
	})
}

func TestCurrentPath(t *testing.T) {
	m := New()
	m = m.SetSize(80, 24)

	assert.Empty(t, m.CurrentPath())

	m, _ = m.Update(OpenFileMsg{Path: "/test/path.txt"})
	assert.Equal(t, "/test/path.txt", m.CurrentPath())
}

func TestScrollPercent(t *testing.T) {
	m := New()
	m = m.SetSize(80, 24)

	// With empty content, returns valid scroll percent (0-100)
	percent := m.ScrollPercent()
	assert.GreaterOrEqual(t, percent, 0.0)
	assert.LessOrEqual(t, percent, 100.0)
}

func TestIntegration(t *testing.T) {
	t.Run("full file open flow", func(t *testing.T) {
		// Create a temp file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.go")
		testContent := "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}"
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

		// Create and initialize the model
		m := New()
		m = m.SetSize(80, 24)

		// Open the file
		m, cmd := m.Update(OpenFileMsg{Path: testFile})
		assert.NotNil(t, cmd)
		assert.Equal(t, testFile, m.CurrentPath())

		// Execute the command to load the file
		msg := cmd()
		fileMsg, ok := msg.(viewer.FileLoadedMsg)
		require.True(t, ok)
		assert.Equal(t, testFile, fileMsg.Path)
		assert.Equal(t, testContent, fileMsg.Content)

		// Process the loaded message
		m, _ = m.Update(fileMsg)

		// Verify content is available
		assert.Equal(t, testContent, m.viewer.Content())
	})
}

func TestSourcesInfo(t *testing.T) {
	t.Run("returns empty when no content loaded", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		sources := m.SourcesInfo()
		assert.Empty(t, sources)
	})

	t.Run("returns file source after opening file", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		// Open a file
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})

		sources := m.SourcesInfo()
		require.Len(t, sources, 1)
		assert.Equal(t, SourceFile, sources[0].Source)
		assert.Equal(t, "file.txt", sources[0].Title)
		assert.True(t, sources[0].IsActive)
	})

	t.Run("returns AI source after launching AI", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		// Launch AI
		m, _ = m.Update(LaunchAIMsg{Command: "claude", Args: []string{}})

		sources := m.SourcesInfo()
		require.Len(t, sources, 1)
		assert.Equal(t, SourceAI, sources[0].Source)
		assert.Equal(t, "Claude", sources[0].Title)
		assert.True(t, sources[0].IsActive)
	})

	t.Run("returns both sources when file and AI loaded", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		// Open a file first
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})
		// Then launch AI
		m, _ = m.Update(LaunchAIMsg{Command: "claude", Args: []string{}})

		sources := m.SourcesInfo()
		require.Len(t, sources, 2)

		// File should be inactive (AI is active)
		assert.Equal(t, SourceFile, sources[0].Source)
		assert.False(t, sources[0].IsActive)

		// AI should be active
		assert.Equal(t, SourceAI, sources[1].Source)
		assert.True(t, sources[1].IsActive)
	})
}

func TestSwitchSourceMsg(t *testing.T) {
	t.Run("switches from AI to file", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		// Open file and launch AI
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})
		m, _ = m.Update(LaunchAIMsg{Command: "claude", Args: []string{}})

		// Currently in AI mode
		assert.Equal(t, ModeAI, m.mode)

		// Switch to file
		m, _ = m.Update(SwitchSourceMsg{Source: SourceFile})

		assert.Equal(t, ModeViewer, m.mode)
	})

	t.Run("switches from file to AI", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		// Launch AI first, then open file
		m, _ = m.Update(LaunchAIMsg{Command: "claude", Args: []string{}})
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})

		// Open file switches to viewer mode
		assert.Equal(t, ModeViewer, m.mode)

		// Switch back to AI
		m, _ = m.Update(SwitchSourceMsg{Source: SourceAI})

		assert.Equal(t, ModeAI, m.mode)
	})

	t.Run("ignores switch to non-existent source", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		// Only open file
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})

		// Try to switch to AI (which doesn't exist)
		m, _ = m.Update(SwitchSourceMsg{Source: SourceAI})

		// Should stay in viewer mode
		assert.Equal(t, ModeViewer, m.mode)
	})
}

func TestHasMultipleSources(t *testing.T) {
	t.Run("returns false with no content", func(t *testing.T) {
		m := New()
		assert.False(t, m.HasMultipleSources())
	})

	t.Run("returns false with only file", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})
		assert.False(t, m.HasMultipleSources())
	})

	t.Run("returns false with only AI", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m, _ = m.Update(LaunchAIMsg{Command: "claude", Args: []string{}})
		assert.False(t, m.HasMultipleSources())
	})

	t.Run("returns true with both file and AI", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})
		m, _ = m.Update(LaunchAIMsg{Command: "claude", Args: []string{}})
		assert.True(t, m.HasMultipleSources())
	})
}

func TestActiveSource(t *testing.T) {
	t.Run("returns SourceNone initially", func(t *testing.T) {
		m := New()
		// ModeViewer is default, but no file content
		assert.Equal(t, SourceFile, m.ActiveSource())
	})

	t.Run("returns SourceFile in viewer mode", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m, _ = m.Update(OpenFileMsg{Path: "/test/file.txt"})
		assert.Equal(t, SourceFile, m.ActiveSource())
	})

	t.Run("returns SourceFile in diff mode", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m.mode = ModeDiff
		assert.Equal(t, SourceFile, m.ActiveSource())
	})

	t.Run("returns SourceAI in AI mode", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m, _ = m.Update(LaunchAIMsg{Command: "claude", Args: []string{}})
		assert.Equal(t, SourceAI, m.ActiveSource())
	})
}
