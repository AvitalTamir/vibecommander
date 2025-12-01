package git

import "context"

// Provider defines the interface for git operations.
type Provider interface {
	// GetBranch returns the current branch name
	GetBranch(ctx context.Context) (string, error)

	// GetStatus returns the status of files in the repository
	GetStatus(ctx context.Context) (*Status, error)

	// GetDiff returns the diff for a file or the entire working tree
	GetDiff(ctx context.Context, path string) (string, error)

	// IsRepo checks if the current directory is a git repository
	IsRepo() bool
}

// Status represents the overall repository status.
type Status struct {
	Branch    string
	IsDirty   bool
	Ahead     int
	Behind    int
	Files     map[string]FileStatus
	Untracked []string
}

// FileStatus represents the status of a single file.
type FileStatus struct {
	Path     string
	Staging  StatusCode
	Worktree StatusCode
}

// StatusCode represents a git status code.
type StatusCode rune

const (
	StatusUnmodified StatusCode = ' '
	StatusModified   StatusCode = 'M'
	StatusAdded      StatusCode = 'A'
	StatusDeleted    StatusCode = 'D'
	StatusRenamed    StatusCode = 'R'
	StatusCopied     StatusCode = 'C'
	StatusUnmerged   StatusCode = 'U'
	StatusUntracked  StatusCode = '?'
	StatusIgnored    StatusCode = '!'
)

// String returns the single-character representation.
func (s StatusCode) String() string {
	return string(s)
}

// IsModified returns true if the file has been modified.
func (s StatusCode) IsModified() bool {
	return s == StatusModified
}

// IsStaged returns true if the file has been staged.
func (f FileStatus) IsStaged() bool {
	return f.Staging != StatusUnmodified && f.Staging != StatusUntracked
}

// HasChanges returns true if the file has any changes.
func (f FileStatus) HasChanges() bool {
	return f.Staging != StatusUnmodified || f.Worktree != StatusUnmodified
}

// NewStatus creates a new Status with initialized maps.
func NewStatus() *Status {
	return &Status{
		Files: make(map[string]FileStatus),
	}
}
