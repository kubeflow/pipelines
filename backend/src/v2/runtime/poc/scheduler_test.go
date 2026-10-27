package poc

import (
	"context"
	"fmt"
	"strings"
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

type rescanExecutor struct {
	fakeExecutorLifecycle
	dependent chan struct{}
	release   chan struct{}
}

func (*rescanExecutor) Type() string { return ExecutorDocker }
func (*rescanExecutor) Submit(_ context.Context, request TaskExecutionRequest) (ExecutionReference, error) {
	return fakeSubmit(request)
}
func (e *rescanExecutor) Wait(ctx context.Context, reference ExecutionReference) (TaskExecutionResult, error) {
	var request TaskExecutionRequest
	if err := fromJSONData(reference.Data, &request); err != nil {
		return TaskExecutionResult{}, err
	}
	switch request.TaskName {
	case "a-dependent":
		close(e.dependent)
	case "b-fast":
	default:
		select {
		case <-e.release:
		case <-ctx.Done():
			return TaskExecutionResult{}, ctx.Err()
		}
	}
	return TaskExecutionResult{Outcome: TaskOutcomeSuccess}, nil
}

type schedulingGateStore struct {
	storage.RuntimeStoreInterface
	tasks  storage.TaskStoreInterface
	gateID string
	fastID string
}

func (s schedulingGateStore) Get(id string) (*model.RuntimeTask, error) {
	if id == s.gateID {
		deadline := time.Now().Add(2 * time.Second)
		for {
			task, err := s.tasks.GetTask(s.fastID)
			if err != nil {
				return nil, err
			}
			if task.State == model.TaskStatus(apiv2.PipelineTask_SUCCEEDED) {
				break
			}
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("fast task did not finalize before scheduling gate")
			}
			time.Sleep(time.Millisecond)
		}
	}
	return s.RuntimeStoreInterface.Get(id)
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

func TestBoundedSchedulingRescansEarlierEligibilityBeforeWaiting(t *testing.T) {
	coordinator, item, preparation := attemptFixture(t)
	var definitions []string
	definitions = append(definitions,
		`"a-dependent":{"componentRef":{"name":"component"},"dependentTasks":["b-fast"]}`,
		`"b-fast":{"componentRef":{"name":"component"}}`)
	for index := 0; index < schedulingBatchSize-3; index++ {
		definitions = append(definitions, fmt.Sprintf(`"c-blocked-%03d":{"componentRef":{"name":"component"},"dependentTasks":["z-slow"]}`, index))
	}
	definitions = append(definitions,
		`"m-probe":{"componentRef":{"name":"component"}}`,
		`"z-slow":{"componentRef":{"name":"component"}}`)
	spec := &pipelinespec.PipelineSpec{}
	require.NoError(t, protojson.Unmarshal([]byte(`{
		"root":{"dag":{"tasks":{`+strings.Join(definitions, ",")+`}}},
		"components":{"component":{"executorLabel":"executor"}},
		"deploymentSpec":{"executors":{"executor":{"container":{"image":"test"}}}}
	}`), spec))
	manifest := NewRuntimeManifest(ExecutorDocker)
	require.NoError(t, coordinator.materializeComponent(item.run, spec, spec.Root, "root", preparation.Task.UUID, manifest))
	fast, ok := manifest.taskByScope("root.b-fast")
	require.True(t, ok)
	gate, ok := manifest.taskByScope("root.m-probe")
	require.True(t, ok)
	coordinator.runtimeStore = schedulingGateStore{
		RuntimeStoreInterface: coordinator.runtimeStore, tasks: coordinator.taskStore,
		gateID: gate.TaskID, fastID: fast.TaskID,
	}
	executor := &rescanExecutor{dependent: make(chan struct{}), release: make(chan struct{})}
	item.spec, item.runtimeManifest = spec, manifest
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- coordinator.executeDagComponent(ctx, item, executor, spec.Root, "root", nil, false) }()
	progressed := false
	select {
	case <-executor.dependent:
		progressed = true
	case <-time.After(2 * time.Second):
	}
	close(executor.release)
	require.NoError(t, <-done)
	require.True(t, progressed, "a task enabled between bounded passes must start before unrelated long-running tasks finish")
}
