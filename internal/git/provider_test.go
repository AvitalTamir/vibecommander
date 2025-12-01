package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusCode(t *testing.T) {
	tests := []struct {
		code       StatusCode
		str        string
		isModified bool
	}{
		{StatusUnmodified, " ", false},
		{StatusModified, "M", true},
		{StatusAdded, "A", false},
		{StatusDeleted, "D", false},
		{StatusRenamed, "R", false},
		{StatusCopied, "C", false},
		{StatusUnmerged, "U", false},
		{StatusUntracked, "?", false},
		{StatusIgnored, "!", false},
	}

	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			assert.Equal(t, tt.str, tt.code.String())
			assert.Equal(t, tt.isModified, tt.code.IsModified())
		})
	}
}

func TestFileStatus(t *testing.T) {
	t.Run("IsStaged returns true for staged files", func(t *testing.T) {
		fs := FileStatus{
			Path:     "test.go",
			Staging:  StatusAdded,
			Worktree: StatusUnmodified,
		}
		assert.True(t, fs.IsStaged())
	})

	t.Run("IsStaged returns false for untracked files", func(t *testing.T) {
		fs := FileStatus{
			Path:     "test.go",
			Staging:  StatusUntracked,
			Worktree: StatusUntracked,
		}
		assert.False(t, fs.IsStaged())
	})

	t.Run("IsStaged returns false for unmodified staging", func(t *testing.T) {
		fs := FileStatus{
			Path:     "test.go",
			Staging:  StatusUnmodified,
			Worktree: StatusModified,
		}
		assert.False(t, fs.IsStaged())
	})

	t.Run("HasChanges returns true when staging has changes", func(t *testing.T) {
		fs := FileStatus{
			Path:     "test.go",
			Staging:  StatusModified,
			Worktree: StatusUnmodified,
		}
		assert.True(t, fs.HasChanges())
	})

	t.Run("HasChanges returns true when worktree has changes", func(t *testing.T) {
		fs := FileStatus{
			Path:     "test.go",
			Staging:  StatusUnmodified,
			Worktree: StatusModified,
		}
		assert.True(t, fs.HasChanges())
	})

	t.Run("HasChanges returns false when no changes", func(t *testing.T) {
		fs := FileStatus{
			Path:     "test.go",
			Staging:  StatusUnmodified,
			Worktree: StatusUnmodified,
		}
		assert.False(t, fs.HasChanges())
	})
}

func TestNewStatus(t *testing.T) {
	s := NewStatus()

	assert.NotNil(t, s)
	assert.NotNil(t, s.Files)
	assert.Empty(t, s.Files)
	assert.Empty(t, s.Branch)
	assert.False(t, s.IsDirty)
	assert.Zero(t, s.Ahead)
	assert.Zero(t, s.Behind)
}

func TestStatus(t *testing.T) {
	t.Run("can add files to status", func(t *testing.T) {
		s := NewStatus()
		s.Branch = "main"
		s.IsDirty = true
		s.Files["test.go"] = FileStatus{
			Path:     "test.go",
			Staging:  StatusModified,
			Worktree: StatusUnmodified,
		}

		assert.Equal(t, "main", s.Branch)
		assert.True(t, s.IsDirty)
		assert.Len(t, s.Files, 1)
		assert.Equal(t, StatusModified, s.Files["test.go"].Staging)
	})
}
