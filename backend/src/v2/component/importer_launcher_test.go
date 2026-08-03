// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestImportLauncher_CreateArtifactReusesMatchingIdentity(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	sharedURI := "gs://bucket/model"
	firstArtifact, err := mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{
			Name:      "model",
			Namespace: "test-namespace",
			Uri:       &sharedURI,
			Type:      apiv2beta1.Artifact_Dataset,
			Metadata: map[string]*structpb.Value{
				"source": structpb.NewStringValue("shared"),
			},
		},
	})
	require.NoError(t, err)

	reusedArtifact, err := mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		ReuseIfExists: true,
		Artifact: &apiv2beta1.Artifact{
			Name:      "model",
			Namespace: "test-namespace",
			Uri:       &sharedURI,
			Type:      apiv2beta1.Artifact_Dataset,
			Metadata: map[string]*structpb.Value{
				"source": structpb.NewStringValue("shared"),
			},
		},
		TaskId:      "task-2",
		RunId:       "run-2",
		ProducerKey: "artifact",
	})
	require.NoError(t, err)
	require.Equal(t, firstArtifact.GetArtifactId(), reusedArtifact.GetArtifactId())
}

func TestImportLauncher_ReimportDisablesReuseIfExists(t *testing.T) {
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "importer"},
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-importer"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
					},
				},
			},
		},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{"importer": taskSpec},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{"comp-importer": componentSpec},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "importer")
	require.NoError(t, err)
	runID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{TaskId: uuid.NewString(), RunId: runID, Name: "root", State: apiv2beta1.PipelineTask_RUNNING, Type: apiv2beta1.PipelineTask_DAG, ScopePath: "root"}
	mockAPI := &importerTestAPI{
		MockAPI: kfpapi.NewMockAPI(),
		runs:    map[string]*apiv2beta1.Run{runID: {RunId: runID}},
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask})
	require.NoError(t, err)
	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
			Reimport: true,
			ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("gs://bucket/model"),
				},
			},
			TypeSchema: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			},
		},
		PipelineSpec: pipelineSpecStruct,
		TaskSpec:     taskSpec,
		ScopePath:    scopePath,
		Run:          &apiv2beta1.Run{RunId: runID, Tasks: []*apiv2beta1.PipelineTask{rootTask}},
		ParentTask:   rootTask,
	}, clientManager)
	require.NoError(t, err)

	require.NoError(t, importerLauncher.Execute(context.Background()))
	require.Len(t, mockAPI.createArtifactRequests, 1)
	require.False(t, mockAPI.createArtifactRequests[0].GetReuseIfExists())
}

func TestImportLauncher_RefreshesRunBeforeUpdatingStatuses(t *testing.T) {
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo: &pipelinespec.PipelineTaskInfo{Name: "importer"},
		ComponentRef: &pipelinespec.ComponentRef{
			Name: "comp-importer",
		},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
						SchemaVersion: "0.0.1",
					},
				},
			},
		},
		Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-importer"},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"importer": taskSpec,
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"comp-importer": componentSpec,
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	if err != nil {
		t.Fatalf("failed to convert pipeline spec to struct: %v", err)
	}
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "importer")
	if err != nil {
		t.Fatalf("failed to create scope path: %v", err)
	}
	importerSpec := &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
		ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
			Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
				Constant: structpb.NewStringValue("gs://bucket/model"),
			},
		},
		TypeSchema: &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			SchemaVersion: "0.0.1",
		},
	}

	runID := uuid.NewString()
	rootTaskID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{
		TaskId:    rootTaskID,
		RunId:     runID,
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
	}
	staleRun := &apiv2beta1.Run{
		RunId: runID,
		Tasks: []*apiv2beta1.PipelineTask{rootTask},
	}

	mockAPI := &importerTestAPI{
		MockAPI: kfpapi.NewMockAPI(),
		runs: map[string]*apiv2beta1.Run{
			runID: {RunId: runID},
		},
	}
	if _, err := mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask}); err != nil {
		t.Fatalf("failed to seed root task: %v", err)
	}

	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec:  importerSpec,
		PipelineSpec:  pipelineSpecStruct,
		TaskSpec:      taskSpec,
		ScopePath:     scopePath,
		Run:           staleRun,
		ParentTask:    rootTask,
	}, clientManager)
	if err != nil {
		t.Fatalf("failed to create importer launcher: %v", err)
	}

	if err := importerLauncher.Execute(context.Background()); err != nil {
		t.Fatalf("failed to execute importer launcher: %v", err)
	}

	updatedRootTask, err := mockAPI.GetTask(context.Background(), &apiv2beta1.GetTaskRequest{TaskId: rootTaskID})
	if err != nil {
		t.Fatalf("failed to get root task: %v", err)
	}
	if updatedRootTask.GetState() != apiv2beta1.PipelineTask_SUCCEEDED {
		t.Fatalf("expected root task to be succeeded, got %v", updatedRootTask.GetState())
	}
}

