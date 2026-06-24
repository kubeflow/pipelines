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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestImportLauncher_FindMatchedArtifactUsesFullArtifactEquality(t *testing.T) {
	mockAPI := kfpapi.NewMockAPI()
	clientManager := clientmanager.NewFakeClientManager(k8sfake.NewClientset(), mockAPI)
	launcher := &ImportLauncher{
		opts: LauncherV2Options{
			Namespace: "test-namespace",
		},
		clientManager: clientManager,
	}

	sharedURI := "gs://bucket/model"
	firstArtifact, err := mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{
			Name:      "first",
			Namespace: "test-namespace",
			Uri:       &sharedURI,
			Type:      apiv2beta1.Artifact_Dataset,
			Metadata: map[string]*structpb.Value{
				"source": structpb.NewStringValue("first"),
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create first artifact: %v", err)
	}
	secondArtifact, err := mockAPI.CreateArtifact(context.Background(), &apiv2beta1.CreateArtifactRequest{
		Artifact: &apiv2beta1.Artifact{
			Name:      "second",
			Namespace: "test-namespace",
			Uri:       &sharedURI,
			Type:      apiv2beta1.Artifact_Dataset,
			Metadata: map[string]*structpb.Value{
				"source": structpb.NewStringValue("second"),
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create second artifact: %v", err)
	}

	matchedArtifact, err := launcher.findMatchedArtifact(context.Background(), proto.Clone(secondArtifact).(*apiv2beta1.Artifact))
	if err != nil {
		t.Fatalf("findMatchedArtifact returned error: %v", err)
	}
	if matchedArtifact == nil {
		t.Fatal("expected matched artifact")
	}
	if matchedArtifact.GetArtifactId() != secondArtifact.GetArtifactId() {
		t.Fatalf(
			"expected artifact %q, got %q (first artifact was %q)",
			secondArtifact.GetArtifactId(),
			matchedArtifact.GetArtifactId(),
			firstArtifact.GetArtifactId(),
		)
	}
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

type importerTestAPI struct {
	*kfpapi.MockAPI
	runs map[string]*apiv2beta1.Run
}

func (m *importerTestAPI) GetRun(ctx context.Context, req *apiv2beta1.GetRunRequest) (*apiv2beta1.Run, error) {
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
