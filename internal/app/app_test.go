package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
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
		newModel, _ = newModel.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
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

		_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})

		// cmd should be tea.Quit
		assert.NotNil(t, cmd)
	})

	t.Run("Quit dialog N cancels quit", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showQuit = true

		newModel, _ := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
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

		_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})

		// cmd should be tea.Quit
		assert.NotNil(t, cmd)
	})

	t.Run("Double-tap ctrl+q quits immediately", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true

		// First ctrl+q shows dialog
		newModel, _ := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
		model := newModel.(Model)
		assert.True(t, model.showQuit)

		// Simulate immediate second ctrl+q (dialog handles it)
		_, cmd := model.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})

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
	t.Run("returns view when not ready", func(t *testing.T) {
		m := New()
		view := m.View()

		// Just verify view is returned without panic
		assert.True(t, view.AltScreen)
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

		// Verify view is properly configured
		assert.True(t, view.AltScreen)
		assert.Equal(t, tea.MouseModeCellMotion, view.MouseMode)
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

		// Verify view is properly configured
		assert.True(t, view.AltScreen)
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
	mouseMsg := tea.MouseClickMsg{
		X:      clickX,
		Y:      clickY,
		Button: tea.MouseLeft,
	}

	updatedModel, _ = m.Update(mouseMsg)
	m = updatedModel.(Model)

	assert.Equal(t, PanelContent, m.focus, "expected focus to change to PanelContent after click")
	assert.True(t, m.content.Focused(), "expected content component to be focused")
	assert.False(t, m.fileTree.Focused(), "expected file tree component to be blurred")

	// Now click back on file tree (left side)
	mouseMsg = tea.MouseClickMsg{
		X:      5, // Within file tree bounds
		Y:      5,
		Button: tea.MouseLeft,
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

func TestCheckCommandAvailable(t *testing.T) {
	t.Run("empty command returns false", func(t *testing.T) {
		assert.False(t, checkCommandAvailable(""))
	})

	t.Run("whitespace only returns false", func(t *testing.T) {
		assert.False(t, checkCommandAvailable("   "))
	})

	t.Run("existing command returns true", func(t *testing.T) {
		// 'ls' should exist on all unix systems
		assert.True(t, checkCommandAvailable("ls"))
	})

	t.Run("non-existent command returns false", func(t *testing.T) {
		assert.False(t, checkCommandAvailable("definitely-not-a-real-command-12345"))
	})

	t.Run("command with args checks only first word", func(t *testing.T) {
		// 'ls' with args should still return true
		assert.True(t, checkCommandAvailable("ls -la --color"))
	})

	t.Run("non-existent command with args returns false", func(t *testing.T) {
		assert.False(t, checkCommandAvailable("not-real-cmd --flag value"))
	})
}

func TestAIDialogOptions(t *testing.T) {
	m := New()

	t.Run("returns four options", func(t *testing.T) {
		options := m.getAIDialogOptions()
		assert.Len(t, options, 4)
		assert.Equal(t, "Claude", options[0].name)
		assert.Equal(t, "Gemini", options[1].name)
		assert.Equal(t, "Codex", options[2].name)
		assert.Equal(t, "Other", options[3].name)
	})

	t.Run("Other option uses custom command", func(t *testing.T) {
		m.aiDialogCustom = "my-custom-cli"
		options := m.getAIDialogOptions()
		assert.Equal(t, "my-custom-cli", options[3].command)
	})

	t.Run("Other option available when empty", func(t *testing.T) {
		m.aiDialogCustom = ""
		options := m.getAIDialogOptions()
		// Empty custom command is considered "available" (for entering)
		assert.True(t, options[3].available)
	})
}

func TestAIDialogKeyHandling(t *testing.T) {
	t.Run("Ctrl+Alt+A opens AI dialog", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true

		// Simulate Ctrl+Alt+A
		newModel, _ := m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl | tea.ModAlt})
		model := newModel.(Model)

		assert.True(t, model.showAIDialog)
		assert.Equal(t, 0, model.aiDialogIndex)
	})

	t.Run("Alt+A without configured AI opens dialog", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.aiCommand = "" // No AI configured

		// Simulate Alt+A
		newModel, _ := m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModAlt})
		model := newModel.(Model)

		assert.True(t, model.showAIDialog)
	})

	t.Run("Escape closes AI dialog", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showAIDialog = true

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyEscape})
		assert.False(t, newModel.showAIDialog)
	})

	t.Run("Down arrow moves selection down", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogIndex = 0

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyDown})
		assert.Equal(t, 1, newModel.aiDialogIndex)
	})

	t.Run("Up arrow moves selection up", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogIndex = 2

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyUp})
		assert.Equal(t, 1, newModel.aiDialogIndex)
	})

	t.Run("j key moves selection down", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogIndex = 0

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: 'j', Text: "j"})
		assert.Equal(t, 1, newModel.aiDialogIndex)
	})

	t.Run("k key moves selection up", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogIndex = 2

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: 'k', Text: "k"})
		assert.Equal(t, 1, newModel.aiDialogIndex)
	})

	t.Run("selection does not go below zero", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogIndex = 0

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyUp})
		assert.Equal(t, 0, newModel.aiDialogIndex)
	})

	t.Run("selection does not exceed max", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogIndex = 3 // Last option (Other)

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyDown})
		assert.Equal(t, 3, newModel.aiDialogIndex)
	})

	t.Run("Enter on Other with empty custom starts editing", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogIndex = 3 // Other
		m.aiDialogCustom = ""

		newModel, _ := m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyEnter})
		assert.True(t, newModel.aiDialogEditing)
		assert.True(t, newModel.showAIDialog) // Dialog still open
	})
}

