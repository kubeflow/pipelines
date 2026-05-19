package poc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	stdtime "time"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/template"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	compiler "github.com/kubeflow/pipelines/backend/src/v2/compiler"
	drivercommon "github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

type fakeKubernetesCoreWithClientset struct {
	clientSet *kubernetesfake.Clientset
}

func (f *fakeKubernetesCoreWithClientset) PodClient(namespace string) corev1client.PodInterface {
	return f.clientSet.CoreV1().Pods(namespace)
}

func (f *fakeKubernetesCoreWithClientset) GetClientSet() kubernetes.Interface {
	return f.clientSet
}

type fakeDockerExecutor struct {
	inputsByTask          map[string]map[string]string
	parameterValuesByTask map[string]map[string]*structpb.Value
	callOrder             []string
}

func (f *fakeDockerExecutor) Type() string {
	return ExecutorDocker
}

func (f *fakeDockerExecutor) PrepareRun(_ context.Context, run *model.Run, _ *pipelinespec.PipelineJob, workspace *WorkspaceHandle) (map[string]string, error) {
	return map[string]string{"workspacePath": workspace.Metadata["path"], "run": run.UUID}, nil
}

func (f *fakeDockerExecutor) ExecuteTask(_ context.Context, request *TaskExecutionRequest) (*TaskExecutionResult, error) {
	if f.inputsByTask == nil {
		f.inputsByTask = map[string]map[string]string{}
	}
	if f.parameterValuesByTask == nil {
		f.parameterValuesByTask = map[string]map[string]*structpb.Value{}
	}
	taskInputs := map[string]string{}
	taskParameterValues := map[string]*structpb.Value{}
	for key, value := range request.ExecutorInput.GetInputs().GetParameterValues() {
		taskParameterValues[key] = value
		if value.GetStringValue() != "" {
			taskInputs[key] = value.GetStringValue()
			continue
		}
		bytes, err := value.MarshalJSON()
		if err != nil {
			return nil, err
		}
		taskInputs[key] = string(bytes)
	}
	f.inputsByTask[request.Task.Name] = taskInputs
	f.parameterValuesByTask[request.Task.Name] = taskParameterValues
	f.callOrder = append(f.callOrder, request.Task.Name)

	result := &TaskExecutionResult{
		StatusMetadata:   model.JSONData{"executor": "fake-docker"},
		OutputParameters: map[string]*structpb.Value{},
	}
	switch request.Task.Name {
	case "create-dataset":
		result.OutputParameters["output_parameter_path"] = structpb.NewStringValue("10.0")
	case "core-comp":
		result.OutputParameters["Output"] = structpb.NewNumberValue(1)
	case "double":
		result.OutputParameters["Output"] = structpb.NewNumberValue(
			request.ExecutorInput.GetInputs().GetParameterValues()["num"].GetNumberValue() * 2,
		)
	case "process-dataset":
		result.OutputParameters["output_int"] = structpb.NewNumberValue(100)
	}
	return result, nil
}

func TestRetrySQLiteLockedErrorRetriesTransientLocks(t *testing.T) {
	originalDelay := sqliteLockRetryDelay
	sqliteLockRetryDelay = 0
	t.Cleanup(func() {
		sqliteLockRetryDelay = originalDelay
	})

	attempts := 0
	err := retrySQLiteLockedError(func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("database is locked")
		}
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetrySQLiteLockedErrorReturnsNonLockErrorsImmediately(t *testing.T) {
	attempts := 0
	expectedErr := fmt.Errorf("boom")
	err := retrySQLiteLockedError(func() error {
		attempts++
		return expectedErr
	})
	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, attempts)
}

type fakeArtifactExecutor struct {
	callOrder              []string
	inputArtifactCounts    map[string]int
	outputArtifactPresence map[string]bool
}

func (f *fakeArtifactExecutor) Type() string {
	return ExecutorDocker
}

func (f *fakeArtifactExecutor) PrepareRun(_ context.Context, run *model.Run, _ *pipelinespec.PipelineJob, workspace *WorkspaceHandle) (map[string]string, error) {
	return map[string]string{"workspacePath": workspace.Metadata["path"], "run": run.UUID}, nil
}

