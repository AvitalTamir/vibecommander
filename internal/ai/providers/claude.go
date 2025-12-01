package providers

import (
	"context"
	"os"
	"os/exec"

	"github.com/avitaltamir/vibecommander/internal/ai"
)

// ClaudeCode implements the ai.Provider interface for Claude Code CLI.
type ClaudeCode struct{}

// NewClaudeCode creates a new Claude Code provider.
func NewClaudeCode() *ClaudeCode {
	return &ClaudeCode{}
}

// Name returns the provider identifier.
func (c *ClaudeCode) Name() string {
	return "claude-code"
}

// Command returns the exec.Cmd to start Claude Code.
func (c *ClaudeCode) Command(ctx context.Context, opts ai.Options) *exec.Cmd {
	args := c.DefaultArgs()

	// Add working directory if specified
	if opts.WorkingDir != "" {
		args = append(args, "--add-dir", opts.WorkingDir)
	}

	// Override system prompt if provided
	if opts.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", opts.SystemPrompt)
	}

	// Add model override
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	// Append any additional args
	args = append(args, opts.AdditionalArgs...)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = opts.WorkingDir

	// Inherit environment with terminal settings
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	return cmd
}

// IsAvailable checks if the Claude CLI is installed.
func (c *ClaudeCode) IsAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// DefaultArgs returns default command-line arguments.
func (c *ClaudeCode) DefaultArgs() []string {
	return []string{
		// Start in interactive mode (default behavior)
	}
}
