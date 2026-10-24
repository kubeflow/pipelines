package poc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// RunStagingCleaner releases executor staging after all workloads and publication finish.
type RunStagingCleaner interface {
	CleanupStaging(context.Context, string) error
}

func (d dockerExecutor) CleanupStaging(ctx context.Context, runID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stagingPath, err := runtimeStagingPath(runID)
	if err != nil || stagingPath == "" {
		return err
	}
	if err := os.RemoveAll(stagingPath); err != nil {
		if !errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("remove Docker staging for run %s: %w", runID, err)
		}
		if err := (hostPathWorkspaceProvisioner{command: d.command}).repairWorkspacePermissions(ctx, stagingPath); err != nil {
			return err
		}
		if err := os.RemoveAll(stagingPath); err != nil {
			return fmt.Errorf("remove repaired Docker staging for run %s: %w", runID, err)
		}
	}
	return nil
}

func runtimeStagingPath(runID string) (string, error) {
	if runID == "" || runID == "." || runID == ".." || filepath.Base(runID) != runID {
		return "", fmt.Errorf("invalid run identity %q for staging cleanup", runID)
	}
	root, err := filepath.EvalSymlinks(WorkspaceRoot())
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("resolve runtime staging root: %w", err)
	}
	runPath := filepath.Join(root, runID)
	stagingPath := filepath.Join(runPath, "artifact-local")
	for _, path := range []string{runPath, stagingPath} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to clean non-directory or symlink staging path %q", path)
		}
	}
	return stagingPath, nil
}

func (c *Coordinator) cleanupRunStaging(ctx context.Context, runID, executorType string) error {
	executor, ok := c.executors[executorType]
	if !ok {
		return fmt.Errorf("cannot clean staging for run %s: executor %q is not configured", runID, executorType)
	}
	if cleaner, ok := executor.(RunStagingCleaner); ok {
		return cleaner.CleanupStaging(ctx, runID)
	}
	path, err := runtimeStagingPath(runID)
	if err != nil || path == "" {
		return err
	}
	return os.RemoveAll(path)
}