func (f *fakeArtifactExecutor) ExecuteTask(_ context.Context, request *TaskExecutionRequest) (*TaskExecutionResult, error) {
	if f.inputArtifactCounts == nil {
		f.inputArtifactCounts = map[string]int{}
	}
	if f.outputArtifactPresence == nil {
		f.outputArtifactPresence = map[string]bool{}
	}
	f.callOrder = append(f.callOrder, request.Task.Name)
	f.inputArtifactCounts[request.Task.Name] = len(request.ExecutorInput.GetInputs().GetArtifacts())
	f.outputArtifactPresence[request.Task.Name] = len(request.ExecutorInput.GetOutputs().GetArtifacts()) > 0

	for _, artifactList := range request.ExecutorInput.GetOutputs().GetArtifacts() {
		for _, artifact := range artifactList.Artifacts {
			path, err := withArtifactLocalPath(request.ArtifactLocalRoot, func() (string, error) {
				return localPathForRuntimeArtifact(artifact)
			})
			if err != nil {
				return nil, err
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(path, []byte("artifact-payload"), 0o644); err != nil {
				return nil, err
			}
		}
	}
	return &TaskExecutionResult{
		StatusMetadata:   model.JSONData{"executor": "fake-artifact"},
		OutputParameters: map[string]*structpb.Value{},
	}, nil
}

type blockingExecutor struct {
	started chan string
}

func (b *blockingExecutor) Type() string {
	return ExecutorDocker
}

func (b *blockingExecutor) PrepareRun(_ context.Context, run *model.Run, _ *pipelinespec.PipelineJob, workspace *WorkspaceHandle) (map[string]string, error) {
	return map[string]string{"workspacePath": workspace.Metadata["path"], "run": run.UUID}, nil
}

func (b *blockingExecutor) ExecuteTask(ctx context.Context, request *TaskExecutionRequest) (*TaskExecutionResult, error) {
	if b.started != nil && request != nil && request.Task != nil {
		b.started <- request.Task.UUID
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCoordinatorExecutesDependentTasks(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	fakeDB.SetMaxOpenConns(1)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)
	fakeExecutor := &fakeDockerExecutor{}
	coordinator.RegisterExecutor(fakeExecutor)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "driver", "test_data", "taskOutputParameter_test.yaml")
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:           "run-test",
		DisplayName:    "run-test",
		K8SName:        "run-test",
		Namespace:      "ns1",
		StorageState:   model.StorageStateAvailable,
		ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Pending",
			State:            model.RuntimeStatePending,
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)

	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	require.NoError(t, err)
	require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))

	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	require.NoError(t, err)
	spec, err := compiler.GetPipelineSpec(job)
	require.NoError(t, err)

	err = coordinator.executeRun(context.Background(), &queuedRun{
		run:             run,
		job:             job,
		spec:            spec,
		runtimeManifest: manifest,
		workspace: &WorkspaceHandle{
			Type: WorkspaceTypeHostPath,
			Metadata: map[string]string{
				"path": filepath.Join(WorkspaceRoot(), run.UUID, "workspace"),
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"create-dataset", "process-dataset", "analyze-artifact"}, fakeExecutor.callOrder)
	assert.Equal(t, "10.0", fakeExecutor.inputsByTask["process-dataset"]["input_dataset"])

	persistedRun, err := runStore.GetRun(run.UUID, true)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateSucceeded, persistedRun.State)
	_, statErr := os.Stat(filepath.Join(WorkspaceRoot(), run.UUID, "workspace"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestPropagateDagOutputs_SkipsExistingArtifactLinks(t *testing.T) {
	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	fakeDB.SetMaxOpenConns(1)
	time := util.NewFakeTimeForEpoch()
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(nil, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)

	run := &model.Run{
		UUID:      "run-test",
		Namespace: "ns1",
	}
	runtimeTask := RuntimeTask{
		TaskID:    "core-task",
		Name:      "core",
		ScopePath: "root.mantle.core",
	}
	producerRuntimeTask := RuntimeTask{
		TaskID:    "core-comp-task",
		Name:      "core-comp",
		ScopePath: "root.mantle.core.core-comp",
	}

	_, err = taskStore.CreateTask(&model.Task{
		UUID:      runtimeTask.TaskID,
		RunUUID:   run.UUID,
		Namespace: run.Namespace,
		Name:      runtimeTask.Name,
		Type:      model.TaskType(apiv2beta1.PipelineTaskDetail_DAG),
		ScopePath: runtimeTask.ScopePath,
	})
	require.NoError(t, err)
	_, err = taskStore.CreateTask(&model.Task{
		UUID:      producerRuntimeTask.TaskID,
		RunUUID:   run.UUID,
		Namespace: run.Namespace,
		Name:      producerRuntimeTask.Name,
		Type:      model.TaskType(apiv2beta1.PipelineTaskDetail_RUNTIME),
		ScopePath: producerRuntimeTask.ScopePath,
	})
	require.NoError(t, err)

	createdArtifact, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: run.Namespace,
		Name:      "dataset",
	})
	require.NoError(t, err)
	_, err = artifactTaskStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  createdArtifact.UUID,
		TaskID:      producerRuntimeTask.TaskID,
		RunUUID:     run.UUID,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "dataset",
		Producer: model.JSONData{
			"task_name": producerRuntimeTask.Name,
		},
	})
	require.NoError(t, err)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(
		filepath.Dir(thisFile),
		"..", "..", "..", "..", "..",
		"test_data", "sdk_compiled_pipelines", "valid", "critical", "artifact_cache.yaml",
	)
	manifestBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	spec, _, err := util.LoadPipelineAndPlatformSpecBytes(manifestBytes)
	require.NoError(t, err)
	componentSpec := spec.GetComponents()["comp-core"]
	require.NotNil(t, componentSpec)

	item := &queuedRun{
		run: run,
		runtimeManifest: &RuntimeManifest{
			Tasks: []RuntimeTask{runtimeTask, producerRuntimeTask},
		},
	}

	require.NoError(t, coordinator.propagateDagOutputs(item, runtimeTask, componentSpec, runtimeTask.ScopePath))
	require.NoError(t, coordinator.propagateDagOutputs(item, runtimeTask, componentSpec, runtimeTask.ScopePath))

	artifactTasks, _, _, err := artifactTaskStore.ListArtifactTasks(
		[]*model.FilterContext{
			{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: runtimeTask.TaskID}},
			{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
		},
		nil,
		list.EmptyOptions(),
	)
	require.NoError(t, err)
	require.Len(t, artifactTasks, 1)
	assert.Equal(t, createdArtifact.UUID, artifactTasks[0].ArtifactID)
	assert.Equal(t, "Output", artifactTasks[0].ArtifactKey)
}

func TestApplyCachedTask_SkipsExistingArtifactLinks(t *testing.T) {
	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(nil, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)

	run := &model.Run{UUID: "run-test", Namespace: "ns1"}
	cachedTask, err := taskStore.CreateTask(&model.Task{
		UUID:      "cached-task",
		RunUUID:   "previous-run",
		Namespace: run.Namespace,
		Name:      "print-text-2",
		Type:      model.TaskType(apiv2beta1.PipelineTaskDetail_RUNTIME),
	})
	require.NoError(t, err)
	currentTask, err := taskStore.CreateTask(&model.Task{
		UUID:      "current-task",
		RunUUID:   run.UUID,
		Namespace: run.Namespace,
		Name:      "print-text-3",
		Type:      model.TaskType(apiv2beta1.PipelineTaskDetail_RUNTIME),
	})
	require.NoError(t, err)
	createdArtifact, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: run.Namespace,
		Name:      "executor-logs",
	})
	require.NoError(t, err)
	_, err = artifactTaskStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  createdArtifact.UUID,
		TaskID:      cachedTask.UUID,
		RunUUID:     cachedTask.RunUUID,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "executor-logs",
		Producer:    model.JSONData{"task_name": cachedTask.Name},
	})
	require.NoError(t, err)

	require.NoError(t, coordinator.applyCachedTask(run, currentTask.UUID, &pipelinespec.PipelineTaskSpec{}, cachedTask))
	require.NoError(t, coordinator.applyCachedTask(run, currentTask.UUID, &pipelinespec.PipelineTaskSpec{}, cachedTask))

	artifactTasks, _, _, err := artifactTaskStore.ListArtifactTasks(
		[]*model.FilterContext{
			{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: currentTask.UUID}},
			{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
		},
		nil,
		list.EmptyOptions(),
	)
	require.NoError(t, err)
	require.Len(t, artifactTasks, 1)
	assert.Equal(t, createdArtifact.UUID, artifactTasks[0].ArtifactID)
	assert.Equal(t, "executor-logs", artifactTasks[0].ArtifactKey)
}

func TestCoordinatorCollectsLoopOutputsForDownstreamTasks(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)
	fakeExecutor := &fakeDockerExecutor{}
	coordinator.RegisterExecutor(fakeExecutor)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(
		filepath.Dir(thisFile),
		"..", "..", "..", "..", "..",
		"test_data", "sdk_compiled_pipelines", "valid", "critical", "parameters_simple.yaml",
	)
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:           "parameters-simple-run",
		DisplayName:    "parameters-simple-run",
		K8SName:        "parameters-simple-run",
		Namespace:      "ns1",
		StorageState:   model.StorageStateAvailable,
		ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Pending",
			State:            model.RuntimeStatePending,
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)

	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	require.NoError(t, err)
	require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))

	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	require.NoError(t, err)
	spec, err := compiler.GetPipelineSpec(job)
	require.NoError(t, err)

	err = coordinator.executeRun(context.Background(), &queuedRun{
		run:             run,
		job:             job,
		spec:            spec,
		runtimeManifest: manifest,
		workspace: &WorkspaceHandle{
			Type: WorkspaceTypeHostPath,
			Metadata: map[string]string{
				"path": filepath.Join(WorkspaceRoot(), run.UUID, "workspace"),
			},
		},
	})
	require.NoError(t, err)

	addInputs := fakeExecutor.parameterValuesByTask["add"]
	require.NotNil(t, addInputs)
	addNums := addInputs["nums"].GetListValue()
	require.NotNil(t, addNums)
	require.Len(t, addNums.GetValues(), 3)
	assert.Equal(t, float64(2), addNums.GetValues()[0].GetNumberValue())
	assert.Equal(t, float64(4), addNums.GetValues()[1].GetNumberValue())
	assert.Equal(t, float64(6), addNums.GetValues()[2].GetNumberValue())

	addContainerInputs := fakeExecutor.parameterValuesByTask["add-container"]
	require.NotNil(t, addContainerInputs)
	addContainerNums := addContainerInputs["nums"].GetListValue()
	require.NotNil(t, addContainerNums)
	require.Len(t, addContainerNums.GetValues(), 3)
	assert.Equal(t, float64(2), addContainerNums.GetValues()[0].GetNumberValue())
	assert.Equal(t, float64(4), addContainerNums.GetValues()[1].GetNumberValue())
	assert.Equal(t, float64(6), addContainerNums.GetValues()[2].GetNumberValue())
}

