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
		{PanelNone, "None"},
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

	t.Run("Quit dialog ctrl+q confirms quit", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showQuit = true

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})

		// cmd should be tea.Quit
		assert.NotNil(t, cmd)
	})

	t.Run("Double-tap ctrl+q quits immediately", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true

		// First ctrl+q shows dialog
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
		model := newModel.(Model)
		assert.True(t, model.showQuit)

		// Simulate immediate second ctrl+q (dialog handles it)
		_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})

		// Should quit
		assert.NotNil(t, cmd)
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
		assert.Contains(t, view, Version)  // Version in status bar
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

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	assert.NotEmpty(t, km.Quit.Keys())
	assert.NotEmpty(t, km.Help.Keys())
	assert.NotEmpty(t, km.FocusTree.Keys())
	assert.NotEmpty(t, km.FocusContent.Keys())
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

func TestSetFocus(t *testing.T) {
	m := New()
	// Ensure initial focus is on file tree
	assert.Equal(t, PanelFileTree, m.focus)

	// Set focus to content
	m = m.setFocus(PanelContent)
	assert.Equal(t, PanelContent, m.focus)

	// Verify file tree is blurred and content is focused
	assert.False(t, m.fileTree.Focused())
	assert.True(t, m.content.Focused())

	// Set focus back to file tree
	m = m.setFocus(PanelFileTree)
	assert.Equal(t, PanelFileTree, m.focus)
	assert.True(t, m.fileTree.Focused())
	assert.False(t, m.content.Focused())
}

func TestMouseClickSetsFocus(t *testing.T) {
	m := New()

	// Set window size to initialize layout
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updatedModel.(Model)
	assert.True(t, m.ready)

	// Initial focus should be file tree
	assert.Equal(t, PanelFileTree, m.focus)

	// Simulate click on content panel (right side)
	// Content panel starts at x = LeftWidth
	clickX := m.layout.LeftWidth + 10
	clickY := 5
	mouseMsg := tea.MouseMsg{
		X:      clickX,
		Y:      clickY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	updatedModel, _ = m.Update(mouseMsg)
	m = updatedModel.(Model)

	assert.Equal(t, PanelContent, m.focus, "expected focus to change to PanelContent after click")
	assert.True(t, m.content.Focused(), "expected content component to be focused")
	assert.False(t, m.fileTree.Focused(), "expected file tree component to be blurred")

	// Now click back on file tree (left side)
	mouseMsg = tea.MouseMsg{
		X:      5, // Within file tree bounds
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	updatedModel, _ = m.Update(mouseMsg)
	m = updatedModel.(Model)

	assert.Equal(t, PanelFileTree, m.focus, "expected focus to change back to PanelFileTree after click")
	assert.True(t, m.fileTree.Focused(), "expected file tree component to be focused")
	assert.False(t, m.content.Focused(), "expected content component to be blurred")
}

func TestPanelAtPosition(t *testing.T) {
	m := New()

	// Set window size
	updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updatedModel.(Model)

	tests := []struct {
		name     string
		x, y     int
		expected PanelID
	}{
		{"left panel top", 5, 5, PanelFileTree},
		{"left panel near edge", m.layout.LeftWidth - 1, 5, PanelFileTree},
		{"right panel start", m.layout.LeftWidth, 5, PanelContent},
		{"right panel middle", 80, 5, PanelContent},
		{"status bar area", 50, m.layout.MainHeight, PanelNone}, // Status bar returns PanelNone
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.panelAtPosition(tt.x, tt.y)
			assert.Equal(t, tt.expected, result, "panelAtPosition(%d, %d)", tt.x, tt.y)
		})
	}
}
