package viewer

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New()

	assert.NotNil(t, m.theme)
	assert.Empty(t, m.path)
	assert.Empty(t, m.content)
	assert.False(t, m.ready)
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()

	assert.Nil(t, cmd)
}

func TestSetSize(t *testing.T) {
	m := New()

	t.Run("initializes viewport on first SetSize", func(t *testing.T) {
		m = m.SetSize(80, 24)

		assert.True(t, m.ready)
		w, h := m.Size()
		assert.Equal(t, 80, w)
		assert.Equal(t, 24, h)
	})

	t.Run("resizes viewport on subsequent SetSize", func(t *testing.T) {
		m = m.SetSize(100, 30)

		w, h := m.Size()
		assert.Equal(t, 100, w)
		assert.Equal(t, 30, h)
	})
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
	t.Run("shows placeholder when not ready", func(t *testing.T) {
		m := New()

		view := m.View()

		assert.Contains(t, view, "Select a file")
	})

	t.Run("shows viewport when ready", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		view := m.View()

		// Should render something (the viewport)
		assert.NotEmpty(t, view)
	})
}

func TestLoadFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!\nLine 2\nLine 3"
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	t.Run("loads existing file", func(t *testing.T) {
		cmd := LoadFile(testFile)
		msg := cmd()

		fileMsg, ok := msg.(FileLoadedMsg)
		require.True(t, ok)
		assert.Equal(t, testFile, fileMsg.Path)
		assert.Equal(t, testContent, fileMsg.Content)
		assert.NoError(t, fileMsg.Err)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		cmd := LoadFile("/non/existent/file.txt")
		msg := cmd()

		fileMsg, ok := msg.(FileLoadedMsg)
		require.True(t, ok)
		assert.Error(t, fileMsg.Err)
	})
}

func TestUpdate(t *testing.T) {
	t.Run("handles FileLoadedMsg with content", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		msg := FileLoadedMsg{
			Path:    "/test/file.txt",
			Content: "Test content",
			Err:     nil,
		}

		m, _ = m.Update(msg)

		assert.Equal(t, "/test/file.txt", m.path)
		assert.Equal(t, "Test content", m.content)
		assert.Nil(t, m.err)
	})

	t.Run("handles FileLoadedMsg with error", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)

		testErr := os.ErrNotExist
		msg := FileLoadedMsg{
			Path: "/test/file.txt",
			Err:  testErr,
		}

		m, _ = m.Update(msg)

		assert.Equal(t, testErr, m.err)
		assert.Empty(t, m.content)
	})

	t.Run("handles keyboard when focused", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		m = m.Focus()

		// Simulate page down key (should be handled by viewport)
		msg := tea.KeyMsg{Type: tea.KeyPgDown}
		_, cmd := m.Update(msg)

		// Just verify no panic and command returned
		_ = cmd
	})

	t.Run("ignores keyboard when not focused", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		// Not focused

		msg := tea.KeyMsg{Type: tea.KeyPgDown}
		_, cmd := m.Update(msg)

		// Should still return without error
		_ = cmd
	})

	t.Run("handles mouse events for scrolling", func(t *testing.T) {
		m := New()
		m = m.SetSize(80, 24)
		// Mouse should work even when not focused

		msg := tea.MouseMsg{
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		}
		_, cmd := m.Update(msg)

		// Should handle without error
		_ = cmd
	})
}

func TestSetContent(t *testing.T) {
	m := New()
	m = m.SetSize(80, 24)

	m.SetContent("Direct content")

	assert.Equal(t, "Direct content", m.content)
	assert.Empty(t, m.path)
	assert.Nil(t, m.err)
}

func TestClear(t *testing.T) {
	m := New()
	m = m.SetSize(80, 24)

	// Set some content first
	m.SetContent("Some content")
	assert.NotEmpty(t, m.content)

	// Clear it
	m.Clear()

	assert.Empty(t, m.path)
	assert.Empty(t, m.content)
	assert.Nil(t, m.err)
	assert.False(t, m.ready)
}

func TestScrollPercent(t *testing.T) {
	m := New()
	m = m.SetSize(80, 24)

	// With empty content, viewport returns 1.0 (at "bottom" of nothing)
	percent := m.ScrollPercent()
	assert.GreaterOrEqual(t, percent, 0.0)
	assert.LessOrEqual(t, percent, 1.0)
}

func TestPath(t *testing.T) {
	m := New()
	m = m.SetSize(80, 24)

	assert.Empty(t, m.Path())

	m, _ = m.Update(FileLoadedMsg{
		Path:    "/test/path.txt",
		Content: "content",
	})

	assert.Equal(t, "/test/path.txt", m.Path())
}

func TestContent(t *testing.T) {
	m := New()
	m = m.SetSize(80, 24)

	assert.Empty(t, m.Content())

	m, _ = m.Update(FileLoadedMsg{
		Path:    "/test/path.txt",
		Content: "test content",
	})

	assert.Equal(t, "test content", m.Content())
}