func TestCoordinatorPrefersDagTaskOutputForMixedParameters(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)
	fakeExecutor := &fakeDockerExecutor{}
	coordinator.RegisterExecutor(fakeExecutor)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(
		filepath.Dir(thisFile),
		"..", "..", "..", "..", "..",
		"test_data", "sdk_compiled_pipelines", "valid", "critical", "mixed_parameters.yaml",
	)
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:           "mixed-parameters-run",
		DisplayName:    "mixed-parameters-run",
		K8SName:        "mixed-parameters-run",
		Namespace:      "ns1",
		StorageState:   model.StorageStateAvailable,
		ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Pending",
			State:            model.RuntimeStatePending,
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)

	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	require.NoError(t, err)
	require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))

	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	require.NoError(t, err)
	spec, err := compiler.GetPipelineSpec(job)
	require.NoError(t, err)

	err = coordinator.executeRun(context.Background(), &queuedRun{
		run:             run,
		job:             job,
		spec:            spec,
		runtimeManifest: manifest,
		workspace: &WorkspaceHandle{
			Type: WorkspaceTypeHostPath,
			Metadata: map[string]string{
				"path": filepath.Join(WorkspaceRoot(), run.UUID, "workspace"),
			},
		},
	})
	require.NoError(t, err)

	crustInputs := fakeExecutor.parameterValuesByTask["crust-comp"]
	require.NotNil(t, crustInputs)
	assert.Equal(t, float64(2), crustInputs["x"].GetNumberValue())
	assert.Equal(t, float64(1), crustInputs["y"].GetNumberValue())
}

func TestShouldDownloadImporterArtifactToWorkspace(t *testing.T) {
	assert.False(t, shouldDownloadImporterArtifactToWorkspace(nil))
	assert.False(t, shouldDownloadImporterArtifactToWorkspace(&WorkspaceHandle{Type: WorkspaceTypePVC}))
	assert.True(t, shouldDownloadImporterArtifactToWorkspace(&WorkspaceHandle{Type: WorkspaceTypeHostPath}))
}

func TestKubernetesExecutorRecreatesTerminalPodForRepeatedTask(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	clientSet := kubernetesfake.NewSimpleClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "double-task-id", Namespace: "kubeflow"},
		Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
	})
	executor := kubernetesExecutor{coreClient: &fakeKubernetesCoreWithClientset{clientSet: clientSet}}

	request := &TaskExecutionRequest{
		Run: &model.Run{
			UUID:           "run-id",
			K8SName:        "run-name",
			Namespace:      "kubeflow",
			DisplayName:    "run-display-name",
			ServiceAccount: "pipeline-runner",
		},
		Task: &model.Task{
			UUID:        "task-id",
			Name:        "double",
			DisplayName: "double",
			State:       model.TaskStatus(apiv2beta1.PipelineTaskDetail_SUCCEEDED),
			Pods:        model.JSONSlice{model.JSONData{"name": "double-task-id", "type": "EXECUTOR"}},
		},
		Container:         &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"},
		ExecutorInput:     &pipelinespec.ExecutorInput{},
		ComponentSpec:     &pipelinespec.ComponentSpec{},
		TaskSpec:          &pipelinespec.PipelineTaskSpec{},
		PipelineName:      "pipeline",
		TaskDisplayName:   "double",
		ArtifactLocalRoot: "/tmp/kfp-artifacts",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*stdtime.Millisecond)
	defer cancel()
	_, err := executor.ExecuteTask(ctx, request)
	require.Error(t, err)

	pod, getErr := clientSet.CoreV1().Pods("kubeflow").Get(context.Background(), "double-task-id", metav1.GetOptions{})
	require.NoError(t, getErr)
	assert.Equal(t, "double-task-id", pod.Name)
}

func TestReconcileManagedRunsQueuesManagedRun(t *testing.T) {
	clientSet := kubernetesfake.NewSimpleClientset()
	viper.Set(common.V2RuntimeExecutor, ExecutorKubernetes)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(
		runStore,
		taskStore,
		artifactStore,
		artifactTaskStore,
		time,
		&fakeKubernetesCoreWithClientset{clientSet: clientSet},
		nil,
		false,
	)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "driver", "test_data", "taskOutputParameter_test.yaml")
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:           "run-requeue",
		DisplayName:    "run-requeue",
		K8SName:        "run-requeue",
		Namespace:      "ns1",
		StorageState:   model.StorageStateAvailable,
		ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
			State:            model.RuntimeStateRunning,
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)
	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	require.NoError(t, err)
	require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))

	coordinator = NewCoordinator(
		runStore,
		taskStore,
		artifactStore,
		artifactTaskStore,
		time,
		&fakeKubernetesCoreWithClientset{clientSet: clientSet},
		nil,
		false,
	)
	require.NoError(t, coordinator.ReconcileManagedRuns(context.Background()))
	assert.Equal(t, 1, len(coordinator.queue))

	persistedRun, err := runStore.GetRun(run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateRunning, persistedRun.State)
}

func TestReconcileManagedRunsFailsRunningLocalDockerRun(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	fakeDB.SetMaxOpenConns(1)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "driver", "test_data", "taskOutputParameter_test.yaml")
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:           "run-restart-fail",
		DisplayName:    "run-restart-fail",
		K8SName:        "run-restart-fail",
		Namespace:      "ns1",
		StorageState:   model.StorageStateAvailable,
		ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Running",
			State:            model.RuntimeStateRunning,
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)
	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	require.NoError(t, err)
	require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))

	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	require.NoError(t, err)
	workspacePath := manifest.Workspace.Metadata["path"]

	coordinator = NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)
	require.NoError(t, coordinator.ReconcileManagedRuns(context.Background()))
	assert.Equal(t, 0, len(coordinator.queue))

	persistedRun, err := runStore.GetRun(run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateFailed, persistedRun.State)

	tasks, _, _, err := taskStore.ListTasks(&model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID},
	}, list.EmptyOptions())
	require.NoError(t, err)
	for _, task := range tasks {
		assert.Equal(t, apiv2beta1.PipelineTaskDetail_FAILED, apiv2beta1.PipelineTaskDetail_TaskState(task.State))
	}

	_, statErr := os.Stat(workspacePath)
	assert.True(t, os.IsNotExist(statErr))
}