func TestAIDialogCustomEditing(t *testing.T) {
	t.Run("typing adds characters", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogEditing = true
		m.aiDialogCustom = ""

		m, _ = m.handleAIDialog(tea.KeyPressMsg{Code: 'a', Text: "a"})
		m, _ = m.handleAIDialog(tea.KeyPressMsg{Code: 'b', Text: "b"})
		m, _ = m.handleAIDialog(tea.KeyPressMsg{Code: 'c', Text: "c"})

		assert.Equal(t, "abc", m.aiDialogCustom)
	})

	t.Run("backspace removes characters", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogEditing = true
		m.aiDialogCustom = "test"

		m, _ = m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyBackspace})
		assert.Equal(t, "tes", m.aiDialogCustom)
	})

	t.Run("backspace on empty string does nothing", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogEditing = true
		m.aiDialogCustom = ""

		m, _ = m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyBackspace})
		assert.Equal(t, "", m.aiDialogCustom)
	})

	t.Run("escape cancels editing", func(t *testing.T) {
		m := New()
		m.showAIDialog = true
		m.aiDialogEditing = true
		m.aiDialogCustom = "partial"

		m, _ = m.handleAIDialog(tea.KeyPressMsg{Code: tea.KeyEscape})
		assert.False(t, m.aiDialogEditing)
		assert.True(t, m.showAIDialog) // Dialog still open
	})
}

func TestAIDialogRender(t *testing.T) {
	t.Run("renders without panic", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showAIDialog = true

		// Should not panic
		result := m.renderAIDialog("")
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "SELECT AI ASSISTANT")
		assert.Contains(t, result, "Claude")
		assert.Contains(t, result, "Gemini")
		assert.Contains(t, result, "Codex")
		assert.Contains(t, result, "Other")
	})

	t.Run("shows cursor when editing custom", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showAIDialog = true
		m.aiDialogIndex = 3
		m.aiDialogEditing = true
		m.aiDialogCustom = "test"

		result := m.renderAIDialog("")
		// Should contain the custom command with cursor
		assert.Contains(t, result, "test_")
	})
}

func TestAIDialogView(t *testing.T) {
	t.Run("View shows AI dialog when active", func(t *testing.T) {
		m := New()
		m.width = 100
		m.height = 40
		m.ready = true
		m.showAIDialog = true
		m.layout.LeftWidth = 25
		m.layout.RightWidth = 75
		m.layout.MainHeight = 39
		m.layout.TotalWidth = 100

		view := m.View()

		// Verify view is properly configured
		assert.True(t, view.AltScreen)
	})
}

func TestSelectAIKeyBinding(t *testing.T) {
	km := DefaultKeyMap()

	assert.NotEmpty(t, km.SelectAI.Keys())
	assert.Contains(t, km.SelectAI.Keys(), "ctrl+alt+a")
}
