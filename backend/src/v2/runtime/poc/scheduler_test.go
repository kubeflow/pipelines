package poc

import (
	"context"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

type barrierExecutor struct {
	fakeExecutorLifecycle
	started chan string
	release chan struct{}
}

func (b *barrierExecutor) Type() string { return ExecutorDocker }

func (b *barrierExecutor) ExecuteTask(ctx context.Context, request *TaskExecutionRequest) (*TaskExecutionResult, error) {
	b.started <- request.TaskName
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.release:
		return &TaskExecutionResult{}, nil
	}
}

func TestCoordinatorDispatchesIndependentTasksConcurrently(t *testing.T) {
	db, dialect, err := storage.NewFakeDB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	clock := util.NewFakeTimeForEpoch()
	runs := storage.NewRunStore(db, clock, dialect)
	tasks := storage.NewTaskStore(db, clock, util.NewUUIDGenerator(), dialect)
	run, err := runs.CreateRun(&model.Run{
		UUID: "parallel-run", Namespace: "ns", K8SName: "parallel-run",
		RunDetails: model.RunDetails{State: model.RuntimeStatePending},
	})
	require.NoError(t, err)
	coordinator := NewCoordinator(runs, tasks, nil, nil, clock, nil, nil, false)
	executor := &barrierExecutor{started: make(chan string, 2), release: make(chan struct{})}
	coordinator.RegisterExecutor(executor)
	manifest := NewRuntimeManifest(ExecutorDocker)
	root, err := coordinator.createTask(run, nil, "ROOT", "root", apiv2.PipelineTask_ROOT, "root", nil)
	require.NoError(t, err)
	spec := &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"root":{"dag":{"tasks":{
			"first":{"componentRef":{"name":"component"}},
			"second":{"componentRef":{"name":"component"}}
		}}},
		"components":{"component":{"executorLabel":"executor"}},
		"deploymentSpec":{"executors":{"executor":{"container":{"image":"test-image"}}}}
	}`), spec))
	require.NoError(t, coordinator.materializeComponent(run, spec, spec.Root, "root", root.UUID, manifest))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	item := &queuedRun{run: run, spec: spec, runtimeManifest: manifest}
	done := make(chan error, 1)
	go func() {
		done <- coordinator.executeDagComponent(ctx, item, executor, spec.Root, "root", nil, false)
	}()
	defer close(executor.release)
	for index := 0; index < 2; index++ {
		select {
		case <-executor.started:
		case err := <-done:
			t.Fatalf("DAG stopped before both independent tasks started: %v", err)
		case <-time.After(time.Second):
			t.Fatal("independent tasks must start concurrently, before either task completes")
		}
	}
	cancel()
	<-done
}