func TestReconcileManagedRunsCancellingRunMarksCanceled(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "true")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)

	manifestJSON, err := NewRuntimeManifest(ExecutorDocker).ToJSON()
	require.NoError(t, err)
	run := &model.Run{
		UUID:         "run-cancelling",
		DisplayName:  "run-cancelling",
		K8SName:      "run-cancelling",
		Namespace:    "ns1",
		StorageState: model.StorageStateAvailable,
		RunDetails: model.RunDetails{
			CreatedAtInSec:          1,
			ScheduledAtInSec:        1,
			Conditions:              "Terminating",
			State:                   model.RuntimeStateCancelling,
			PipelineRuntimeManifest: model.LargeText(manifestJSON),
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)

	task, err := taskStore.CreateTask(&model.Task{
		UUID:           "task-cancelling",
		Namespace:      run.Namespace,
		RunUUID:        run.UUID,
		Name:           "task",
		DisplayName:    "task",
		State:          model.TaskStatus(apiv2beta1.PipelineTaskDetail_RUNNING),
		Type:           model.TaskType(apiv2beta1.PipelineTaskDetail_RUNTIME),
		ScopePath:      "root.task",
		CreatedAtInSec: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, task)

	require.NoError(t, coordinator.ReconcileManagedRuns(context.Background()))
	assert.Equal(t, 0, len(coordinator.queue))

	persistedRun, err := runStore.GetRun(run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateCanceled, persistedRun.State)

	persistedTask, err := taskStore.GetTask(task.UUID)
	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SKIPPED, apiv2beta1.PipelineTaskDetail_TaskState(persistedTask.State))
}

func TestTerminateManagedRunCancelsActiveExecution(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)
	blocking := &blockingExecutor{started: make(chan string, 1)}
	coordinator.RegisterExecutor(blocking)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "driver", "test_data", "taskOutputParameter_test.yaml")
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:           "run-active-cancel",
		DisplayName:    "run-active-cancel",
		K8SName:        "run-active-cancel",
		Namespace:      "ns1",
		StorageState:   model.StorageStateAvailable,
		ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Pending",
			State:            model.RuntimeStatePending,
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)

	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	require.NoError(t, err)
	require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))

	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	require.NoError(t, err)
	spec, err := compiler.GetPipelineSpec(job)
	require.NoError(t, err)

	executeErrCh := make(chan error, 1)
	go func() {
		executeErrCh <- coordinator.executeRun(context.Background(), &queuedRun{
			run:             run,
			job:             job,
			spec:            spec,
			runtimeManifest: manifest,
			workspace: &WorkspaceHandle{
				Type: WorkspaceTypeHostPath,
				Metadata: map[string]string{
					"path": filepath.Join(WorkspaceRoot(), run.UUID, "workspace"),
				},
			},
		})
	}()

	var startedTaskID string
	select {
	case startedTaskID = <-blocking.started:
	case <-stdtime.After(5 * stdtime.Second):
	}
	require.NotEmpty(t, startedTaskID)

	require.NoError(t, runStore.TerminateRun(run.UUID))
	require.NoError(t, coordinator.TerminateManagedRun(context.Background(), run))
	require.NoError(t, <-executeErrCh)

	persistedRun, err := runStore.GetRun(run.UUID, false)
	require.NoError(t, err)
	assert.Equal(t, model.RuntimeStateCanceled, persistedRun.State)

	persistedTask, err := taskStore.GetTask(startedTaskID)
	require.NoError(t, err)
	assert.Equal(t, apiv2beta1.PipelineTaskDetail_SKIPPED, apiv2beta1.PipelineTaskDetail_TaskState(persistedTask.State))
}

func TestCoordinatorRunLoopExecutesRunsConcurrently(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	fakeDB.SetMaxOpenConns(1)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)
	blocking := &blockingExecutor{started: make(chan string, 2)}
	coordinator.RegisterExecutor(blocking)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "driver", "test_data", "taskOutputParameter_test.yaml")
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	runLoopCtx, cancelRunLoop := context.WithCancel(context.Background())
	defer cancelRunLoop()
	go coordinator.runLoop(runLoopCtx)

	createRun := func(runID string) *model.Run {
		run := &model.Run{
			UUID:           runID,
			DisplayName:    runID,
			K8SName:        runID,
			Namespace:      "ns1",
			StorageState:   model.StorageStateAvailable,
			ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
			PipelineSpec: model.PipelineSpec{
				PipelineSpecManifest: model.LargeText(string(templateBytes)),
				RuntimeConfig: model.RuntimeConfig{
					Parameters: "{}",
				},
			},
			RunDetails: model.RunDetails{
				CreatedAtInSec:   1,
				ScheduledAtInSec: 1,
				Conditions:       "Pending",
				State:            model.RuntimeStatePending,
			},
		}
		_, err := runStore.CreateRun(run)
		require.NoError(t, err)
		job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
		require.NoError(t, err)
		require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))
		return run
	}

	runOne := createRun("run-concurrent-1")
	runTwo := createRun("run-concurrent-2")

	startedTasks := map[string]struct{}{}
	deadline := stdtime.After(2 * stdtime.Second)
	for len(startedTasks) < 2 {
		select {
		case taskID := <-blocking.started:
			startedTasks[taskID] = struct{}{}
		case <-deadline:
			t.Fatalf("timed out waiting for both coordinator runs to start; started=%v", startedTasks)
		}
	}
	assert.Len(t, startedTasks, 2)

	cancelRunLoop()
	require.Eventually(t, func() bool {
		runOneState, err := runStore.GetRun(runOne.UUID, false)
		if err != nil {
			return false
		}
		runTwoState, err := runStore.GetRun(runTwo.UUID, false)
		if err != nil {
			return false
		}
		return runOneState.State == model.RuntimeStateCanceled && runTwoState.State == model.RuntimeStateCanceled
	}, stdtime.Second, 20*stdtime.Millisecond)
}

func TestCreatePVCPlatformTaskRejectsInvalidSize(t *testing.T) {
	clientSet := kubernetesfake.NewSimpleClientset()
	coordinator := NewCoordinator(nil, nil, nil, nil, util.NewFakeTimeForEpoch(), &fakeKubernetesCoreWithClientset{clientSet: clientSet}, nil, false)

	_, err := coordinator.createPVCPlatformTask(
		context.Background(),
		&model.Run{UUID: "run-pvc", Namespace: "kubeflow"},
		&model.Task{UUID: "task-pvc"},
		&pipelinespec.ExecutorInput{
			Inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"access_modes": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{structpb.NewStringValue("ReadWriteOnce")},
					}),
					"size": structpb.NewStringValue("not-a-size"),
				},
			},
		},
		&pipelinespec.PipelineTaskSpec{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid size")

	pvcs, listErr := clientSet.CoreV1().PersistentVolumeClaims("kubeflow").List(context.Background(), metav1.ListOptions{})
	require.NoError(t, listErr)
	assert.Empty(t, pvcs.Items)
}

