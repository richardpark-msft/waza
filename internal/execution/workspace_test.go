package execution

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/microsoft/waza/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupWorkspaceResources_WritesFiles(t *testing.T) {
	resources := []ResourceFile{
		{Path: "root.txt", Content: []byte("root")},
		{Path: "nested/child.txt", Content: []byte("child")},
		{Path: "", Content: []byte("ignored")},
	}

	resp, err := setupWorkspaceResources(context.Background(), resources, nil)
	require.NoError(t, err)

	defer func() { require.NoError(t, resp.CleanupFunc(context.Background())) }()

	rootContent, err := os.ReadFile(filepath.Join(resp.Dir, "root.txt"))
	require.NoError(t, err)
	assert.Equal(t, "root", string(rootContent))

	childContent, err := os.ReadFile(filepath.Join(resp.Dir, "nested", "child.txt"))
	require.NoError(t, err)
	assert.Equal(t, "child", string(childContent))
}

func TestSetupWorkspaceResources_RejectsAbsolutePath(t *testing.T) {
	absPath := "/etc/passwd"
	if runtime.GOOS == "windows" {
		absPath = `C:\etc\passwd`
	}

	_, err := setupWorkspaceResources(context.Background(), []ResourceFile{{Path: absPath, Content: []byte("x")}}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be relative")
}

func TestSetupWorkspaceResources_RejectsPathTraversal(t *testing.T) {
	_, err := setupWorkspaceResources(context.Background(), []ResourceFile{{Path: "../outside.txt", Content: []byte("x")}}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes workspace")
}

func TestSetupWorkspaceResources_Worktree(t *testing.T) {
	resources := []ResourceFile{
		{Path: "root.txt", Content: []byte("root")},
		{Path: "nested/child.txt", Content: []byte("child")},
		{Path: "", Content: []byte("ignored")},
	}

	resp, err := setupWorkspaceResources(context.Background(), resources, &models.GitResource{
		Commit: "HEAD",
		Type:   models.GitTypeWorktree,
		Source: ".",
	})
	require.NoError(t, err)

	defer func() { require.NoError(t, resp.CleanupFunc(context.Background())) }()

	// in git worktrees the .git "folder" is actually a file with the path to the actual git folder.
	require.FileExists(t, filepath.Join(resp.Dir, ".git"))
	require.FileExists(t, filepath.Join(resp.Dir, "root.txt"))
}

