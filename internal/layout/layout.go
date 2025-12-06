package layout

// Layout constants
const (
	DefaultLeftPanelPercent = 25
	MinLeftPanelPercent     = 15
	MaxLeftPanelPercent     = 60
	MiniBufferPercent       = 25
	GitPanelPercent         = 50 // Git panel takes 50% of left panel when visible
	StatusBarHeight         = 1
	MinPanelWidth           = 20
	MinPanelHeight          = 5
)

// Layout holds calculated dimensions for all panels.
type Layout struct {
	// Total terminal dimensions
	TotalWidth  int
	TotalHeight int

	// Panel widths
	LeftWidth  int
	RightWidth int

	// Panel heights
	MainHeight int
	MiniHeight int

	// Left panel split (for git panel)
	FileTreeHeight int
	GitPanelHeight int

	// Status bar
	StatusHeight int

	// Visibility flags
	MiniVisible     bool
	GitPanelVisible bool
}

// Calculate computes the layout dimensions based on terminal size.
// leftPercent controls the width of the left panel (file tree).
// gitPanelVisible controls whether the git panel splits the left panel.
func Calculate(width, height int, miniVisible bool, leftPercent int, gitPanelVisible bool) Layout {
	l := Layout{
		TotalWidth:      width,
		TotalHeight:     height,
		StatusHeight:    StatusBarHeight,
		MiniVisible:     miniVisible,
		GitPanelVisible: gitPanelVisible,
	}

	// Clamp left panel percentage to valid range
	if leftPercent < MinLeftPanelPercent {
		leftPercent = MinLeftPanelPercent
	}
	if leftPercent > MaxLeftPanelPercent {
		leftPercent = MaxLeftPanelPercent
	}

	// Calculate horizontal split
	l.LeftWidth = max(width*leftPercent/100, MinPanelWidth)
	l.RightWidth = max(width-l.LeftWidth, MinPanelWidth)

	// Ensure we don't exceed total width
	if l.LeftWidth+l.RightWidth > width {
		l.RightWidth = width - l.LeftWidth
	}

	// Reserve status bar
	availableHeight := height - l.StatusHeight

	// Calculate vertical split for mini buffer
	if miniVisible {
		l.MiniHeight = max(availableHeight*MiniBufferPercent/100, MinPanelHeight)
		l.MainHeight = max(availableHeight-l.MiniHeight-1, MinPanelHeight)
	} else {
		l.MiniHeight = 0
		l.MainHeight = availableHeight
	}

	// Calculate left panel vertical split for git panel
	if gitPanelVisible {
		l.GitPanelHeight = max(l.MainHeight*GitPanelPercent/100, MinPanelHeight)
		l.FileTreeHeight = max(l.MainHeight-l.GitPanelHeight, MinPanelHeight)
	} else {
		l.FileTreeHeight = l.MainHeight
		l.GitPanelHeight = 0
	}

	return l
}

// ContentWidth returns the inner width for content (excluding borders).
func (l Layout) ContentWidth(panelWidth int, borderWidth int) int {
	return max(panelWidth-borderWidth*2, 0)
}

// ContentHeight returns the inner height for content (excluding borders).
func (l Layout) ContentHeight(panelHeight int, borderHeight int) int {
	return max(panelHeight-borderHeight*2, 0)
}

// LeftPanelBounds returns the position and size of the left panel (file tree area).
func (l Layout) LeftPanelBounds() (x, y, width, height int) {
	return 0, 0, l.LeftWidth, l.FileTreeHeight
}

// GitPanelBounds returns the position and size of the git panel.
func (l Layout) GitPanelBounds() (x, y, width, height int) {
	if !l.GitPanelVisible {
		return 0, 0, 0, 0
	}
	return 0, l.FileTreeHeight, l.LeftWidth, l.GitPanelHeight
}

// RightPanelBounds returns the position and size of the right panel.
func (l Layout) RightPanelBounds() (x, y, width, height int) {
	return l.LeftWidth, 0, l.RightWidth, l.MainHeight
}

// MiniBufferBounds returns the position and size of the mini buffer.
func (l Layout) MiniBufferBounds() (x, y, width, height int) {
	if !l.MiniVisible {
		return 0, 0, 0, 0
	}
	return 0, l.MainHeight, l.TotalWidth, l.MiniHeight
}

// StatusBarBounds returns the position and size of the status bar.
func (l Layout) StatusBarBounds() (x, y, width, height int) {
	statusY := l.MainHeight
	if l.MiniVisible {
		statusY += l.MiniHeight
	}
	return 0, statusY, l.TotalWidth, l.StatusHeight
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
