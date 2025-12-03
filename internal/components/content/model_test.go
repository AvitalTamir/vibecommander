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

	m = m.Focus()
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

	// With empty content, returns valid scroll percent
	percent := m.ScrollPercent()
	assert.GreaterOrEqual(t, percent, 0.0)
	assert.LessOrEqual(t, percent, 1.0)
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
