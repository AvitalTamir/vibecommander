package ai

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider implements Provider for testing
type MockProvider struct {
	name      string
	available bool
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Command(ctx context.Context, opts Options) *exec.Cmd {
	return exec.CommandContext(ctx, "echo", "mock")
}

func (m *MockProvider) IsAvailable() bool {
	return m.available
}

func (m *MockProvider) DefaultArgs() []string {
	return []string{"--mock"}
}

func TestRegistry(t *testing.T) {
	t.Run("NewRegistry creates empty registry", func(t *testing.T) {
		r := NewRegistry()
		assert.NotNil(t, r)
		assert.Empty(t, r.Names())
	})

	t.Run("Register adds provider", func(t *testing.T) {
		r := NewRegistry()
		p := &MockProvider{name: "test", available: true}

		r.Register(p)

		got, ok := r.Get("test")
		require.True(t, ok)
		assert.Equal(t, "test", got.Name())
	})

	t.Run("SetDefault and Get with empty name", func(t *testing.T) {
		r := NewRegistry()
		p := &MockProvider{name: "default-provider", available: true}

		r.Register(p)
		r.SetDefault("default-provider")

		got, ok := r.Get("")
		require.True(t, ok)
		assert.Equal(t, "default-provider", got.Name())
	})

	t.Run("Get returns false for unknown provider", func(t *testing.T) {
		r := NewRegistry()

		_, ok := r.Get("unknown")
		assert.False(t, ok)
	})

	t.Run("Available returns only available providers", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&MockProvider{name: "available", available: true})
		r.Register(&MockProvider{name: "unavailable", available: false})

		available := r.Available()
		assert.Len(t, available, 1)
		assert.Equal(t, "available", available[0].Name())
	})

	t.Run("All returns all providers", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&MockProvider{name: "p1", available: true})
		r.Register(&MockProvider{name: "p2", available: false})

		all := r.All()
		assert.Len(t, all, 2)
	})

	t.Run("Names returns all provider names", func(t *testing.T) {
		r := NewRegistry()
		r.Register(&MockProvider{name: "alpha", available: true})
		r.Register(&MockProvider{name: "beta", available: true})

		names := r.Names()
		assert.Len(t, names, 2)
		assert.Contains(t, names, "alpha")
		assert.Contains(t, names, "beta")
	})

	t.Run("Default returns default name", func(t *testing.T) {
		r := NewRegistry()
		r.SetDefault("my-default")

		assert.Equal(t, "my-default", r.Default())
	})
}

func TestOptions(t *testing.T) {
	t.Run("Options can be created with all fields", func(t *testing.T) {
		opts := Options{
			WorkingDir:     "/test/dir",
			SystemPrompt:   "You are helpful",
			Model:          "claude-3",
			AdditionalArgs: []string{"--verbose"},
		}

		assert.Equal(t, "/test/dir", opts.WorkingDir)
		assert.Equal(t, "You are helpful", opts.SystemPrompt)
		assert.Equal(t, "claude-3", opts.Model)
		assert.Contains(t, opts.AdditionalArgs, "--verbose")
	})
}
