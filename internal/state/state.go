package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	configDirName = ".config"
	appDirName    = "vibecommander"
	stateFileName = "state.json"
)

// State represents the persisted application state.
type State struct {
	// AIWindowOpen indicates if the AI assistant (Alt+2) was open
	AIWindowOpen bool `json:"ai_window_open"`
	// ThemeIndex is the index of the selected theme
	ThemeIndex int `json:"theme_index"`
	// LeftPanelPercent is the width percentage of the file tree panel (15-60)
	LeftPanelPercent int `json:"left_panel_percent,omitempty"`
	// CompactIndent indicates if the file tree uses compact (2-space) indentation
	CompactIndent bool `json:"compact_indent,omitempty"`
}

// DefaultState returns the default state for first run.
func DefaultState() State {
	return State{
		AIWindowOpen: false,
		ThemeIndex:   0,
	}
}

// configDir returns the path to the config directory (~/.config/vibecommander).
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, appDirName), nil
}

// statePath returns the global path to the state file.
func statePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

// Load reads the global application state.
// Returns default state if file doesn't exist or can't be read.
func Load() State {
	path, err := statePath()
	if err != nil {
		return DefaultState()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// File doesn't exist or can't be read - return defaults
		return DefaultState()
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		// Invalid JSON - return defaults
		return DefaultState()
	}

	return s
}

// Save writes the global application state.
func Save(s State) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path, err := statePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
