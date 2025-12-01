package layout

// Layout constants
const (
	LeftPanelPercent  = 25
	RightPanelPercent = 75
	MiniBufferPercent = 25
	StatusBarHeight   = 1
	MinPanelWidth     = 20
	MinPanelHeight    = 5
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

	// Status bar
	StatusHeight int

	// Mini buffer visibility
	MiniVisible bool
}

// Calculate computes the layout dimensions based on terminal size.
func Calculate(width, height int, miniVisible bool) Layout {
	l := Layout{
		TotalWidth:   width,
		TotalHeight:  height,
		StatusHeight: StatusBarHeight,
		MiniVisible:  miniVisible,
	}

	// Calculate horizontal split (25/75)
	l.LeftWidth = max(width*LeftPanelPercent/100, MinPanelWidth)
	l.RightWidth = max(width-l.LeftWidth, MinPanelWidth)

	// Ensure we don't exceed total width
	if l.LeftWidth+l.RightWidth > width {
		l.RightWidth = width - l.LeftWidth
	}

	// Reserve status bar
	availableHeight := height - l.StatusHeight

	// Calculate vertical split
	if miniVisible {
		l.MiniHeight = max(availableHeight*MiniBufferPercent/100, MinPanelHeight)
		l.MainHeight = max(availableHeight-l.MiniHeight-1, MinPanelHeight)
	} else {
		l.MiniHeight = 0
		l.MainHeight = availableHeight
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

// LeftPanelBounds returns the position and size of the left panel.
func (l Layout) LeftPanelBounds() (x, y, width, height int) {
	return 0, 0, l.LeftWidth, l.MainHeight
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