func TestImportLauncher_RetryReusesExistingTask(t *testing.T) {
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "importer"},
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-importer"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
						SchemaVersion: "0.0.1",
					},
				},
			},
		},
		Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-importer"},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						"importer": taskSpec,
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"comp-importer": componentSpec,
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "importer")
	require.NoError(t, err)
	importerSpec := &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
		ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
			Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
				Constant: structpb.NewStringValue("gs://bucket/model"),
			},
		},
		TypeSchema: &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			SchemaVersion: "0.0.1",
		},
	}

	runID := uuid.NewString()
	rootTaskID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{
		TaskId:    rootTaskID,
		RunId:     runID,
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
	}
	staleRun := &apiv2beta1.Run{
		RunId: runID,
		Tasks: []*apiv2beta1.PipelineTask{rootTask},
	}

	mockAPI := &importerTestAPI{
		MockAPI: kfpapi.NewMockAPI(),
		runs: map[string]*apiv2beta1.Run{
			runID: {RunId: runID},
		},
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask})
	require.NoError(t, err)

	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec:  importerSpec,
		PipelineSpec:  pipelineSpecStruct,
		TaskSpec:      taskSpec,
		ScopePath:     scopePath,
		Run:           staleRun,
		ParentTask:    rootTask,
	}, clientManager)
	require.NoError(t, err)

	require.NoError(t, importerLauncher.Execute(context.Background()))

	firstTasks, err := mockAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{RunId: runID})
	require.NoError(t, err)
	var firstImporter *apiv2beta1.PipelineTask
	for _, task := range firstTasks.GetTasks() {
		if task.GetType() == apiv2beta1.PipelineTask_IMPORTER {
			firstImporter = task
			break
		}
	}
	require.NotNil(t, firstImporter)
	require.Len(t, firstImporter.GetPods(), 1)
	require.Equal(t, "test-pod", firstImporter.GetPods()[0].GetName())

	// Simulate retry reset: clear attempt-local pods and reopen the task.
	firstImporter.State = apiv2beta1.PipelineTask_RUNNING
	firstImporter.Pods = []*apiv2beta1.PipelineTask_TaskPod{}
	firstImporter.EndTime = nil
	_, err = mockAPI.UpdateTask(context.Background(), &apiv2beta1.UpdateTaskRequest{
		TaskId: firstImporter.GetTaskId(),
		Task:   firstImporter,
		RunId:  runID,
	})
	require.NoError(t, err)

	importerLauncher.opts.PodName = "retry-pod"
	importerLauncher.opts.PodUID = "retry-pod-uid"
	require.NoError(t, importerLauncher.Execute(context.Background()))

	refreshedRun, err := mockAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: runID})
	require.NoError(t, err)
	var importerTasks []*apiv2beta1.PipelineTask
	for _, task := range refreshedRun.GetTasks() {
		if task.GetType() == apiv2beta1.PipelineTask_IMPORTER {
			importerTasks = append(importerTasks, task)
		}
	}
	require.Len(t, importerTasks, 1)
	require.Len(t, importerTasks[0].GetPods(), 1)
	require.Equal(t, "retry-pod", importerTasks[0].GetPods()[0].GetName())
	require.Equal(t, "retry-pod-uid", importerTasks[0].GetPods()[0].GetUid())
}

