package poc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type blockedWorkspaceCleanup struct {
	WorkspaceProvisionerPlugin
	entered chan struct{}
	release chan struct{}
}

func (*blockedWorkspaceCleanup) Type() string { return WorkspaceTypeHostPath }
func (p *blockedWorkspaceCleanup) Cleanup(ctx context.Context, _ *WorkspaceHandle) error {
	close(p.entered)
	select {
	case <-p.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestManagedRetryCannotReopenBetweenDeletionCleanupAndMetadataRemoval(t *testing.T) {
	original, root := viper.Get(common.V2RuntimeExecutor), viper.Get(common.V2RuntimeWorkspaceRoot)
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeWorkspaceRoot, t.TempDir())
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, original)
		viper.Set(common.V2RuntimeWorkspaceRoot, root)
	})
	coordinator, _, _ := publicationFixture(t)
	provisioner := &blockedWorkspaceCleanup{entered: make(chan struct{}), release: make(chan struct{})}
	coordinator.RegisterWorkspaceProvisioner(provisioner)
	manifest := NewRuntimeManifest(ExecutorDocker)
	manifest.Workspace = &WorkspaceHandle{Version: 1, Type: WorkspaceTypeHostPath}
	encoded, err := manifest.ToJSON()
	require.NoError(t, err)
	executor := ExecutorDocker
	run, err := coordinator.runStore.CreateRun(&model.Run{
		UUID: "lifecycle-race", K8SName: "lifecycle-race",
		RunDetails: model.RunDetails{
			State: model.RuntimeStateFailed, ExecutorType: &executor, PipelineRuntimeManifest: model.LargeText(encoded),
		},
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	deleted := make(chan error, 1)
	go func() { deleted <- coordinator.DeleteManagedRun(ctx, run.UUID) }()
	<-provisioner.entered
	retried := make(chan error, 1)
	go func() { retried <- coordinator.RetryManagedRun(ctx, run) }()
	select {
	case err := <-retried:
		t.Fatalf("retry escaped deletion lifecycle ownership: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(provisioner.release)
	require.NoError(t, <-deleted)
	require.Error(t, <-retried)
	_, err = coordinator.runStore.GetRun(run.UUID, false)
	require.Error(t, err)
}

func TestRunLifecycleWaitHonorsCancellation(t *testing.T) {
	coordinator := NewCoordinator(nil, nil, nil, nil, nil, nil, nil, false)
	unlock, err := coordinator.lockRunLifecycle(context.Background(), "run")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = coordinator.lockRunLifecycle(ctx, "run")
	require.True(t, errors.Is(err, context.DeadlineExceeded))
	unlock()
	require.Empty(t, coordinator.lifecycleLocks)
}
