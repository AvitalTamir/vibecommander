package git

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

// ShellProvider implements Provider using shell git commands.
type ShellProvider struct {
	workDir string
}

// NewShellProvider creates a new shell-based git provider.
func NewShellProvider(workDir string) *ShellProvider {
	return &ShellProvider{workDir: workDir}
}

// IsRepo checks if the current directory is a git repository.
func (p *ShellProvider) IsRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = p.workDir
	err := cmd.Run()
	return err == nil
}

// GetBranch returns the current branch name.
func (p *ShellProvider) GetBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = p.workDir
	out, err := cmd.Output()
	if err != nil {
		// Try getting HEAD ref for detached state
		cmd = exec.CommandContext(ctx, "git", "rev-parse", "--short", "HEAD")
		cmd.Dir = p.workDir
		out, err = cmd.Output()
		if err != nil {
			return "", err
		}
		return "(" + strings.TrimSpace(string(out)) + ")", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// GetStatus returns the status of files in the repository.
func (p *ShellProvider) GetStatus(ctx context.Context) (*Status, error) {
	status := NewStatus()

	// Get branch
	branch, err := p.GetBranch(ctx)
	if err == nil {
		status.Branch = branch
	}

	// Get status with porcelain format
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v1", "-uall")
	cmd.Dir = p.workDir
	out, err := cmd.Output()
	if err != nil {
		return status, err
	}

	// Parse status output
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		staging := StatusCode(line[0])
		worktree := StatusCode(line[1])
		path := strings.TrimSpace(line[3:])

		// Handle renamed files (format: "R  old -> new")
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				path = parts[1]
			}
		}

		// Make path relative to workdir
		if !filepath.IsAbs(path) {
			path = filepath.Clean(path)
		}

		fs := FileStatus{
			Path:     path,
			Staging:  staging,
			Worktree: worktree,
		}

		status.Files[path] = fs

		if staging == StatusUntracked || worktree == StatusUntracked {
			status.Untracked = append(status.Untracked, path)
		}

		if fs.HasChanges() {
			status.IsDirty = true
		}
	}

	// Get ahead/behind info
	cmd = exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	cmd.Dir = p.workDir
	out, err = cmd.Output()
	if err == nil {
		var behind, ahead int
		parts := strings.Fields(strings.TrimSpace(string(out)))
		if len(parts) == 2 {
			// Parse behind (first number) and ahead (second number)
			if n, err := parseIntSafe(parts[0]); err == nil {
				behind = n
			}
			if n, err := parseIntSafe(parts[1]); err == nil {
				ahead = n
			}
		}
		status.Ahead = ahead
		status.Behind = behind
	}

	return status, nil
}

// GetDiff returns the diff for a file or the entire working tree.
func (p *ShellProvider) GetDiff(ctx context.Context, path string) (string, error) {
	var cmd *exec.Cmd
	if path == "" {
		cmd = exec.CommandContext(ctx, "git", "diff")
	} else {
		cmd = exec.CommandContext(ctx, "git", "diff", "--", path)
	}
	cmd.Dir = p.workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return stdout.String(), nil
}

func parseIntSafe(s string) (int, error) {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n, nil
}
