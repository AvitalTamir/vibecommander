package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase(t *testing.T) {
	t.Run("NewBase creates with dimensions", func(t *testing.T) {
		b := NewBase(100, 50)

		w, h := b.Size()
		assert.Equal(t, 100, w)
		assert.Equal(t, 50, h)
		assert.False(t, b.Focused())
	})

	t.Run("Focus and Blur toggle state", func(t *testing.T) {
		b := NewBase(100, 50)

		assert.False(t, b.Focused())

		b.Focus()
		assert.True(t, b.Focused())

		b.Blur()
		assert.False(t, b.Focused())
	})

	t.Run("SetSize updates dimensions", func(t *testing.T) {
		b := NewBase(100, 50)

		b.SetSize(200, 100)

		w, h := b.Size()
		assert.Equal(t, 200, w)
		assert.Equal(t, 100, h)
	})

	t.Run("Zero dimensions are valid", func(t *testing.T) {
		b := NewBase(0, 0)

		w, h := b.Size()
		assert.Equal(t, 0, w)
		assert.Equal(t, 0, h)
	})
}