func TestCreatePVCPlatformTaskUsesClusterDefaultStorageClass(t *testing.T) {
	clientSet := kubernetesfake.NewSimpleClientset()
	coordinator := NewCoordinator(nil, nil, nil, nil, util.NewFakeTimeForEpoch(), &fakeKubernetesCoreWithClientset{clientSet: clientSet}, nil, false)

	result, err := coordinator.createPVCPlatformTask(
		context.Background(),
		&model.Run{UUID: "run-pvc-default-sc", Namespace: "kubeflow"},
		&model.Task{UUID: "task-pvc-default-sc"},
		&pipelinespec.ExecutorInput{
			Inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"access_modes": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{structpb.NewStringValue("ReadWriteOnce")},
					}),
					"size": structpb.NewStringValue("1Gi"),
				},
			},
		},
		&pipelinespec.PipelineTaskSpec{},
	)
	require.NoError(t, err)

	pvcName := result.OutputParameters["name"].GetStringValue()
	pvc, err := clientSet.CoreV1().PersistentVolumeClaims("kubeflow").Get(context.Background(), pvcName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Nil(t, pvc.Spec.StorageClassName)
}

func TestCoordinatorPublishesAndResolvesArtifacts(t *testing.T) {
	viper.Set(common.V2RuntimeExecutor, ExecutorDocker)
	viper.Set(common.V2RuntimeAutoExecute, "false")
	t.Cleanup(func() {
		viper.Set(common.V2RuntimeExecutor, "")
		viper.Set(common.V2RuntimeAutoExecute, "false")
	})

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	runStore := storage.NewRunStore(fakeDB, time)
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(runStore, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)
	fakeExecutor := &fakeArtifactExecutor{}
	coordinator.RegisterExecutor(fakeExecutor)

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "driver", "test_data", "taskOutputArtifact_test.yaml")
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)
	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:           "artifact-run",
		DisplayName:    "artifact-run",
		K8SName:        "artifact-run",
		Namespace:      "ns1",
		StorageState:   model.StorageStateAvailable,
		ServiceAccount: common.DefaultPipelineRunnerServiceAccount,
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "Pending",
			State:            model.RuntimeStatePending,
		},
	}
	_, err = runStore.CreateRun(run)
	require.NoError(t, err)

	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{RunID: run.UUID, RunAt: run.CreatedAtInSec})
	require.NoError(t, err)
	require.NoError(t, coordinator.SubmitRun(context.Background(), run, job, nil))

	manifest, err := ParseManifest(string(run.PipelineRuntimeManifest))
	require.NoError(t, err)
	spec, err := compiler.GetPipelineSpec(job)
	require.NoError(t, err)

	err = coordinator.executeRun(context.Background(), &queuedRun{
		run:             run,
		job:             job,
		spec:            spec,
		runtimeManifest: manifest,
		workspace: &WorkspaceHandle{
			Type: WorkspaceTypeHostPath,
			Metadata: map[string]string{
				"path": filepath.Join(WorkspaceRoot(), run.UUID, "workspace"),
			},
		},
	})
	require.NoError(t, err)

	assert.Contains(t, fakeExecutor.callOrder, "create-dataset")
	assert.Contains(t, fakeExecutor.callOrder, "process-dataset")
	assert.GreaterOrEqual(t, fakeExecutor.inputArtifactCounts["process-dataset"], 1)

	artifactTasks, _, _, err := artifactTaskStore.ListArtifactTasks([]*model.FilterContext{
		{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
	}, nil, list.EmptyOptions())
	require.NoError(t, err)
	assert.NotEmpty(t, artifactTasks)
}

func TestCoordinatorPublishesMetadataOnlyArtifacts(t *testing.T) {
	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(nil, nil, artifactStore, artifactTaskStore, time, nil, nil, false)

	outputDir := t.TempDir()
	outputFile := filepath.Join(outputDir, "output_metadata.json")
	metadata, err := structpb.NewStruct(map[string]any{
		"confusionMatrix": map[string]any{
			"annotationSpecs": []any{
				map[string]any{"displayName": "cat"},
				map[string]any{"displayName": "dog"},
			},
			"rows": []any{
				map[string]any{"row": []any{1.0, 2.0}},
				map[string]any{"row": []any{3.0, 4.0}},
			},
		},
	})
	require.NoError(t, err)

	artifactType := &pipelinespec.ArtifactTypeSchema{
		Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.ClassificationMetrics"},
	}
	executorOutput := &pipelinespec.ExecutorOutput{
		Artifacts: map[string]*pipelinespec.ArtifactList{
			"metrics": {
				Artifacts: []*pipelinespec.RuntimeArtifact{{
					Name:     "metrics",
					Type:     artifactType,
					Uri:      "minio://mlpipeline/v2/artifacts/run/train/metrics",
					Metadata: metadata,
				}},
			},
		},
	}
	outputBytes, err := protojson.Marshal(executorOutput)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(outputFile, outputBytes, 0o644))

	executorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"metrics": {
					Artifacts: []*pipelinespec.RuntimeArtifact{{
						Name: "metrics",
						Type: artifactType,
						Uri:  "minio://mlpipeline/v2/artifacts/run/train/metrics",
					}},
				},
			},
			OutputFile: outputFile,
		},
	}

	run := &model.Run{
		UUID:      "run-metadata-only",
		Namespace: "ns1",
	}
	err = coordinator.publishOutputArtifacts(context.Background(), run, "task-1", "train-model", executorInput)
	require.NoError(t, err)

	artifactTasks, _, _, err := artifactTaskStore.ListArtifactTasks([]*model.FilterContext{
		{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
	}, nil, list.EmptyOptions())
	require.NoError(t, err)
	require.Len(t, artifactTasks, 1)

	artifact, err := artifactStore.GetArtifact(artifactTasks[0].ArtifactID)
	require.NoError(t, err)
	require.NotNil(t, artifact.Metadata)
	assert.Contains(t, artifact.Metadata, "confusionMatrix")
}

func TestCoordinatorPublishesOCIArtifactsWithoutLocalFile(t *testing.T) {
	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(nil, nil, artifactStore, artifactTaskStore, time, nil, nil, false)

	executorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"model": {
					Artifacts: []*pipelinespec.RuntimeArtifact{{
						Name: "model",
						Type: &pipelinespec.ArtifactTypeSchema{
							Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Model"},
						},
						Uri: "oci://registry.domain.local/org/repo:v1.0",
					}},
				},
			},
		},
	}

	run := &model.Run{
		UUID:      "run-oci-model",
		Namespace: "ns1",
	}
	err = coordinator.publishOutputArtifacts(context.Background(), run, "task-1", "build-model-car", executorInput)
	require.NoError(t, err)

	artifactTasks, _, _, err := artifactTaskStore.ListArtifactTasks([]*model.FilterContext{
		{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
	}, nil, list.EmptyOptions())
	require.NoError(t, err)
	require.Len(t, artifactTasks, 1)

	artifact, err := artifactStore.GetArtifact(artifactTasks[0].ArtifactID)
	require.NoError(t, err)
	require.NotNil(t, artifact.URI)
	assert.Equal(t, "oci://registry.domain.local/org/repo:v1.0", *artifact.URI)
}

