package filetree

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelNew(t *testing.T) {
	m := New()

	assert.NotNil(t, m.loading)
	assert.NotNil(t, m.theme)
	// New() now initializes root to current working directory
	assert.NotNil(t, m.root)
}

func TestNewWithPath(t *testing.T) {
	tmpDir := t.TempDir()

	m, err := NewWithPath(tmpDir)
	require.NoError(t, err)

	assert.NotNil(t, m.root)
	assert.Equal(t, tmpDir, m.root.Path)
}

func TestModelInit(t *testing.T) {
	t.Run("creates root from current directory when nil", func(t *testing.T) {
		m := New()
		cmd := m.Init()

		// Should return a command to load children
		assert.NotNil(t, cmd)
	})

	t.Run("loads children when root exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		m, err := NewWithPath(tmpDir)
		require.NoError(t, err)

		cmd := m.Init()
		assert.NotNil(t, cmd)
	})
}

func TestModelFocusBlur(t *testing.T) {
	m := New()

	assert.False(t, m.Focused())

	m = m.Focus()
	assert.True(t, m.Focused())

	m = m.Blur()
	assert.False(t, m.Focused())
}

func TestModelSetSize(t *testing.T) {
	m := New()
	m = m.SetSize(30, 40)

	w, h := m.Size()
	assert.Equal(t, 30, w)
	assert.Equal(t, 40, h)
}

func TestModelUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755))

	t.Run("ignores input when not focused", func(t *testing.T) {
		m, _ := NewWithPath(tmpDir)
		m = m.SetSize(30, 40)
		// Not focused

		msg := tea.KeyMsg{Type: tea.KeyDown}
		newM, _ := m.Update(msg)

		// Model should be unchanged
		assert.Equal(t, m.cursor, newM.cursor)
	})

	t.Run("handles navigation keys when focused", func(t *testing.T) {
		m, _ := NewWithPath(tmpDir)
		m = m.SetSize(30, 40)
		m = m.Focus()

		// Load the children first
		msg := LoadedMsg{
			Path: tmpDir,
			Children: []*Node{
				{Name: "subdir", Path: filepath.Join(tmpDir, "subdir"), IsDir: true},
				{Name: "file1.txt", Path: filepath.Join(tmpDir, "file1.txt"), IsDir: false},
			},
		}
		m, _ = m.Update(msg)

		// Now test navigation
		assert.Equal(t, 0, m.cursor)

		// Move down
		downMsg := tea.KeyMsg{Type: tea.KeyDown}
		m, _ = m.Update(downMsg)
		assert.Equal(t, 1, m.cursor)

		// Move up
		upMsg := tea.KeyMsg{Type: tea.KeyUp}
		m, _ = m.Update(upMsg)
		assert.Equal(t, 0, m.cursor)
	})

	t.Run("handles LoadedMsg", func(t *testing.T) {
		m, _ := NewWithPath(tmpDir)
		m = m.SetSize(30, 40)
		m = m.Focus()

		children := []*Node{
			{Name: "test.go", Path: filepath.Join(tmpDir, "test.go"), IsDir: false},
		}

		msg := LoadedMsg{
			Path:     tmpDir,
			Children: children,
		}

		m, _ = m.Update(msg)

		assert.True(t, m.root.Loaded)
		assert.Len(t, m.root.Children, 1)
	})
}

func TestModelView(t *testing.T) {
	t.Run("returns empty for zero dimensions", func(t *testing.T) {
		m := New()
		view := m.View()
		assert.Empty(t, view)
	})

	t.Run("renders file tree content", func(t *testing.T) {
		tmpDir := t.TempDir()
		m, _ := NewWithPath(tmpDir)
		m = m.SetSize(30, 40)
		m = m.Focus()

		// Add some visible nodes
		m.root.Loaded = true
		m.root.Children = []*Node{
			{Name: "test.go", Path: filepath.Join(tmpDir, "test.go"), IsDir: false, Depth: 1, Parent: m.root},
		}
		m.rebuildVisible()

		view := m.View()
		assert.NotEmpty(t, view)
	})
}

func TestModelSelectMsg(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("content"), 0644))

	m, _ := NewWithPath(tmpDir)
	m = m.SetSize(30, 40)
	m = m.Focus()

	// Load children
	m.root.Loaded = true
	m.root.Children = []*Node{
		{Name: "test.txt", Path: testFile, IsDir: false, Depth: 1, Parent: m.root},
	}
	m.rebuildVisible()

	// Move to the file and select it
	m.cursor = 1 // Skip root

	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(enterMsg)

	// Should return a command that produces SelectMsg
	assert.NotNil(t, cmd)
	msg := cmd()
	selectMsg, ok := msg.(SelectMsg)
	assert.True(t, ok)
	assert.Equal(t, testFile, selectMsg.Path)
	assert.False(t, selectMsg.IsDir)
}

