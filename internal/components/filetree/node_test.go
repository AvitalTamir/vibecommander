package filetree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNode(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main"), 0644))

	t.Run("creates node for directory", func(t *testing.T) {
		node, err := NewNode(tmpDir, nil)
		require.NoError(t, err)

		assert.True(t, node.IsDir)
		assert.Equal(t, filepath.Base(tmpDir), node.Name)
		assert.Equal(t, tmpDir, node.Path)
		assert.Equal(t, 0, node.Depth)
		assert.Nil(t, node.Parent)
	})

	t.Run("creates node for file", func(t *testing.T) {
		parent := &Node{Path: tmpDir, Depth: 0}
		node, err := NewNode(testFile, parent)
		require.NoError(t, err)

		assert.False(t, node.IsDir)
		assert.Equal(t, "test.go", node.Name)
		assert.Equal(t, 1, node.Depth)
		assert.Equal(t, parent, node.Parent)
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		_, err := NewNode("/non/existent/path", nil)
		assert.Error(t, err)
	})
}

func TestNewRootNode(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates expanded root node", func(t *testing.T) {
		node, err := NewRootNode(tmpDir)
		require.NoError(t, err)

		assert.True(t, node.IsDir)
		assert.True(t, node.Expanded)
		assert.Equal(t, 0, node.Depth)
	})

	t.Run("resolves relative paths", func(t *testing.T) {
		// Create a relative path
		cwd, _ := os.Getwd()
		relPath := "."

		node, err := NewRootNode(relPath)
		require.NoError(t, err)

		assert.Equal(t, cwd, node.Path)
	})
}

func TestLoadChildren(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte(""), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte(""), 0644))

	t.Run("loads children sorted correctly", func(t *testing.T) {
		node, err := NewRootNode(tmpDir)
		require.NoError(t, err)

		err = node.LoadChildren()
		require.NoError(t, err)

		assert.True(t, node.Loaded)
		assert.Len(t, node.Children, 4) // subdir, .hidden, file1.txt, file2.go

		// First should be directory (subdir comes before .hidden alphabetically? No, . comes first)
		// Actually: directories first, then alphabetical
		// So: .hidden (hidden dir? no, it's a file), subdir, then files
		// Wait - subdir is only dir. So: subdir, then .hidden, file1.txt, file2.go
		assert.True(t, node.Children[0].IsDir, "first should be directory")
		assert.Equal(t, "subdir", node.Children[0].Name)
	})

	t.Run("sets correct depth for children", func(t *testing.T) {
		node, _ := NewRootNode(tmpDir)
		node.LoadChildren()

		for _, child := range node.Children {
			assert.Equal(t, 1, child.Depth)
			assert.Equal(t, node, child.Parent)
		}
	})

	t.Run("does nothing for files", func(t *testing.T) {
		fileNode := &Node{Path: filepath.Join(tmpDir, "file1.txt"), IsDir: false}
		err := fileNode.LoadChildren()

		assert.NoError(t, err)
		assert.Nil(t, fileNode.Children)
	})
}

func TestNodeProperties(t *testing.T) {
	t.Run("IsHidden", func(t *testing.T) {
		hidden := &Node{Name: ".gitignore"}
		notHidden := &Node{Name: "main.go"}

		assert.True(t, hidden.IsHidden())
		assert.False(t, notHidden.IsHidden())
	})

	t.Run("Extension", func(t *testing.T) {
		goFile := &Node{Name: "main.go", IsDir: false}
		txtFile := &Node{Name: "README.TXT", IsDir: false}
		noExt := &Node{Name: "Makefile", IsDir: false}
		dir := &Node{Name: "src", IsDir: true}

		assert.Equal(t, ".go", goFile.Extension())
		assert.Equal(t, ".txt", txtFile.Extension()) // lowercase
		assert.Equal(t, "", noExt.Extension())
		assert.Equal(t, "", dir.Extension())
	})

	t.Run("IsLastChild", func(t *testing.T) {
		parent := &Node{Name: "parent"}
		child1 := &Node{Name: "child1", Parent: parent}
		child2 := &Node{Name: "child2", Parent: parent}
		parent.Children = []*Node{child1, child2}

		assert.False(t, child1.IsLastChild())
		assert.True(t, child2.IsLastChild())
	})
}

