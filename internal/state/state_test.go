package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultState(t *testing.T) {
	s := DefaultState()

	assert.False(t, s.AIWindowOpen, "default AI window should be closed")
	assert.Equal(t, 0, s.ThemeIndex, "default theme index should be 0")
}

func TestConfigDir(t *testing.T) {
	dir, err := configDir()
	assert.NoError(t, err)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "vibecommander")
	assert.Equal(t, expected, dir)
}

func TestStatePath(t *testing.T) {
	path, err := statePath()
	assert.NoError(t, err)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "vibecommander", "state.json")
	assert.Equal(t, expected, path)
}

func TestStateAIFields(t *testing.T) {
	t.Run("DefaultState has empty AI fields", func(t *testing.T) {
		s := DefaultState()
		assert.Empty(t, s.AICommand)
		assert.Nil(t, s.AIArgs)
	})

	t.Run("State can hold AI command", func(t *testing.T) {
		s := State{
			AICommand: "claude",
			AIArgs:    nil,
		}
		assert.Equal(t, "claude", s.AICommand)
	})

	t.Run("State can hold AI args", func(t *testing.T) {
		s := State{
			AICommand: "my-cli",
			AIArgs:    []string{"--flag", "value"},
		}
		assert.Equal(t, "my-cli", s.AICommand)
		assert.Equal(t, []string{"--flag", "value"}, s.AIArgs)
	})
}

func TestSaveAndLoadState(t *testing.T) {
	// Create a temp directory for testing
	tempDir := t.TempDir()

	// Temporarily override the home directory for this test
	// by using a custom state that we serialize/deserialize manually
	t.Run("State serializes AI command correctly", func(t *testing.T) {
		original := State{
			AIWindowOpen:     true,
			ThemeIndex:       2,
			LeftPanelPercent: 30,
			CompactIndent:    true,
			AICommand:        "gemini",
			AIArgs:           []string{"--model", "pro"},
		}

		// Test JSON marshaling
		data, err := json.Marshal(original)
		assert.NoError(t, err)

		var loaded State
		err = json.Unmarshal(data, &loaded)
		assert.NoError(t, err)

		assert.Equal(t, original.AIWindowOpen, loaded.AIWindowOpen)
		assert.Equal(t, original.ThemeIndex, loaded.ThemeIndex)
		assert.Equal(t, original.LeftPanelPercent, loaded.LeftPanelPercent)
		assert.Equal(t, original.CompactIndent, loaded.CompactIndent)
		assert.Equal(t, original.AICommand, loaded.AICommand)
		assert.Equal(t, original.AIArgs, loaded.AIArgs)
	})

	t.Run("Empty AICommand omitted from JSON", func(t *testing.T) {
		s := State{
			AIWindowOpen: true,
			ThemeIndex:   1,
		}

		data, err := json.Marshal(s)
		assert.NoError(t, err)

		// AICommand should not be in JSON since it's empty and has omitempty
		jsonStr := string(data)
		assert.NotContains(t, jsonStr, "ai_command")
	})

	t.Run("Save and load with file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_state.json")
		original := State{
			AIWindowOpen: true,
			ThemeIndex:   3,
			AICommand:    "codex",
			AIArgs:       []string{"--verbose"},
		}

		// Marshal and write
		data, err := json.MarshalIndent(original, "", "  ")
		assert.NoError(t, err)
		err = os.WriteFile(testFile, data, 0644)
		assert.NoError(t, err)

		// Read and unmarshal
		readData, err := os.ReadFile(testFile)
		assert.NoError(t, err)

		var loaded State
		err = json.Unmarshal(readData, &loaded)
		assert.NoError(t, err)

		assert.Equal(t, original.AICommand, loaded.AICommand)
		assert.Equal(t, original.AIArgs, loaded.AIArgs)
	})
}
