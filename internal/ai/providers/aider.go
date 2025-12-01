package providers

import (
	"context"
	"os"
	"os/exec"

	"github.com/avitaltamir/vibecommander/internal/ai"
)

// Aider implements the ai.Provider interface for Aider CLI.
type Aider struct{}

// NewAider creates a new Aider provider.
func NewAider() *Aider {
	return &Aider{}
}

// Name returns the provider identifier.
func (a *Aider) Name() string {
	return "aider"
}

// Command returns the exec.Cmd to start Aider.
func (a *Aider) Command(ctx context.Context, opts ai.Options) *exec.Cmd {
	args := a.DefaultArgs()

	// Model selection
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Append any additional args
	args = append(args, opts.AdditionalArgs...)

	cmd := exec.CommandContext(ctx, "aider", args...)
	cmd.Dir = opts.WorkingDir

	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
	)

	return cmd
}

// IsAvailable checks if Aider is installed.
func (a *Aider) IsAvailable() bool {
	_, err := exec.LookPath("aider")
	return err == nil
}

// DefaultArgs returns default command-line arguments.
func (a *Aider) DefaultArgs() []string {
	return []string{
		"--no-auto-commits", // Let Vibe Commander handle git
	}
}
