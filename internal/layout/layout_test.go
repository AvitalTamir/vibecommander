package layout

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		miniVisible bool
		wantLeft    int
		wantRight   int
		wantMain    int
		wantMini    int
	}{
		{
			name:        "standard layout without mini buffer",
			width:       100,
			height:      40,
			miniVisible: false,
			wantLeft:    25,
			wantRight:   75,
			wantMain:    39, // 40 - 1 (status bar)
			wantMini:    0,
		},
		{
			name:        "standard layout with mini buffer",
			width:       100,
			height:      40,
			miniVisible: true,
			wantLeft:    25,
			wantRight:   75,
			wantMain:    29, // 39 - 10 (mini buffer ~25%)
			wantMini:    10, // ~25% of 39
		},
		{
			name:        "small terminal",
			width:       60,
			height:      20,
			miniVisible: false,
			wantLeft:    20, // min width
			wantRight:   40,
			wantMain:    19,
			wantMini:    0,
		},
		{
			name:        "very small terminal respects minimums",
			width:       30,
			height:      10,
			miniVisible: true,
			wantLeft:    20, // min width
			wantRight:   20, // min width (will exceed total)
			wantMain:    5,  // min height
			wantMini:    5,  // min height (adjusted)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Calculate(tt.width, tt.height, tt.miniVisible, DefaultLeftPanelPercent)

			assert.Equal(t, tt.width, l.TotalWidth, "TotalWidth")
			assert.Equal(t, tt.height, l.TotalHeight, "TotalHeight")
			assert.Equal(t, tt.wantLeft, l.LeftWidth, "LeftWidth")
			assert.Equal(t, tt.miniVisible, l.MiniVisible, "MiniVisible")
			assert.Equal(t, StatusBarHeight, l.StatusHeight, "StatusHeight")

			if !tt.miniVisible {
				assert.Equal(t, 0, l.MiniHeight, "MiniHeight when not visible")
			}
		})
	}
}

func TestLayoutBounds(t *testing.T) {
	l := Calculate(100, 40, true, DefaultLeftPanelPercent)

	t.Run("LeftPanelBounds", func(t *testing.T) {
		x, y, width, height := l.LeftPanelBounds()
		assert.Equal(t, 0, x)
		assert.Equal(t, 0, y)
		assert.Equal(t, l.LeftWidth, width)
		assert.Equal(t, l.MainHeight, height)
	})

	t.Run("RightPanelBounds", func(t *testing.T) {
		x, y, width, height := l.RightPanelBounds()
		assert.Equal(t, l.LeftWidth, x)
		assert.Equal(t, 0, y)
		assert.Equal(t, l.RightWidth, width)
		assert.Equal(t, l.MainHeight, height)
	})

	t.Run("MiniBufferBounds when visible", func(t *testing.T) {
		x, y, width, height := l.MiniBufferBounds()
		assert.Equal(t, 0, x)
		assert.Equal(t, l.MainHeight, y)
		assert.Equal(t, l.TotalWidth, width)
		assert.Equal(t, l.MiniHeight, height)
	})

	t.Run("MiniBufferBounds when hidden", func(t *testing.T) {
		l2 := Calculate(100, 40, false, DefaultLeftPanelPercent)
		x, y, width, height := l2.MiniBufferBounds()
		assert.Equal(t, 0, x)
		assert.Equal(t, 0, y)
		assert.Equal(t, 0, width)
		assert.Equal(t, 0, height)
	})
}

func TestContentDimensions(t *testing.T) {
	l := Calculate(100, 40, false, DefaultLeftPanelPercent)

	t.Run("ContentWidth", func(t *testing.T) {
		width := l.ContentWidth(50, 1)
		assert.Equal(t, 48, width) // 50 - 2*1

		width = l.ContentWidth(50, 2)
		assert.Equal(t, 46, width) // 50 - 2*2
	})

	t.Run("ContentHeight", func(t *testing.T) {
		height := l.ContentHeight(30, 1)
		assert.Equal(t, 28, height) // 30 - 2*1
	})

	t.Run("ContentWidth handles zero", func(t *testing.T) {
		width := l.ContentWidth(2, 2)
		assert.Equal(t, 0, width) // max(2-4, 0) = 0
	})
}
