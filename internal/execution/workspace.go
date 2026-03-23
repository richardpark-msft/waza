package execution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/microsoft/waza/internal/models"
)

type CleanupFunc func(ctx context.Context) error

type setupWorkspaceResourcesResult struct {
	// Dir is the temp directory where the workspace was created
	Dir string

	// CleanupFunc takes care of removing the resource properly, including any bookkeeping that might
	// be required from git.
	CleanupFunc CleanupFunc
}

func newCleanupDirFunc(workspaceDir string) CleanupFunc {
	return func(ctx context.Context) error {
		return os.RemoveAll(workspaceDir)
	}
}

// setupWorkspaceResources creates a temporary workspace folder, and writes resource files with path-traversal protection.
// It also supports using a git worktree for the initial workspace contents.
// Both CopilotEngine and MockEngine share this logic to keep sandbox behavior consistent.
func setupWorkspaceResources(ctx context.Context, resources []ResourceFile, gitResource *models.GitResource) (*setupWorkspaceResourcesResult, error) {
	workspaceDir, err := os.MkdirTemp("", "waza-*")

	if err != nil {
		return nil, fmt.Errorf("failed to create temp workspace: %w", err)
	}

	cleanup := newCleanupDirFunc(workspaceDir)

	defer func() {
		if err != nil && cleanup != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = cleanup(ctx)
		}
	}()

	// if they're using git worktree then we need to let that command create the folder since it won't
	// work if the folder already exists.
	if gitResource != nil {
		if err := os.RemoveAll(workspaceDir); err != nil {
			return nil, fmt.Errorf("failed to remove temp directory, to replace with worktree: %w", err)
		}

		res, err := CloneGitResource(ctx, *gitResource, workspaceDir)

		if err != nil {
			return nil, err
		}

		// when using some resources you have to do some bookkeeping so we can't just delete the folder
		cleanup = res.Cleanup
	}

	baseWorkspace := filepath.Clean(workspaceDir)
	if baseWorkspace == "" {
		return nil, fmt.Errorf("workspace path %s could not be normalized: %w", workspaceDir, err)
	}

	baseWithSep := baseWorkspace + string(os.PathSeparator)

	for _, res := range resources {
		if res.Path == "" {
			continue
		}

		relPath := filepath.Clean(res.Path)

		if filepath.IsAbs(relPath) {
			return nil, fmt.Errorf("resource path %q must be relative", res.Path)
		}

		fullPath := filepath.Join(baseWorkspace, relPath)

		fullPathClean := filepath.Clean(fullPath)
		fullWithSep := fullPathClean + string(os.PathSeparator)

		if !strings.HasPrefix(fullWithSep, baseWithSep) {
			return nil, fmt.Errorf("resource path %q escapes workspace", res.Path)
		}

		dir := filepath.Dir(fullPathClean)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating directory for resource %q: %w", res.Path, err)
		}

		if err := os.WriteFile(fullPathClean, res.Content, 0644); err != nil {
			return nil, fmt.Errorf("writing resource %q: %w", res.Path, err)
		}
	}

	return &setupWorkspaceResourcesResult{
		CleanupFunc: cleanup,
		Dir:         workspaceDir,
	}, nil
}
