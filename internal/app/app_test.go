package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	m := New()

	assert.Equal(t, PanelFileTree, m.Focus())
	assert.False(t, m.MiniVisible())
	assert.NotNil(t, m.theme)
	assert.NotNil(t, m.keys)
}

func TestPanelIDString(t *testing.T) {
	tests := []struct {
		panel    PanelID
		expected string
	}{
		{PanelFileTree, "FileTree"},
		{PanelContent, "Content"},
		{PanelMiniBuffer, "MiniBuffer"},
		{PanelID(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.panel.String())
		})
	}
}

func TestModelUpdate(t *testing.T) {
	t.Run("WindowSizeMsg sets dimensions", func(t *testing.T) {
		m := New()

		newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

		assert.Equal(t, 100, newModel.(Model).width)
		assert.Equal(t, 40, newModel.(Model).height)
		assert.True(t, newModel.(Model).ready)
	})

	t.Run("Quit key shows quit dialog", func(t *testing.T) {
		m := New()
		// First set a size so the model is ready
		newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

		// Press ctrl+q to show quit dialog
		newModel, _ = newModel.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
		model := newModel.(Model)

		// Should show quit dialog
		assert.True(t, model.showQuit)
	})

	t.Run("Quit dialog Y confirms quit", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showQuit = true

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

		// cmd should be tea.Quit
		assert.NotNil(t, cmd)
	})

	t.Run("Quit dialog N cancels quit", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showQuit = true

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		model := newModel.(Model)

		// Should hide quit dialog
		assert.False(t, model.showQuit)
	})

	t.Run("Tab cycles focus forward", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true

		assert.Equal(t, PanelFileTree, m.Focus())

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})

		assert.Equal(t, PanelContent, newModel.(Model).Focus())
	})

	t.Run("Tab cycles back to first panel", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.focus = PanelContent

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})

		assert.Equal(t, PanelFileTree, newModel.(Model).Focus())
	})

	t.Run("Tab includes mini buffer when visible", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.miniVisible = true
		m.focus = PanelContent

		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})

		assert.Equal(t, PanelMiniBuffer, newModel.(Model).Focus())
	})

	t.Run("ToggleMiniBufferMsg toggles visibility", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true

		assert.False(t, m.MiniVisible())

		newModel, _ := m.Update(ToggleMiniBufferMsg{})

		assert.True(t, newModel.(Model).MiniVisible())
	})

	t.Run("FocusMsg sets focus", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true

		newModel, _ := m.Update(FocusMsg{Target: PanelContent})
		model := newModel.(Model)

		assert.Equal(t, PanelContent, model.Focus())
		assert.Equal(t, PanelFileTree, model.prevFocus)
	})
}

func TestModelView(t *testing.T) {
	t.Run("returns loading when not ready", func(t *testing.T) {
		m := New()
		view := m.View()

		assert.Contains(t, view, "Initializing")
	})

	t.Run("renders panels when ready", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.layout.LeftWidth = 25
		m.layout.RightWidth = 75
		m.layout.MainHeight = 39
		m.layout.TotalWidth = 100
		// Propagate sizes to child components
		m = m.updateSizes()

		view := m.View()

		assert.Contains(t, view, "FILES")
		assert.Contains(t, view, "VIEWER") // Content pane shows current mode
		assert.Contains(t, view, "Vibe Commander")
	})

	t.Run("renders mini buffer when visible", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.miniVisible = true
		m.layout.LeftWidth = 25
		m.layout.RightWidth = 75
		m.layout.MainHeight = 29
		m.layout.MiniHeight = 10
		m.layout.TotalWidth = 100
		m.layout.MiniVisible = true
		m = m.updateSizes()

		view := m.View()

		assert.Contains(t, view, "TERMINAL")
	})
}

func TestCycleFocus(t *testing.T) {
	t.Run("cycles forward without mini buffer", func(t *testing.T) {
		m := New()
		m.miniVisible = false

		m = m.cycleFocus(1)
		assert.Equal(t, PanelContent, m.Focus())

		m = m.cycleFocus(1)
		assert.Equal(t, PanelFileTree, m.Focus())
	})

	t.Run("cycles backward without mini buffer", func(t *testing.T) {
		m := New()
		m.miniVisible = false

		m = m.cycleFocus(-1)
		assert.Equal(t, PanelContent, m.Focus())

		m = m.cycleFocus(-1)
		assert.Equal(t, PanelFileTree, m.Focus())
	})

	t.Run("cycles through all panels with mini buffer", func(t *testing.T) {
		m := New()
		m.miniVisible = true

		m = m.cycleFocus(1)
		assert.Equal(t, PanelContent, m.Focus())

		m = m.cycleFocus(1)
		assert.Equal(t, PanelMiniBuffer, m.Focus())

		m = m.cycleFocus(1)
		assert.Equal(t, PanelFileTree, m.Focus())
	})
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
	assert.NotEmpty(t, km.FocusNext.Keys())
	assert.NotEmpty(t, km.Up.Keys())
	assert.NotEmpty(t, km.Down.Keys())
	assert.NotEmpty(t, km.Enter.Keys())
}

func TestKeyMapHelp(t *testing.T) {
	km := DefaultKeyMap()

	t.Run("ShortHelp returns bindings", func(t *testing.T) {
		short := km.ShortHelp()
		assert.NotEmpty(t, short)
	})

	t.Run("FullHelp returns binding groups", func(t *testing.T) {
		full := km.FullHelp()
		assert.NotEmpty(t, full)
		assert.Greater(t, len(full), 1)
	})
}