func TestCoordinatorPublishesCustomPathArtifacts(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(nil, nil, artifactStore, artifactTaskStore, time, nil, nil, false)

	customPath := "/tmp/out_dataset"
	run := &model.Run{
		UUID:      "run-custom-path",
		Namespace: "ns1",
	}
	artifactLocalRoot := coordinator.artifactLocalRoot(run, ExecutorDocker)
	hostArtifactPath := filepath.Join(dockerTempMountSourceForArtifactRoot(artifactLocalRoot), "out_dataset")
	require.NoError(t, os.MkdirAll(filepath.Dir(hostArtifactPath), 0o755))
	require.NoError(t, os.WriteFile(hostArtifactPath, []byte("Hello, World!"), 0o644))

	executorInput := &pipelinespec.ExecutorInput{
		Outputs: &pipelinespec.ExecutorInput_Outputs{
			Artifacts: map[string]*pipelinespec.ArtifactList{
				"out_dataset": {
					Artifacts: []*pipelinespec.RuntimeArtifact{{
						Name: "out_dataset",
						Type: &pipelinespec.ArtifactTypeSchema{
							Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
							SchemaVersion: "0.0.1",
						},
						CustomPath: &customPath,
					}},
				},
			},
		},
	}

	err = coordinator.publishOutputArtifacts(context.Background(), run, "task-1", "component-output-artifact", executorInput)
	require.NoError(t, err)

	artifactTasks, _, _, err := artifactTaskStore.ListArtifactTasks([]*model.FilterContext{
		{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: run.UUID}},
	}, nil, list.EmptyOptions())
	require.NoError(t, err)
	require.Len(t, artifactTasks, 1)

	artifact, err := artifactStore.GetArtifact(artifactTasks[0].ArtifactID)
	require.NoError(t, err)
	require.NotNil(t, artifact.Metadata)
	assert.Equal(t, customPath, artifact.Metadata[runtimeArtifactCustomPathMetadataKey])

	runtimeArtifact := modelArtifactToRuntimeArtifact(artifact)
	require.NotNil(t, runtimeArtifact)
	require.NotNil(t, runtimeArtifact.CustomPath)
	assert.Equal(t, customPath, runtimeArtifact.GetCustomPath())
}

func TestModelArtifactToRuntimeArtifactPreservesDatasetType(t *testing.T) {
	artifact := &model.Artifact{
		Name: "dataset",
		Type: model.ArtifactType(apiv2beta1.Artifact_Dataset),
		URI:  util.StringPointer("file:///tmp/artifact"),
	}

	runtimeArtifact := modelArtifactToRuntimeArtifact(artifact)
	require.NotNil(t, runtimeArtifact)
	require.NotNil(t, runtimeArtifact.Type)
	assert.Equal(t, "system.Dataset", runtimeArtifact.GetType().GetSchemaTitle())
	assert.Equal(t, "0.0.1", runtimeArtifact.GetType().GetSchemaVersion())
}

func TestMergeRuntimeArtifactsPreservesRemoteURIForLocalExecutorPath(t *testing.T) {
	src := &pipelinespec.RuntimeArtifact{
		Uri: " /tmp/should-not-be-used ",
	}
	dst := &pipelinespec.RuntimeArtifact{
		Uri: "minio://mlpipeline/path/to/artifact",
	}

	src.Uri = strings.TrimSpace(src.Uri)
	mergeRuntimeArtifacts(src, dst)

	require.Equal(t, "minio://mlpipeline/path/to/artifact", dst.Uri)
	require.NotNil(t, dst.CustomPath)
	assert.Equal(t, "/tmp/should-not-be-used", *dst.CustomPath)
}

func TestResolveImporterArtifactURIFromRuntimeParameter(t *testing.T) {
	importerSpec := &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
		ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
			Value: &pipelinespec.ValueOrRuntimeParameter_RuntimeParameter{
				RuntimeParameter: "uri",
			},
		},
	}
	value := structpb.NewStringValue("minio://mlpipeline/input/artifact")
	artifactURI, err := resolveImporterArtifactURI(importerSpec, map[string]*structpb.Value{
		"uri": value,
	})
	require.NoError(t, err)
	assert.Equal(t, "minio://mlpipeline/input/artifact", artifactURI)
}

