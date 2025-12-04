package selection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := New()
	assert.False(t, m.HasSelection())
	assert.Equal(t, "", m.GetSelectedText())
}

func TestStartSelection(t *testing.T) {
	m := New()
	m.StartSelection(5, 10)

	assert.True(t, m.Selection.Active)
	assert.Equal(t, 5, m.Selection.Start.Line)
	assert.Equal(t, 10, m.Selection.Start.Column)
	assert.Equal(t, 5, m.Selection.End.Line)
	assert.Equal(t, 10, m.Selection.End.Column)
	assert.False(t, m.Selection.Complete)
}

func TestUpdateSelection(t *testing.T) {
	m := New()
	m.StartSelection(5, 10)
	m.UpdateSelection(7, 15)

	assert.True(t, m.Selection.Active)
	assert.Equal(t, 5, m.Selection.Start.Line)
	assert.Equal(t, 10, m.Selection.Start.Column)
	assert.Equal(t, 7, m.Selection.End.Line)
	assert.Equal(t, 15, m.Selection.End.Column)
}

func TestUpdateSelectionNotActive(t *testing.T) {
	m := New()
	// Try to update without starting selection
	m.UpdateSelection(7, 15)

	// Should not change anything
	assert.False(t, m.Selection.Active)
	assert.Equal(t, 0, m.Selection.End.Line)
	assert.Equal(t, 0, m.Selection.End.Column)
}

func TestEndSelection(t *testing.T) {
	m := New()
	m.StartSelection(5, 10)
	m.UpdateSelection(7, 15)
	m.EndSelection()

	assert.False(t, m.Selection.Active)
	assert.True(t, m.Selection.Complete)
}

func TestEndSelectionNotActive(t *testing.T) {
	m := New()
	// Try to end without starting
	m.EndSelection()

	assert.False(t, m.Selection.Active)
	assert.False(t, m.Selection.Complete)
}

func TestClearSelection(t *testing.T) {
	m := New()
	m.StartSelection(5, 10)
	m.UpdateSelection(7, 15)
	m.EndSelection()
	m.ClearSelection()

	assert.False(t, m.HasSelection())
	assert.False(t, m.Selection.Active)
	assert.False(t, m.Selection.Complete)
}

func TestHasSelection(t *testing.T) {
	m := New()

	// No selection initially
	assert.False(t, m.HasSelection())

	// Active but not complete
	m.StartSelection(5, 10)
	assert.False(t, m.HasSelection())

	// Complete selection
	m.UpdateSelection(7, 15)
	m.EndSelection()
	assert.True(t, m.HasSelection())

	// Same start and end (no actual selection)
	m2 := New()
	m2.StartSelection(5, 10)
	m2.EndSelection()
	assert.False(t, m2.HasSelection())
}

func TestGetSelectedTextSingleLine(t *testing.T) {
	m := New()
	m.SetContent([]string{
		"Hello, World!",
		"This is line 2",
		"And line 3",
	})

	m.StartSelection(0, 7)
	m.UpdateSelection(0, 12)
	m.EndSelection()

	assert.Equal(t, "World", m.GetSelectedText())
}

func TestGetSelectedTextMultiLine(t *testing.T) {
	m := New()
	m.SetContent([]string{
		"First line",
		"Second line",
		"Third line",
	})

	m.StartSelection(0, 6)
	m.UpdateSelection(2, 5)
	m.EndSelection()

	expected := "line\nSecond line\nThird"
	assert.Equal(t, expected, m.GetSelectedText())
}

func TestGetSelectedTextReverseSelection(t *testing.T) {
	m := New()
	m.SetContent([]string{
		"Hello, World!",
	})

	// Select backwards (end before start)
	m.StartSelection(0, 12)
	m.UpdateSelection(0, 7)
	m.EndSelection()

	assert.Equal(t, "World", m.GetSelectedText())
}

func TestGetSelectedTextEmpty(t *testing.T) {
	m := New()
	m.SetContent([]string{})

	// No selection
	assert.Equal(t, "", m.GetSelectedText())

	// Selection on empty content
	m.StartSelection(0, 0)
	m.UpdateSelection(0, 5)
	m.EndSelection()
	assert.Equal(t, "", m.GetSelectedText())
}

func TestIsSelected(t *testing.T) {
	m := New()
	m.SetContent([]string{
		"Line 0",
		"Line 1",
		"Line 2",
	})

	m.StartSelection(0, 3)
	m.UpdateSelection(2, 2)
	m.EndSelection()

	// Before selection start
	assert.False(t, m.IsSelected(0, 0))
	assert.False(t, m.IsSelected(0, 2))

	// Within selection
	assert.True(t, m.IsSelected(0, 3))
	assert.True(t, m.IsSelected(0, 5))
	assert.True(t, m.IsSelected(1, 0))
	assert.True(t, m.IsSelected(1, 5))
	assert.True(t, m.IsSelected(2, 0))
	assert.True(t, m.IsSelected(2, 1))

	// At or after selection end
	assert.False(t, m.IsSelected(2, 2))
	assert.False(t, m.IsSelected(2, 5))
	assert.False(t, m.IsSelected(3, 0))
}

func TestIsCopyKey(t *testing.T) {
	// Copy keys should be recognized
	assert.True(t, IsCopyKey("ctrl+c"))
	assert.True(t, IsCopyKey("y"))       // Vim-style yank
	assert.True(t, IsCopyKey("ctrl+y"))  // Alternative binding

	// Other keys should not be recognized
	assert.False(t, IsCopyKey("c"))
	assert.False(t, IsCopyKey("ctrl+v"))
	assert.False(t, IsCopyKey("ctrl+x"))
	assert.False(t, IsCopyKey("enter"))
}

func TestClamp(t *testing.T) {
	assert.Equal(t, 5, clamp(5, 0, 10))
	assert.Equal(t, 0, clamp(-5, 0, 10))
	assert.Equal(t, 10, clamp(15, 0, 10))
	assert.Equal(t, 0, clamp(0, 0, 10))
	assert.Equal(t, 10, clamp(10, 0, 10))
}