func TestImportLauncher_UsesCanonicalTaskNameNotDisplayName(t *testing.T) {
	const (
		canonicalTaskName = "importer-task"
		displayTaskName   = "Import Dataset Display"
	)
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: displayTaskName},
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-importer"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
						SchemaVersion: "0.0.1",
					},
				},
			},
		},
		Implementation: &pipelinespec.ComponentSpec_ExecutorLabel{ExecutorLabel: "exec-importer"},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{
						canonicalTaskName: taskSpec,
					},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{
			"comp-importer": componentSpec,
		},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", canonicalTaskName)
	require.NoError(t, err)
	importerSpec := &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
		ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
			Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
				Constant: structpb.NewStringValue("gs://bucket/model"),
			},
		},
		TypeSchema: &pipelinespec.ArtifactTypeSchema{
			Kind:          &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			SchemaVersion: "0.0.1",
		},
	}

	runID := uuid.NewString()
	rootTaskID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{
		TaskId:    rootTaskID,
		RunId:     runID,
		Name:      "root",
		State:     apiv2beta1.PipelineTask_RUNNING,
		Type:      apiv2beta1.PipelineTask_DAG,
		ScopePath: "root",
	}
	mockAPI := &importerTestAPI{
		MockAPI: kfpapi.NewMockAPI(),
		runs: map[string]*apiv2beta1.Run{
			runID: {RunId: runID},
		},
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask})
	require.NoError(t, err)

	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec:  importerSpec,
		PipelineSpec:  pipelineSpecStruct,
		TaskSpec:      taskSpec,
		ScopePath:     scopePath,
		Run:           &apiv2beta1.Run{RunId: runID, Tasks: []*apiv2beta1.PipelineTask{rootTask}},
		ParentTask:    rootTask,
	}, clientManager)
	require.NoError(t, err)

	require.NoError(t, importerLauncher.Execute(context.Background()))

	refreshedRun, err := mockAPI.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: runID})
	require.NoError(t, err)
	var importerTask *apiv2beta1.PipelineTask
	for _, task := range refreshedRun.GetTasks() {
		if task.GetType() == apiv2beta1.PipelineTask_IMPORTER {
			importerTask = task
			break
		}
	}
	require.NotNil(t, importerTask)
	require.Equal(t, canonicalTaskName, importerTask.GetName(), "Name must be the canonical DAG task key")
	require.Equal(t, displayTaskName, importerTask.GetDisplayName(), "DisplayName must come from TaskInfo")
	require.Len(t, importerTask.GetOutputs().GetArtifacts(), 1)
	require.Equal(
		t,
		canonicalTaskName,
		importerTask.GetOutputs().GetArtifacts()[0].GetProducer().GetTaskName(),
		"IOProducer.TaskName must be the canonical DAG task key, not DisplayName",
	)
}

type importerTestAPI struct {
	*kfpapi.MockAPI
	runs                   map[string]*apiv2beta1.Run
	createArtifactRequests []*apiv2beta1.CreateArtifactRequest
	updateStatusesErr      error
	getRunErr              error
	failNextUpdateTask     error
	updateTaskCalls        int
	updateStatusesCalls    int
}

func (m *importerTestAPI) GetRun(ctx context.Context, req *apiv2beta1.GetRunRequest) (*apiv2beta1.Run, error) {
	if m.getRunErr != nil {
		return nil, m.getRunErr
	}
	run, ok := m.runs[req.GetRunId()]
	if !ok {
		return nil, fmt.Errorf("run not found: %s", req.GetRunId())
	}
	populatedRun := proto.Clone(run).(*apiv2beta1.Run)
	tasks, err := m.ListTasks(ctx, &apiv2beta1.ListTasksRequest{RunId: req.GetRunId()})
	if err != nil {
		return nil, err
	}
	populatedRun.Tasks = tasks.GetTasks()
	return populatedRun, nil
}

func (m *importerTestAPI) UpdateTask(ctx context.Context, req *apiv2beta1.UpdateTaskRequest) (*apiv2beta1.PipelineTask, error) {
	m.updateTaskCalls++
	if m.failNextUpdateTask != nil {
		err := m.failNextUpdateTask
		m.failNextUpdateTask = nil
		return nil, err
	}
	return m.MockAPI.UpdateTask(ctx, req)
}

