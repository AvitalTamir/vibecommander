package components

import tea "github.com/charmbracelet/bubbletea"

// Component defines the interface all Vibe Commander components must implement.
// It extends tea.Model with focus management and sizing capabilities.
type Component interface {
	tea.Model

	// Focus gives focus to this component
	Focus() Component
	// Blur removes focus from this component
	Blur() Component
	// Focused returns whether this component currently has focus
	Focused() bool

	// SetSize updates the component's dimensions
	SetSize(width, height int) Component
	// Size returns the component's current dimensions
	Size() (width, height int)
}

// Base provides common functionality for all components.
// Embed this in your component structs to get default implementations.
type Base struct {
	focused bool
	width   int
	height  int
}

// NewBase creates a new Base with the given dimensions.
func NewBase(width, height int) Base {
	return Base{
		width:  width,
		height: height,
	}
}

// Focus sets the focused state to true.
func (b *Base) Focus() {
	b.focused = true
}

// Blur sets the focused state to false.
func (b *Base) Blur() {
	b.focused = false
}

// Focused returns the current focus state.
func (b Base) Focused() bool {
	return b.focused
}

// SetSize updates the component's dimensions.
func (b *Base) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// Size returns the component's current dimensions.
func (b Base) Size() (width, height int) {
	return b.width, b.height
}