func TestResolveInputArtifactsMarksImporterWorkspaceArtifacts(t *testing.T) {
	fakeDB, err := storage.NewFakeDB()
	require.NoError(t, err)
	time := util.NewFakeTimeForEpoch()
	taskStore := storage.NewTaskStore(fakeDB, time, util.NewUUIDGenerator())
	artifactStore := storage.NewArtifactStore(fakeDB, time, util.NewUUIDGenerator())
	artifactTaskStore := storage.NewArtifactTaskStore(fakeDB, util.NewUUIDGenerator())
	coordinator := NewCoordinator(nil, taskStore, artifactStore, artifactTaskStore, time, nil, nil, false)

	run := &model.Run{UUID: "run-importer", Namespace: "ns1"}
	producerTask, err := taskStore.CreateTask(&model.Task{
		UUID:           "task-importer",
		RunUUID:        run.UUID,
		Namespace:      run.Namespace,
		Name:           "importer",
		DisplayName:    "importer",
		State:          model.TaskStatus(apiv2beta1.PipelineTaskDetail_SUCCEEDED),
		Type:           model.TaskType(apiv2beta1.PipelineTaskDetail_IMPORTER),
		TypeAttrs:      model.JSONData{"downloadToWorkspace": true},
		CreatedAtInSec: 1,
		ScopePath:      "root.importer",
	})
	require.NoError(t, err)
	createdArtifact, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: run.Namespace,
		Name:      "artifact",
		URI:       util.StringPointer("minio://mlpipeline/path/to/artifact"),
		Metadata: model.JSONData{
			drivercommon.WorkspaceMetadataField: true,
		},
	})
	require.NoError(t, err)
	_, err = artifactTaskStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  createdArtifact.UUID,
		TaskID:      producerTask.UUID,
		RunUUID:     run.UUID,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "artifact",
		Producer:    model.JSONData{"task_name": "importer"},
	})
	require.NoError(t, err)

	artifacts, err := coordinator.resolveInputArtifacts(&queuedRun{
		run: run,
		runtimeManifest: &RuntimeManifest{
			Tasks: []RuntimeTask{{
				TaskID:    producerTask.UUID,
				Name:      "importer",
				ScopePath: "root.importer",
			}},
		},
	}, "root.consumer", &pipelinespec.PipelineTaskSpec{
		Inputs: &pipelinespec.TaskInputsSpec{
			Artifacts: map[string]*pipelinespec.TaskInputsSpec_InputArtifactSpec{
				"dataset": {
					Kind: &pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifact{
						TaskOutputArtifact: &pipelinespec.TaskInputsSpec_InputArtifactSpec_TaskOutputArtifactSpec{
							ProducerTask:      "importer",
							OutputArtifactKey: "artifact",
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Contains(t, artifacts, "dataset")
	require.Len(t, artifacts["dataset"].Artifacts, 1)
	workspaceFlag := artifacts["dataset"].Artifacts[0].GetMetadata().GetFields()[drivercommon.WorkspaceMetadataField]
	require.NotNil(t, workspaceFlag)
	assert.True(t, workspaceFlag.GetBoolValue())
}

func TestGetPipelineRootUsesK8SNameForPathSafety(t *testing.T) {
	coordinator := NewCoordinator(nil, nil, nil, nil, util.NewFakeTimeForEpoch(), nil, nil, false)
	run := &model.Run{
		UUID:        "run-uuid",
		DisplayName: "Display Name With Spaces",
		K8SName:     "display-name-with-spaces",
	}
	pipelineRoot, err := coordinator.getPipelineRoot(run)
	require.NoError(t, err)
	assert.Contains(t, pipelineRoot, "display-name-with-spaces/run-uuid")
	assert.NotContains(t, pipelineRoot, "Display Name With Spaces")
}

func TestCoordinatorMLPipelineServerConfigUsesLocalOverride(t *testing.T) {
	t.Setenv("LOCAL_API_SERVER", "true")
	t.Setenv("KFP_V2_LOCAL_API_SERVER_ADDRESS", "172.19.0.1")
	t.Setenv("KFP_V2_LOCAL_API_SERVER_PORT", "18887")

	serverConfig := coordinatorMLPipelineServerConfig()

	require.NotNil(t, serverConfig)
	assert.Equal(t, "172.19.0.1", serverConfig.Address)
	assert.Equal(t, "18887", serverConfig.Port)
}

func TestDockerProxyEnvArgsUsesConfiguredProxyVariables(t *testing.T) {
	t.Setenv("http_proxy", "http://127.0.0.1:3128")
	t.Setenv("NO_PROXY", "localhost,127.0.0.1")

	args := dockerProxyEnvArgs()

	assert.Contains(t, args, "-e")
	assert.Contains(t, args, "http_proxy=http://127.0.0.1:3128")
	assert.Contains(t, args, "NO_PROXY=localhost,127.0.0.1")
}

func TestBuildCoordinatorExecutorPodAppliesSecretAsEnvFromTaskInputs(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	inputParameters, err := model.ProtoSliceToJSONSlice([]*apiv2beta1.PipelineTaskDetail_InputOutputs_IOParameter{{
		ParameterKey: "some_output",
		Value:        structpb.NewStringValue("test-secret-3"),
		Type:         apiv2beta1.IOType_TASK_OUTPUT_INPUT,
		Producer: &apiv2beta1.IOProducer{
			TaskName: "generate-secret-name",
		},
	}})
	require.NoError(t, err)

	pod, err := buildCoordinatorExecutorPod(&TaskExecutionRequest{
		Run: &model.Run{
			UUID:           "run-id",
			K8SName:        "run-name",
			Namespace:      "kubeflow",
			DisplayName:    "run-display-name",
			ServiceAccount: "pipeline-runner",
		},
		Task: &model.Task{
			UUID:            "task-id",
			Name:            "comp",
			DisplayName:     "comp",
			InputParameters: inputParameters,
		},
		Container: &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"},
		ExecutorInput: &pipelinespec.ExecutorInput{
			Inputs: &pipelinespec.ExecutorInput_Inputs{
				ParameterValues: map[string]*structpb.Value{
					"secret_parm": structpb.NewStringValue("test-secret-1"),
				},
			},
		},
		ComponentSpec: &pipelinespec.ComponentSpec{},
		KubernetesExecutorConfig: &kubernetesplatform.KubernetesExecutorConfig{
			SecretAsEnv: []*kubernetesplatform.SecretAsEnv{
				{
					SecretNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_ComponentInputParameter{
							ComponentInputParameter: "secret_parm",
						},
					},
					KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{{
						SecretKey: "username",
						EnvVar:    "USER_NAME",
					}},
				},
				{
					SecretNameParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec{
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameter{
							TaskOutputParameter: &pipelinespec.TaskInputsSpec_InputParameterSpec_TaskOutputParameterSpec{
								ProducerTask:       "generate-secret-name",
								OutputParameterKey: "some_output",
							},
						},
					},
					KeyToEnv: []*kubernetesplatform.SecretAsEnv_SecretKeyToEnvMap{{
						SecretKey: "password",
						EnvVar:    "PASSWORD_VAR2",
					}},
				},
			},
		},
		PipelineName:      "pipeline-secret-env",
		TaskDisplayName:   "comp",
		ArtifactLocalRoot: "/tmp/kfp-artifacts",
	})
	require.NoError(t, err)
	require.NotNil(t, pod)
	require.NotEmpty(t, pod.Spec.Containers)

	var secretEnv *corev1.EnvVar
	for idx := range pod.Spec.Containers[0].Env {
		env := &pod.Spec.Containers[0].Env[idx]
		if env.Name == "USER_NAME" {
			secretEnv = env
			break
		}
	}
	require.NotNil(t, secretEnv)
	require.NotNil(t, secretEnv.ValueFrom)
	require.NotNil(t, secretEnv.ValueFrom.SecretKeyRef)
	assert.Equal(t, "test-secret-1", secretEnv.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "username", secretEnv.ValueFrom.SecretKeyRef.Key)

	var outputSecretEnv *corev1.EnvVar
	for idx := range pod.Spec.Containers[0].Env {
		env := &pod.Spec.Containers[0].Env[idx]
		if env.Name == "PASSWORD_VAR2" {
			outputSecretEnv = env
			break
		}
	}
	require.NotNil(t, outputSecretEnv)
	require.NotNil(t, outputSecretEnv.ValueFrom)
	require.NotNil(t, outputSecretEnv.ValueFrom.SecretKeyRef)
	assert.Equal(t, "test-secret-3", outputSecretEnv.ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, "password", outputSecretEnv.ValueFrom.SecretKeyRef.Key)
}

func TestBuildCoordinatorExecutorPodUsesCanonicalTaskNameFlag(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	pod, err := buildCoordinatorExecutorPod(&TaskExecutionRequest{
		Run: &model.Run{
			UUID:           "run-id",
			K8SName:        "run-name",
			Namespace:      "kubeflow",
			DisplayName:    "run-display-name",
			ServiceAccount: "pipeline-runner",
		},
		Task: &model.Task{
			UUID:        "task-id",
			Name:        "canonical-task-name",
			DisplayName: "same display name",
		},
		Container:         &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"},
		ExecutorInput:     &pipelinespec.ExecutorInput{},
		ComponentSpec:     &pipelinespec.ComponentSpec{},
		PipelineName:      "pipeline-loop",
		TaskDisplayName:   "same display name",
		ArtifactLocalRoot: "/tmp/kfp-artifacts",
	})
	require.NoError(t, err)
	require.NotNil(t, pod)
	require.NotEmpty(t, pod.Spec.Containers)

	launcherArgs := append([]string{}, pod.Spec.Containers[0].Command...)
	launcherArgs = append(launcherArgs, pod.Spec.Containers[0].Args...)
	found := false
	for index := 0; index < len(launcherArgs)-1; index++ {
		if launcherArgs[index] == "--task_name" {
			assert.Equal(t, "canonical-task-name", launcherArgs[index+1])
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestBuildCoordinatorExecutorPodConfiguresLauncherTLS(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Set(common.CaBundleSecretName, "kfp-api-tls-cert")
	t.Cleanup(func() {
		viper.Set(common.CaBundleSecretName, "")
	})

	pod, err := buildCoordinatorExecutorPod(&TaskExecutionRequest{
		Run: &model.Run{
			UUID:           "run-id",
			K8SName:        "run-name",
			Namespace:      "kubeflow",
			DisplayName:    "run-display-name",
			ServiceAccount: "pipeline-runner",
		},
		Task: &model.Task{
			UUID:        "task-id",
			Name:        "tls-task",
			DisplayName: "tls-task",
		},
		Container:            &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"},
		ExecutorInput:        &pipelinespec.ExecutorInput{},
		ComponentSpec:        &pipelinespec.ComponentSpec{},
		PipelineName:         "pipeline-tls",
		TaskDisplayName:      "tls-task",
		ArtifactLocalRoot:    "/tmp/kfp-artifacts",
		MLPipelineTLSEnabled: true,
	})
	require.NoError(t, err)
	require.NotNil(t, pod)
	require.NotEmpty(t, pod.Spec.Containers)

	launcherArgs := append([]string{}, pod.Spec.Containers[0].Command...)
	launcherArgs = append(launcherArgs, pod.Spec.Containers[0].Args...)
	assert.Contains(t, launcherArgs, "--ml_pipeline_tls_enabled")
	assert.Contains(t, launcherArgs, "--ca_cert_path")
	assert.Contains(t, launcherArgs, common.CustomCaCertPath)

	var customCAVolume *corev1.Volume
	for index := range pod.Spec.Volumes {
		volume := &pod.Spec.Volumes[index]
		if volume.Name == "custom-ca" {
			customCAVolume = volume
			break
		}
	}
	require.NotNil(t, customCAVolume)
	require.NotNil(t, customCAVolume.Secret)
	assert.Equal(t, "kfp-api-tls-cert", customCAVolume.Secret.SecretName)

	var customCAMount *corev1.VolumeMount
	for index := range pod.Spec.Containers[0].VolumeMounts {
		volumeMount := &pod.Spec.Containers[0].VolumeMounts[index]
		if volumeMount.Name == "custom-ca" {
			customCAMount = volumeMount
			break
		}
	}
	require.NotNil(t, customCAMount)
	assert.Equal(t, common.CABundleDir, customCAMount.MountPath)
}

func TestBuildCoordinatorExecutorPodAppliesPodMetadata(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	pod, err := buildCoordinatorExecutorPod(&TaskExecutionRequest{
		Run: &model.Run{
			UUID:           "run-id",
			K8SName:        "run-name",
			Namespace:      "kubeflow",
			DisplayName:    "run-display-name",
			ServiceAccount: "pipeline-runner",
		},
		Task: &model.Task{
			UUID:        "task-id",
			Name:        "task-a",
			DisplayName: "task-a",
		},
		Container:     &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"},
		ComponentSpec: &pipelinespec.ComponentSpec{},
		KubernetesExecutorConfig: &kubernetesplatform.KubernetesExecutorConfig{
			PodMetadata: &kubernetesplatform.PodMetadata{
				Annotations: map[string]string{"task-annotation": "annotation"},
				Labels: map[string]string{
					"task-label-1": "label-1",
					"task-label-2": "label-2",
				},
			},
		},
		PipelineName:      "pipeline-with-pod-metadata",
		TaskDisplayName:   "task-a",
		ArtifactLocalRoot: "/tmp/kfp-artifacts",
	})
	require.NoError(t, err)
	require.NotNil(t, pod)
	assert.Equal(t, "annotation", pod.Annotations["task-annotation"])
	assert.Equal(t, commonutil.AnnotationValueIstioSidecarInjectDisabled, pod.Annotations[commonutil.AnnotationKeyIstioSidecarInject])
	assert.Equal(t, "label-1", pod.Labels["task-label-1"])
	assert.Equal(t, "label-2", pod.Labels["task-label-2"])
	assert.Equal(t, "run-id", pod.Labels["pipelines.kubeflow.org/run_id"])
	assert.Equal(t, "task-id", pod.Labels["pipelines.kubeflow.org/task_id"])
	assert.NotContains(t, pod.Spec.Containers[0].Command, "--pipeline_spec")
}

func TestBuildCoordinatorExecutorPodPreservesIstioInjectionAnnotation(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()

	pod, err := buildCoordinatorExecutorPod(&TaskExecutionRequest{
		Run: &model.Run{
			UUID:           "run-id",
			K8SName:        "run-name",
			Namespace:      "kubeflow",
			DisplayName:    "run-display-name",
			ServiceAccount: "pipeline-runner",
		},
		Task: &model.Task{
			UUID:        "task-id",
			Name:        "task-a",
			DisplayName: "task-a",
		},
		Container:     &pipelinespec.PipelineDeploymentConfig_PipelineContainerSpec{Image: "python:3.11"},
		ComponentSpec: &pipelinespec.ComponentSpec{},
		KubernetesExecutorConfig: &kubernetesplatform.KubernetesExecutorConfig{
			PodMetadata: &kubernetesplatform.PodMetadata{
				Annotations: map[string]string{
					commonutil.AnnotationKeyIstioSidecarInject: commonutil.AnnotationValueIstioSidecarInjectEnabled,
				},
			},
		},
		PipelineName:      "pipeline-with-pod-metadata",
		TaskDisplayName:   "task-a",
		ArtifactLocalRoot: "/tmp/kfp-artifacts",
	})
	require.NoError(t, err)
	require.NotNil(t, pod)
	assert.Equal(t, commonutil.AnnotationValueIstioSidecarInjectEnabled, pod.Annotations[commonutil.AnnotationKeyIstioSidecarInject])
}

func TestSupportsPipelineJobAcceptsCoordinatorFixtures(t *testing.T) {
	testCases := map[string]string{
		"flat-dag":      filepath.Join("..", "..", "driver", "test_data", "taskOutputParameter_test.yaml"),
		"artifact-flow": filepath.Join("..", "..", "driver", "test_data", "taskOutputArtifact_test.yaml"),
		"nested-dag": filepath.Join(
			"..", "..", "..", "..", "..",
			"test_data", "sdk_compiled_pipelines", "valid", "parallel_and_nested", "pipeline_in_pipeline.yaml",
		),
		"parameter-iterator": filepath.Join(
			"..", "..", "..", "..", "..",
			"test_data", "sdk_compiled_pipelines", "valid", "parallel_and_nested", "parallel_for_mount_pvc.yaml",
		),
		"condition-branch": filepath.Join(
			"..", "..", "..", "..", "..",
			"test_data", "sdk_compiled_pipelines", "valid", "parallel_and_nested", "pipeline_in_pipeline_complex.yaml",
		),
	}

	for name, relativePath := range testCases {
		t.Run(name, func(t *testing.T) {
			job := loadPipelineJobForSupportTest(t, relativePath)
			require.NoError(t, SupportsPipelineJob(job))
		})
	}
}

func loadPipelineJobForSupportTest(t *testing.T, relativePath string) *pipelinespec.PipelineJob {
	t.Helper()

	_, thisFile, _, _ := runtime.Caller(0)
	specPath := filepath.Join(filepath.Dir(thisFile), relativePath)
	templateBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)

	tmpl, err := template.New(templateBytes, template.TemplateOptions{})
	require.NoError(t, err)

	v2Spec, ok := tmpl.(*template.V2Spec)
	require.True(t, ok)

	run := &model.Run{
		UUID:        "support-test-run",
		DisplayName: "support-test-run",
		PipelineSpec: model.PipelineSpec{
			PipelineSpecManifest: model.LargeText(string(templateBytes)),
			RuntimeConfig: model.RuntimeConfig{
				Parameters: "{}",
			},
		},
		RunDetails: model.RunDetails{
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			State:            model.RuntimeStatePending,
		},
	}
	job, _, err := v2Spec.BuildPipelineJob(run, template.RunWorkflowOptions{
		RunID: run.UUID,
		RunAt: run.CreatedAtInSec,
	})
	require.NoError(t, err)
	return job
}