func (m *importerTestAPI) UpdateStatuses(ctx context.Context, run *apiv2beta1.Run, pipelineSpec *structpb.Struct, currentTask *apiv2beta1.PipelineTask) error {
	m.updateStatusesCalls++
	if m.updateStatusesErr != nil {
		return m.updateStatusesErr
	}
	if currentTask != nil && currentTask.GetParentTaskId() != "" {
		parentTask, err := m.GetTask(ctx, &apiv2beta1.GetTaskRequest{TaskId: currentTask.GetParentTaskId(), RunId: currentTask.GetRunId()})
		if err != nil {
			return err
		}
		parentTask.State = currentTask.GetState()
		_, err = m.MockAPI.UpdateTask(ctx, &apiv2beta1.UpdateTaskRequest{TaskId: parentTask.GetTaskId(), Task: parentTask, RunId: parentTask.GetRunId()})
		return err
	}
	return nil
}

func (m *importerTestAPI) CreateArtifact(ctx context.Context, req *apiv2beta1.CreateArtifactRequest) (*apiv2beta1.Artifact, error) {
	m.createArtifactRequests = append(m.createArtifactRequests, proto.Clone(req).(*apiv2beta1.CreateArtifactRequest))
	return m.MockAPI.CreateArtifact(ctx, req)
}

func TestImportLauncher_PassesIterationIndexToCreateArtifact(t *testing.T) {
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "importer"},
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-importer"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
					},
				},
			},
		},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{"importer": taskSpec},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{"comp-importer": componentSpec},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "importer")
	require.NoError(t, err)
	runID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{TaskId: uuid.NewString(), RunId: runID, Name: "root", State: apiv2beta1.PipelineTask_RUNNING, Type: apiv2beta1.PipelineTask_DAG, ScopePath: "root"}
	iterationIndex := int64(2)
	mockAPI := &importerTestAPI{
		MockAPI: kfpapi.NewMockAPI(),
		runs:    map[string]*apiv2beta1.Run{runID: {RunId: runID}},
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask})
	require.NoError(t, err)
	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
			ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("gs://bucket/model"),
				},
			},
			TypeSchema: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			},
		},
		PipelineSpec:   pipelineSpecStruct,
		TaskSpec:       taskSpec,
		ScopePath:      scopePath,
		Run:            &apiv2beta1.Run{RunId: runID, Tasks: []*apiv2beta1.PipelineTask{rootTask}},
		ParentTask:     rootTask,
		IterationIndex: &iterationIndex,
	}, clientManager)
	require.NoError(t, err)

	require.NoError(t, importerLauncher.Execute(context.Background()))
	require.Len(t, mockAPI.createArtifactRequests, 1)
	require.NotNil(t, mockAPI.createArtifactRequests[0].IterationIndex)
	require.EqualValues(t, iterationIndex, *mockAPI.createArtifactRequests[0].IterationIndex)
	require.True(t, mockAPI.createArtifactRequests[0].GetReuseIfExists())
}

