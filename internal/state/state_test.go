package state

import (
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
