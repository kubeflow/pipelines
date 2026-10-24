package poc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestDockerStagingCleanupPreservesPublishedArtifacts(t *testing.T) {
	original := viper.Get(common.V2RuntimeWorkspaceRoot)
	root := t.TempDir()
	viper.Set(common.V2RuntimeWorkspaceRoot, root)
	t.Cleanup(func() { viper.Set(common.V2RuntimeWorkspaceRoot, original) })
	staging := filepath.Join(root, "run-id", "artifact-local")
	require.NoError(t, os.MkdirAll(staging, 0o700))
	published := filepath.Join(root, "published-artifact")
	require.NoError(t, os.WriteFile(published, []byte("published"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(staging, "scratch"), []byte("temporary"), 0o600))
	require.NoError(t, (dockerExecutor{}).CleanupStaging(context.Background(), "run-id"))
	_, err := os.Stat(staging)
	require.ErrorIs(t, err, os.ErrNotExist)
	content, err := os.ReadFile(published)
	require.NoError(t, err)
	require.Equal(t, "published", string(content))
	require.NoError(t, (dockerExecutor{}).CleanupStaging(context.Background(), "run-id"))
}

func TestDockerStagingCleanupRejectsSymlinksAndInvalidIdentities(t *testing.T) {
	original := viper.Get(common.V2RuntimeWorkspaceRoot)
	root := t.TempDir()
	viper.Set(common.V2RuntimeWorkspaceRoot, root)
	t.Cleanup(func() { viper.Set(common.V2RuntimeWorkspaceRoot, original) })
	runPath := filepath.Join(root, "run-id")
	require.NoError(t, os.MkdirAll(runPath, 0o700))
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "artifact"), []byte("published"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(runPath, "artifact-local")))
	executor := dockerExecutor{}
	require.ErrorContains(t, executor.CleanupStaging(context.Background(), "run-id"), "symlink")
	for _, identity := range []string{"", ".", "..", "../run-id", "/tmp/run-id"} {
		require.ErrorContains(t, executor.CleanupStaging(context.Background(), identity), "invalid run identity")
	}
	content, err := os.ReadFile(filepath.Join(outside, "artifact"))
	require.NoError(t, err)
	require.Equal(t, "published", string(content))
}