func TestImportLauncher_PropagatesStatusRefreshFailures(t *testing.T) {
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "importer"},
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-importer"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
					},
				},
			},
		},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{"importer": taskSpec},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{"comp-importer": componentSpec},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "importer")
	require.NoError(t, err)
	runID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{TaskId: uuid.NewString(), RunId: runID, Name: "root", State: apiv2beta1.PipelineTask_RUNNING, Type: apiv2beta1.PipelineTask_DAG, ScopePath: "root"}
	mockAPI := &importerTestAPI{
		MockAPI:   kfpapi.NewMockAPI(),
		runs:      map[string]*apiv2beta1.Run{runID: {RunId: runID}},
		getRunErr: fmt.Errorf("refresh failed"),
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask})
	require.NoError(t, err)
	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
			ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("gs://bucket/model"),
				},
			},
			TypeSchema: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			},
		},
		PipelineSpec: pipelineSpecStruct,
		TaskSpec:     taskSpec,
		ScopePath:    scopePath,
		Run:          &apiv2beta1.Run{RunId: runID, Tasks: []*apiv2beta1.PipelineTask{rootTask}},
		ParentTask:   rootTask,
	}, clientManager)
	require.NoError(t, err)

	err = importerLauncher.Execute(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to refresh run")

	tasks, listErr := mockAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{RunId: runID})
	require.NoError(t, listErr)
	var importerTask *apiv2beta1.PipelineTask
	var parentTask *apiv2beta1.PipelineTask
	for _, task := range tasks.GetTasks() {
		switch {
		case task.GetType() == apiv2beta1.PipelineTask_IMPORTER:
			importerTask = task
		case task.GetTaskId() == rootTask.GetTaskId():
			parentTask = task
		}
	}
	require.NotNil(t, importerTask, "importer task should be persisted")
	require.Equal(t, apiv2beta1.PipelineTask_FAILED, importerTask.GetState(),
		"importer must not remain SUCCEEDED after post-success finalization failure")
	require.NotNil(t, parentTask)
	// GetRun fails permanently, so ancestor propagation cannot complete; leaf
	// reconciliation still must leave the importer FAILED.
	require.Equal(t, apiv2beta1.PipelineTask_RUNNING, parentTask.GetState())
}

func TestImportLauncher_PropagatesOrdinaryExecutionFailures(t *testing.T) {
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "importer"},
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-importer"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
					},
				},
			},
		},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{"importer": taskSpec},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{"comp-importer": componentSpec},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "importer")
	require.NoError(t, err)
	runID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{TaskId: uuid.NewString(), RunId: runID, Name: "root", State: apiv2beta1.PipelineTask_RUNNING, Type: apiv2beta1.PipelineTask_DAG, ScopePath: "root"}
	mockAPI := &importerTestAPI{
		MockAPI: kfpapi.NewMockAPI(),
		runs:    map[string]*apiv2beta1.Run{runID: {RunId: runID}},
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask})
	require.NoError(t, err)
	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
			ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue(""),
				},
			},
			TypeSchema: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			},
		},
		PipelineSpec: pipelineSpecStruct,
		TaskSpec:     taskSpec,
		ScopePath:    scopePath,
		Run:          &apiv2beta1.Run{RunId: runID, Tasks: []*apiv2beta1.PipelineTask{rootTask}},
		ParentTask:   rootTask,
	}, clientManager)
	require.NoError(t, err)

	err = importerLauncher.Execute(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty Artifact URI")
	require.GreaterOrEqual(t, mockAPI.updateStatusesCalls, 1)

	tasks, listErr := mockAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{RunId: runID})
	require.NoError(t, listErr)
	var importerTask *apiv2beta1.PipelineTask
	var parentTask *apiv2beta1.PipelineTask
	for _, task := range tasks.GetTasks() {
		switch {
		case task.GetType() == apiv2beta1.PipelineTask_IMPORTER:
			importerTask = task
		case task.GetTaskId() == rootTask.GetTaskId():
			parentTask = task
		}
	}
	require.NotNil(t, importerTask)
	require.Equal(t, apiv2beta1.PipelineTask_FAILED, importerTask.GetState())
	require.NotNil(t, parentTask)
	require.Equal(t, apiv2beta1.PipelineTask_FAILED, parentTask.GetState(),
		"ordinary importer execution failures must propagate to ancestors")
}