func TestToggleExpandCollapse(t *testing.T) {
	dir := &Node{Name: "dir", IsDir: true, Expanded: false}
	file := &Node{Name: "file.go", IsDir: false}

	t.Run("Toggle toggles directory expansion", func(t *testing.T) {
		dir.Toggle()
		assert.True(t, dir.Expanded)

		dir.Toggle()
		assert.False(t, dir.Expanded)
	})

	t.Run("Toggle does nothing for files", func(t *testing.T) {
		file.Toggle()
		assert.False(t, file.Expanded)
	})

	t.Run("Collapse collapses directory", func(t *testing.T) {
		dir.Expanded = true
		dir.Collapse()
		assert.False(t, dir.Expanded)
	})
}

func TestExpand(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte(""), 0644))

	t.Run("Expand loads children and expands", func(t *testing.T) {
		node, _ := NewRootNode(tmpDir)
		node.Expanded = false
		node.Loaded = false

		err := node.Expand()
		require.NoError(t, err)

		assert.True(t, node.Expanded)
		assert.True(t, node.Loaded)
		assert.NotEmpty(t, node.Children)
	})

	t.Run("Expand does nothing for files", func(t *testing.T) {
		file := &Node{Name: "file.go", IsDir: false}
		err := file.Expand()

		assert.NoError(t, err)
		assert.False(t, file.Expanded)
	})
}

func TestFlatten(t *testing.T) {
	// Build a test tree manually
	root := &Node{Name: "root", IsDir: true, Expanded: true, Depth: 0}
	dir1 := &Node{Name: "src", IsDir: true, Expanded: true, Parent: root, Depth: 1}
	dir2 := &Node{Name: "test", IsDir: true, Expanded: false, Parent: root, Depth: 1}
	file1 := &Node{Name: "main.go", IsDir: false, Parent: dir1, Depth: 2}
	file2 := &Node{Name: "README.md", IsDir: false, Parent: root, Depth: 1}
	hidden := &Node{Name: ".gitignore", IsDir: false, Parent: root, Depth: 1}

	dir1.Children = []*Node{file1}
	root.Children = []*Node{dir1, dir2, hidden, file2}

	t.Run("flattens visible nodes", func(t *testing.T) {
		flat := root.Flatten(false)

		// Should include: root, dir1 (expanded), file1, dir2 (collapsed, no children), file2
		// Hidden .gitignore should be excluded
		assert.Len(t, flat, 5)
		assert.Equal(t, "root", flat[0].Name)
		assert.Equal(t, "src", flat[1].Name)
		assert.Equal(t, "main.go", flat[2].Name)
		assert.Equal(t, "test", flat[3].Name)
		assert.Equal(t, "README.md", flat[4].Name)
	})

	t.Run("includes hidden when showHidden is true", func(t *testing.T) {
		flat := root.Flatten(true)

		// Should include all nodes including .gitignore
		assert.Len(t, flat, 6)
	})

	t.Run("respects collapsed directories", func(t *testing.T) {
		// Add a child to dir2 (collapsed)
		dir2.Children = []*Node{{Name: "test.go", IsDir: false, Parent: dir2, Depth: 2}}
		dir2.Loaded = true

		flat := root.Flatten(false)

		// test.go should NOT be in the list since dir2 is collapsed
		for _, n := range flat {
			assert.NotEqual(t, "test.go", n.Name)
		}
	})
}

func TestFindByPath(t *testing.T) {
	root := &Node{Path: "/project", IsDir: true}
	child := &Node{Path: "/project/src", IsDir: true, Parent: root}
	grandchild := &Node{Path: "/project/src/main.go", IsDir: false, Parent: child}

	child.Children = []*Node{grandchild}
	root.Children = []*Node{child}

	t.Run("finds root", func(t *testing.T) {
		found := root.FindByPath("/project")
		assert.Equal(t, root, found)
	})

	t.Run("finds nested node", func(t *testing.T) {
		found := root.FindByPath("/project/src/main.go")
		assert.Equal(t, grandchild, found)
	})

	t.Run("returns nil for not found", func(t *testing.T) {
		found := root.FindByPath("/nonexistent")
		assert.Nil(t, found)
	})
}

func TestVisibleChildren(t *testing.T) {
	dir := &Node{Name: "dir", IsDir: true}
	child := &Node{Name: "child", Parent: dir}
	dir.Children = []*Node{child}

	t.Run("returns nil when collapsed", func(t *testing.T) {
		dir.Expanded = false
		assert.Nil(t, dir.VisibleChildren())
	})

	t.Run("returns children when expanded", func(t *testing.T) {
		dir.Expanded = true
		assert.Equal(t, []*Node{child}, dir.VisibleChildren())
	})

	t.Run("returns nil for files", func(t *testing.T) {
		file := &Node{Name: "file", IsDir: false}
		assert.Nil(t, file.VisibleChildren())
	})
}