func TestModelDirectoryToggle(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	m, _ := NewWithPath(tmpDir)
	m = m.SetSize(30, 40)
	m = m.Focus()

	// Load children
	m.root.Loaded = true
	m.root.Children = []*Node{
		{Name: "subdir", Path: subDir, IsDir: true, Depth: 1, Parent: m.root},
	}
	m.rebuildVisible()

	// visible[0] is root (expanded), visible[1] is subdir
	require.Len(t, m.visible, 2)
	assert.Equal(t, "subdir", m.visible[1].Name)
	assert.False(t, m.visible[1].Expanded) // Initially not expanded

	// Move to subdir
	m.cursor = 1

	// Use Enter key to expand directory (more reliable than space key)
	enterMsg := tea.KeyMsg{Type: tea.KeyEnter}
	m, cmd := m.Update(enterMsg)

	// Directory should be expanded (and trigger load since not loaded)
	assert.True(t, m.visible[1].Expanded)
	assert.NotNil(t, cmd) // Should trigger load for unloaded directory
}

func TestModelMouseHandling(t *testing.T) {
	tmpDir := t.TempDir()
	m, _ := NewWithPath(tmpDir)
	m = m.SetSize(30, 40)
	m = m.Focus()

	// Load children
	m.root.Loaded = true
	m.root.Children = []*Node{
		{Name: "file1.txt", Path: filepath.Join(tmpDir, "file1.txt"), IsDir: false, Depth: 1, Parent: m.root},
		{Name: "file2.txt", Path: filepath.Join(tmpDir, "file2.txt"), IsDir: false, Depth: 1, Parent: m.root},
	}
	m.rebuildVisible()

	t.Run("handles wheel up", func(t *testing.T) {
		m.cursor = 2
		msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
		m, _ = m.Update(msg)
		assert.Less(t, m.cursor, 2)
	})

	t.Run("handles wheel down", func(t *testing.T) {
		m.cursor = 0
		msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
		m, _ = m.Update(msg)
		assert.Greater(t, m.cursor, 0)
	})
}

func TestModelRoot(t *testing.T) {
	t.Run("returns current dir for default New()", func(t *testing.T) {
		m := New()
		cwd, _ := os.Getwd()
		assert.Equal(t, cwd, m.Root())
	})

	t.Run("returns root path for NewWithPath", func(t *testing.T) {
		tmpDir := t.TempDir()
		m, _ := NewWithPath(tmpDir)
		assert.Equal(t, tmpDir, m.Root())
	})
}

func TestModelSelectedPath(t *testing.T) {
	tmpDir := t.TempDir()
	m, _ := NewWithPath(tmpDir)
	m = m.SetSize(30, 40)

	// Load children
	testFile := filepath.Join(tmpDir, "test.go")
	m.root.Loaded = true
	m.root.Children = []*Node{
		{Name: "test.go", Path: testFile, IsDir: false, Depth: 1, Parent: m.root},
	}
	m.rebuildVisible()

	// Select the file
	m.cursor = 1

	assert.Equal(t, testFile, m.SelectedPath())
}

func TestModelSelectedNode(t *testing.T) {
	tmpDir := t.TempDir()
	m, _ := NewWithPath(tmpDir)
	m = m.SetSize(30, 40)

	// Load children
	fileNode := &Node{Name: "test.go", Path: filepath.Join(tmpDir, "test.go"), IsDir: false, Depth: 1}
	m.root.Loaded = true
	m.root.Children = []*Node{fileNode}
	fileNode.Parent = m.root
	m.rebuildVisible()

	// Select the file
	m.cursor = 1

	selected := m.SelectedNode()
	assert.NotNil(t, selected)
	assert.Equal(t, "test.go", selected.Name)
}

func TestModelSetShowHidden(t *testing.T) {
	tmpDir := t.TempDir()
	m, _ := NewWithPath(tmpDir)
	m = m.SetSize(30, 40)

	// Load children including hidden file
	m.root.Loaded = true
	m.root.Children = []*Node{
		{Name: ".hidden", Path: filepath.Join(tmpDir, ".hidden"), IsDir: false, Depth: 1, Parent: m.root},
		{Name: "visible.txt", Path: filepath.Join(tmpDir, "visible.txt"), IsDir: false, Depth: 1, Parent: m.root},
	}
	m.rebuildVisible()

	// Hidden files should be shown by default
	assert.True(t, m.ShowHidden())
	hiddenCount := 0
	for _, n := range m.visible {
		if n.IsHidden() && n.Parent != nil {
			hiddenCount++
		}
	}
	assert.Equal(t, 1, hiddenCount)

	// Hide hidden files
	m.SetShowHidden(false)
	assert.False(t, m.ShowHidden())

	hiddenCount = 0
	for _, n := range m.visible {
		if n.IsHidden() && n.Parent != nil {
			hiddenCount++
		}
	}
	assert.Equal(t, 0, hiddenCount)
}

func TestModelSetRoot(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	m, _ := NewWithPath(tmpDir1)
	assert.Equal(t, tmpDir1, m.Root())

	err := m.SetRoot(tmpDir2)
	require.NoError(t, err)
	assert.Equal(t, tmpDir2, m.Root())
	assert.Equal(t, 0, m.cursor)
	assert.Equal(t, 0, m.offset)
}