func TestImportLauncher_PodUpdateFailureFinalizesAndPropagates(t *testing.T) {
	taskSpec := &pipelinespec.PipelineTaskSpec{
		TaskInfo:     &pipelinespec.PipelineTaskInfo{Name: "importer"},
		ComponentRef: &pipelinespec.ComponentRef{Name: "comp-importer"},
	}
	componentSpec := &pipelinespec.ComponentSpec{
		OutputDefinitions: &pipelinespec.ComponentOutputsSpec{
			Artifacts: map[string]*pipelinespec.ComponentOutputsSpec_ArtifactSpec{
				"artifact": {
					ArtifactType: &pipelinespec.ArtifactTypeSchema{
						Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
					},
				},
			},
		},
	}
	pipelineSpec := &pipelinespec.PipelineSpec{
		PipelineInfo: &pipelinespec.PipelineInfo{Name: "pipeline-with-importer"},
		Root: &pipelinespec.ComponentSpec{
			Implementation: &pipelinespec.ComponentSpec_Dag{
				Dag: &pipelinespec.DagSpec{
					Tasks: map[string]*pipelinespec.PipelineTaskSpec{"importer": taskSpec},
				},
			},
		},
		Components: map[string]*pipelinespec.ComponentSpec{"comp-importer": componentSpec},
	}
	pipelineSpecStruct, err := pipelineSpecToStruct(t, pipelineSpec)
	require.NoError(t, err)
	scopePath, err := util.ScopePathFromStringPathWithNewTask(pipelineSpecStruct, "root", "importer")
	require.NoError(t, err)
	runID := uuid.NewString()
	rootTask := &apiv2beta1.PipelineTask{TaskId: uuid.NewString(), RunId: runID, Name: "root", State: apiv2beta1.PipelineTask_RUNNING, Type: apiv2beta1.PipelineTask_DAG, ScopePath: "root"}
	mockAPI := &importerTestAPI{
		MockAPI:            kfpapi.NewMockAPI(),
		runs:               map[string]*apiv2beta1.Run{runID: {RunId: runID}},
		failNextUpdateTask: fmt.Errorf("pod reattach failed"),
	}
	_, err = mockAPI.CreateTask(context.Background(), &apiv2beta1.CreateTaskRequest{RunId: runID, Task: rootTask})
	require.NoError(t, err)
	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	importerLauncher, err := NewImporterLauncher(&LauncherV2Options{
		Namespace:     "test-namespace",
		PodName:       "test-pod",
		PodUID:        "test-pod-uid",
		PipelineName:  pipelineSpec.GetPipelineInfo().GetName(),
		ComponentSpec: componentSpec,
		ImporterSpec: &pipelinespec.PipelineDeploymentConfig_ImporterSpec{
			ArtifactUri: &pipelinespec.ValueOrRuntimeParameter{
				Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
					Constant: structpb.NewStringValue("gs://bucket/model"),
				},
			},
			TypeSchema: &pipelinespec.ArtifactTypeSchema{
				Kind: &pipelinespec.ArtifactTypeSchema_SchemaTitle{SchemaTitle: "system.Dataset"},
			},
		},
		PipelineSpec: pipelineSpecStruct,
		TaskSpec:     taskSpec,
		ScopePath:    scopePath,
		Run:          &apiv2beta1.Run{RunId: runID, Tasks: []*apiv2beta1.PipelineTask{rootTask}},
		ParentTask:   rootTask,
	}, clientManager)
	require.NoError(t, err)

	err = importerLauncher.Execute(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to update importer attempt-local pods after create")
	require.GreaterOrEqual(t, mockAPI.updateStatusesCalls, 1)

	tasks, listErr := mockAPI.ListTasks(context.Background(), &apiv2beta1.ListTasksRequest{RunId: runID})
	require.NoError(t, listErr)
	var importerTask *apiv2beta1.PipelineTask
	var parentTask *apiv2beta1.PipelineTask
	for _, task := range tasks.GetTasks() {
		switch {
		case task.GetType() == apiv2beta1.PipelineTask_IMPORTER:
			importerTask = task
		case task.GetTaskId() == rootTask.GetTaskId():
			parentTask = task
		}
	}
	require.NotNil(t, importerTask)
	require.Equal(t, apiv2beta1.PipelineTask_FAILED, importerTask.GetState(),
		"pod UpdateTask failure must not leave importer RUNNING")
	require.NotNil(t, parentTask)
	require.Equal(t, apiv2beta1.PipelineTask_FAILED, parentTask.GetState())
}

func pipelineSpecToStruct(t *testing.T, pipelineSpec *pipelinespec.PipelineSpec) (*structpb.Struct, error) {
	t.Helper()
	specJSON, err := protojson.Marshal(pipelineSpec)
	if err != nil {
		return nil, err
	}
	specStruct := &structpb.Struct{}
	if err := protojson.Unmarshal(specJSON, specStruct); err != nil {
		return nil, err
	}
	return specStruct, nil
}
