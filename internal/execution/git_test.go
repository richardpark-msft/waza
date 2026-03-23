package execution

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microsoft/waza/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustGetTempDir(t *testing.T) string {
	workspaceDir := t.TempDir()
	err := os.RemoveAll(workspaceDir) // will get created when we create the worktree
	require.NoError(t, err)
	return workspaceDir
}

func TestCreateGitResource(t *testing.T) {
	workspaceDir := mustGetTempDir(t)
	repoDir, _ := mustCreateRepo(t)

	resp, err := CloneGitResource(context.Background(),
		models.GitResource{Commit: "", Type: models.GitTypeWorktree, Source: repoDir},
		workspaceDir)
	require.NoError(t, err)

	require.FileExists(t, filepath.Join(workspaceDir, ".git")) // worktrees have a .git _file_ in the root
	require.FileExists(t, filepath.Join(workspaceDir, "hello.txt"))

	err = resp.Cleanup(context.Background())
	require.NoError(t, err)

	require.NoDirExists(t, workspaceDir)
}

func TestCloneGitResource_WorktreeDetachHEAD(t *testing.T) {
	repoDir, headCommit := mustCreateRepo(t)
	workspaceDir := mustGetTempDir(t)

	gitRes := &models.GitResource{
		Type:   models.GitTypeWorktree,
		Source: repoDir,
		Commit: headCommit,
	}

	res, err := CloneGitResource(context.Background(), *gitRes, workspaceDir)
	require.NoError(t, err, "CloneGitResource (detach)")

	_, err = os.Stat(filepath.Join(workspaceDir, "hello.txt"))
	require.NoError(t, err, "expected hello.txt in worktree")

	// Cleanup
	err = res.Cleanup(context.Background())
	require.NoError(t, err)
}

func TestCloneGitResource_UnsupportedType(t *testing.T) {
	_, commitSHA := mustCreateRepo(t)
	workspaceDir := mustGetTempDir(t)

	gitRes := &models.GitResource{
		Commit: commitSHA,
		Type:   "clone",
		Source: "/tmp/repo",
	}

	_, err := CloneGitResource(context.Background(), *gitRes, workspaceDir)
	require.Error(t, err, "expected unsupported type to be rejected")
	require.Contains(t, err.Error(), "invalid repo type")
}

func TestCloneGitResource_SourceDoesNotExist(t *testing.T) {
	workspaceDir := mustGetTempDir(t)
	missingDir := filepath.Join(t.TempDir(), "missing-repo")

	gitRes := &models.GitResource{
		Type:   models.GitTypeWorktree,
		Source: missingDir,
	}

	_, err := CloneGitResource(context.Background(), *gitRes, workspaceDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCloneGitResource_SourceIsNotDirectory(t *testing.T) {
	workspaceDir := mustGetTempDir(t)
	notDir := filepath.Join(t.TempDir(), "repo.txt")
	require.NoError(t, os.WriteFile(notDir, []byte("not a dir"), 0o644))

	gitRes := &models.GitResource{
		Type:   models.GitTypeWorktree,
		Source: notDir,
	}

	_, err := CloneGitResource(context.Background(), *gitRes, workspaceDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestCloneGitResource_SourceIsNotGitRepo(t *testing.T) {
	workspaceDir := mustGetTempDir(t)
	nonRepoDir := t.TempDir()

	gitRes := &models.GitResource{
		Type:   models.GitTypeWorktree,
		Source: nonRepoDir,
	}

	_, err := CloneGitResource(context.Background(), *gitRes, workspaceDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

// mustCreateRepo creates a repo with a single commit, with 'hello.txt' in the root (contents: "hello world")
func mustCreateRepo(t *testing.T) (repoDir string, headCommitSHA string) {
	repoDir = t.TempDir()

	_, err := runGitCommand(context.Background(), repoDir, "init")
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(repoDir, "hello.txt"), []byte("hello world"), 0644)
	require.NoError(t, err)

	_, err = runGitCommand(context.Background(), repoDir, "add", "hello.txt")
	require.NoError(t, err)

	_, err = runGitCommand(context.Background(), repoDir,
		"-c", "user.name=waza",
		"-c", "user.email=waza",
		"commit",
		"-m", "first and only file", "hello.txt")
	require.NoError(t, err)

	// Get commit SHA
	output, err := runGitCommand(context.Background(), repoDir, "rev-parse", "HEAD")
	require.NoError(t, err)

	return repoDir, strings.TrimSpace(output)
}
