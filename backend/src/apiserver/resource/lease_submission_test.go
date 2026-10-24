package resource

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	pocruntime "github.com/kubeflow/pipelines/backend/src/v2/runtime/poc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type observedCoordinatorRunStore struct {
	storage.RunStoreInterface
	afterCreate func(*model.Run) error
	creates     int
}

type invalidLeaseRecoveryExecutor struct{ invalidRecoveryExecutor }

func (invalidLeaseRecoveryExecutor) Type() string { return pocruntime.ExecutorKubernetes }

func (s *observedCoordinatorRunStore) CreateRun(run *model.Run) (*model.Run, error) {
	s.creates++
	created, err := s.RunStoreInterface.CreateRun(run)
	if err != nil {
		return nil, err
	}
	if s.afterCreate != nil {
		if err := s.afterCreate(created); err != nil {
			return nil, err
		}
	}
	return created, nil
}

func TestKubernetesSubmissionCannotOverwriteAClaimedRun(t *testing.T) {
	enableCoordinatorRuntimeModeForTest(t)
	for key, value := range map[string]any{
		common.V2RuntimeExecutor: pocruntime.ExecutorKubernetes, common.V2RuntimeAutoExecute: false,
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	t.Setenv("KFP_DEFAULT_PIPELINE_ROOT", "file://"+t.TempDir())
	initEnvVars()
	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "lease-submission", Namespace: "ns1"})
	require.NoError(t, err)
	provider, ok := manager.taskStore.(interface {
		RunLeaseStore() storage.RunLeaseStoreInterface
		LeaseScopedStores(storage.RunLeaseToken) (*storage.LeaseScopedStores, error)
	})
	require.True(t, ok)
	observed := &observedCoordinatorRunStore{RunStoreInterface: manager.runStore}
	observed.afterCreate = func(created *model.Run) error {
		token, err := provider.RunLeaseStore().Acquire(created.UUID, "concurrent-worker", time.Minute, false)
		if err != nil {
			return err
		}
		scoped, err := provider.LeaseScopedStores(*token)
		if err != nil {
			return err
		}
		progress := *created
		manifest := pocruntime.NewRuntimeManifest(pocruntime.ExecutorKubernetes)
		manifest.Tasks = []pocruntime.RuntimeTask{{TaskID: "owned-root", Name: "ROOT", ScopePath: "root"}}
		encoded, err := manifest.ToJSON()
		if err != nil {
			return err
		}
		progress.PipelineRuntimeManifest = model.LargeText(encoded)
		progress.State = model.RuntimeStateRunning
		return scoped.Runs.UpdateRun(&progress)
	}
	manager.runStore = observed
	manager.pocCoordinator = pocruntime.NewCoordinator(
		observed, manager.taskStore, manager.artifactStore, manager.artifactTaskStore,
		manager.time, nil, nil, false,
	)
	created, err := manager.CreateRun(context.Background(), &model.Run{
		DisplayName: "lease-submission", ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, observed.creates)
	require.NotEmpty(t, created.PipelineRoot, "the resolved root must be persisted before lease acquisition")
	var parameters map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(created.RuntimeConfig.Parameters), &parameters))
	require.Equal(t, "world", parameters["text"])
	require.Equal(t, model.RuntimeStateRunning, created.State, "API initialization must not overwrite the new owner's progress")
	manifest, err := pocruntime.ParseManifest(string(created.PipelineRuntimeManifest))
	require.NoError(t, err)
	require.Len(t, manifest.Tasks, 1, "API initialization must not replace the owner's materialized task manifest")
	require.Equal(t, "owned-root", manifest.Tasks[0].TaskID)

	manager.pocCoordinator.RegisterExecutor(invalidLeaseRecoveryExecutor{})
	_, err = manager.CreateRun(context.Background(), &model.Run{
		DisplayName: "invalid-lease-submission", ExperimentId: experiment.UUID,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
			RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`},
		},
	})
	require.ErrorContains(t, err, "recovery")
	require.Equal(t, 1, observed.creates, "failed initialization must never insert a run that a worker could claim")
}
