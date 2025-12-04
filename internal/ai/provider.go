package ai

import (
	"context"
	"os/exec"
)

// Provider defines the interface for AI CLI tools.
type Provider interface {
	// Name returns the provider identifier (e.g., "claude-code", "aider")
	Name() string

	// Command returns the exec.Cmd to start the AI tool
	Command(ctx context.Context, opts Options) *exec.Cmd

	// IsAvailable checks if the CLI tool is installed and accessible
	IsAvailable() bool

	// DefaultArgs returns default command-line arguments for this provider
	DefaultArgs() []string
}

// Options configures AI provider behavior.
type Options struct {
	// WorkingDir is the project directory
	WorkingDir string

	// SystemPrompt overrides the default system prompt (if supported)
	SystemPrompt string

	// Model specifies which AI model to use (if supported)
	Model string

	// AdditionalArgs for provider-specific flags
	AdditionalArgs []string
}

// Registry manages available AI providers.
type Registry struct {
	providers   map[string]Provider
	defaultName string
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// SetDefault sets the default provider by name.
func (r *Registry) SetDefault(name string) {
	r.defaultName = name
}

// Default returns the default provider name.
func (r *Registry) Default() string {
	return r.defaultName
}

// Get retrieves a provider by name. If name is empty, returns the default.
func (r *Registry) Get(name string) (Provider, bool) {
	if name == "" {
		name = r.defaultName
	}
	p, ok := r.providers[name]
	return p, ok
}

// Available returns all providers that are currently available (installed).
func (r *Registry) Available() []Provider {
	available := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		if p.IsAvailable() {
			available = append(available, p)
		}
	}
	return available
}

// All returns all registered providers.
func (r *Registry) All() []Provider {
	all := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		all = append(all, p)
	}
	return all
}

// Names returns all registered provider names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
