package filetree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Node represents a file or directory in the tree.
type Node struct {
	Path     string
	Name     string
	IsDir    bool
	Children []*Node
	Parent   *Node
	Depth    int
	Loaded   bool  // Whether children have been loaded (for lazy loading)
	Expanded bool  // Whether directory is expanded in view
	Size     int64 // File size in bytes
	ModTime  int64 // Modification time as unix timestamp
}

// NewNode creates a new node from a path.
func NewNode(path string, parent *Node) (*Node, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	depth := 0
	if parent != nil {
		depth = parent.Depth + 1
	}

	return &Node{
		Path:    path,
		Name:    filepath.Base(path),
		IsDir:   info.IsDir(),
		Parent:  parent,
		Depth:   depth,
		Size:    info.Size(),
		ModTime: info.ModTime().Unix(),
	}, nil
}

// NewRootNode creates a root node for a directory.
func NewRootNode(path string) (*Node, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	node, err := NewNode(absPath, nil)
	if err != nil {
		return nil, err
	}

	node.Expanded = true // Root is always expanded
	return node, nil
}

// LoadChildren loads the immediate children of a directory node.
// Returns an error if the node is not a directory or can't be read.
func (n *Node) LoadChildren() error {
	if !n.IsDir {
		return nil
	}

	entries, err := os.ReadDir(n.Path)
	if err != nil {
		return err
	}

	children := make([]*Node, 0, len(entries))
	for _, entry := range entries {
		childPath := filepath.Join(n.Path, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		child := &Node{
			Path:    childPath,
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Parent:  n,
			Depth:   n.Depth + 1,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		}
		children = append(children, child)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(children, func(i, j int) bool {
		if children[i].IsDir != children[j].IsDir {
			return children[i].IsDir // Directories come first
		}
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})

	n.Children = children
	n.Loaded = true
	return nil
}

// IsHidden returns true if the node is a hidden file/directory.
func (n *Node) IsHidden() bool {
	return len(n.Name) > 0 && n.Name[0] == '.'
}

// Extension returns the file extension (empty for directories).
func (n *Node) Extension() string {
	if n.IsDir {
		return ""
	}
	return strings.ToLower(filepath.Ext(n.Name))
}

// Toggle expands or collapses a directory node.
func (n *Node) Toggle() {
	if n.IsDir {
		n.Expanded = !n.Expanded
	}
}

// Expand expands a directory node, loading children if needed.
func (n *Node) Expand() error {
	if !n.IsDir {
		return nil
	}

	if !n.Loaded {
		if err := n.LoadChildren(); err != nil {
			return err
		}
	}

	n.Expanded = true
	return nil
}

// Collapse collapses a directory node.
func (n *Node) Collapse() {
	if n.IsDir {
		n.Expanded = false
	}
}

// VisibleChildren returns children if expanded, nil otherwise.
func (n *Node) VisibleChildren() []*Node {
	if !n.IsDir || !n.Expanded {
		return nil
	}
	return n.Children
}

// Flatten returns a flat list of all visible nodes in the tree.
// This is used for rendering and navigation.
func (n *Node) Flatten(showHidden bool) []*Node {
	var result []*Node
	n.flattenInto(&result, showHidden)
	return result
}

func (n *Node) flattenInto(result *[]*Node, showHidden bool) {
	// Always skip .git directory
	if n.Name == ".git" && n.IsDir && n.Parent != nil {
		return
	}

	if !showHidden && n.IsHidden() && n.Parent != nil {
		return // Skip hidden files (but not root)
	}

	*result = append(*result, n)

	if n.IsDir && n.Expanded && n.Children != nil {
		for _, child := range n.Children {
			child.flattenInto(result, showHidden)
		}
	}
}

// FindByPath finds a node by its path in the tree.
func (n *Node) FindByPath(path string) *Node {
	if n.Path == path {
		return n
	}

	if n.Children != nil {
		for _, child := range n.Children {
			if found := child.FindByPath(path); found != nil {
				return found
			}
		}
	}

	return nil
}

// RelativePath returns the path relative to the root.
func (n *Node) RelativePath(root string) string {
	rel, err := filepath.Rel(root, n.Path)
	if err != nil {
		return n.Path
	}
	return rel
}

// IsLastChild returns true if this node is the last child of its parent.
func (n *Node) IsLastChild() bool {
	if n.Parent == nil || len(n.Parent.Children) == 0 {
		return true
	}
	return n.Parent.Children[len(n.Parent.Children)-1] == n
}
