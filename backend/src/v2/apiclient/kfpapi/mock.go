// Copyright 2025 The Kubeflow Authors
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

// Package kfpapi provides KFP API client implementation.
package kfpapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// MockAPI provides a mock implementation of API for testing
type MockAPI struct {
	runs             map[string]*apiv2beta1.Run
	tasks            map[string]*apiv2beta1.PipelineTaskDetail
	artifacts        map[string]*apiv2beta1.Artifact
	artifactTasks    map[string]*apiv2beta1.ArtifactTask
	pipelineVersions map[string]*apiv2beta1.PipelineVersion
}

// NewMockAPI creates a new mock API
func NewMockAPI() *MockAPI {
	return &MockAPI{
		runs:             make(map[string]*apiv2beta1.Run),
		tasks:            make(map[string]*apiv2beta1.PipelineTaskDetail),
		artifacts:        make(map[string]*apiv2beta1.Artifact),
		artifactTasks:    make(map[string]*apiv2beta1.ArtifactTask),
		pipelineVersions: make(map[string]*apiv2beta1.PipelineVersion),
	}
}

func (m *MockAPI) GetRun(_ context.Context, req *apiv2beta1.GetRunRequest) (*apiv2beta1.Run, error) {
	if run, exists := m.runs[req.RunId]; exists {
		// Create a copy of the run to populate with tasks
		populatedRun := &apiv2beta1.Run{
			RunId:          run.RunId,
			DisplayName:    run.DisplayName,
			PipelineSource: &apiv2beta1.Run_PipelineSpec{PipelineSpec: run.GetPipelineSpec()},
			RuntimeConfig:  run.RuntimeConfig,
			State:          run.State,
			Tasks:          []*apiv2beta1.PipelineTaskDetail{},
		}

		// Find all tasks for this run
		for _, task := range m.tasks {
			if task.RunId == req.RunId {
				// Create a copy of the task to populate with artifacts
				populatedTask := m.hydrateTask(task)
				populatedRun.Tasks = append(populatedRun.Tasks, populatedTask)
			}
		}
		return populatedRun, nil
	}
	return nil, fmt.Errorf("run not found: %s", req.RunId)
}

func (m *MockAPI) hydrateTask(task *apiv2beta1.PipelineTaskDetail) *apiv2beta1.PipelineTaskDetail {
	// Create a copy of the task to populate with artifacts
	populatedTask := proto.Clone(task).(*apiv2beta1.PipelineTaskDetail)
	populatedTask.Inputs = &apiv2beta1.PipelineTaskDetail_InputOutputs{}
	populatedTask.Outputs = &apiv2beta1.PipelineTaskDetail_InputOutputs{}

	// Copy existing parameters if they exist
	if task.Inputs != nil {
		populatedTask.Inputs.Parameters = task.Inputs.Parameters
	}
	if task.Outputs != nil {
		populatedTask.Outputs.Parameters = task.Outputs.Parameters
	}

	// Find artifacts associated with this task
	var inputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact
	var outputArtifacts []*apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact

	for _, artifactTask := range m.artifactTasks {
		if artifactTask.TaskId == task.TaskId {
			// Get the associated artifact
			if artifact, exists := m.artifacts[artifactTask.ArtifactId]; exists {
				ioArtifact := &apiv2beta1.PipelineTaskDetail_InputOutputs_IOArtifact{
					Artifacts: []*apiv2beta1.Artifact{artifact},
					Type:      artifactTask.Type,
				}

				ioArtifact.ArtifactKey = artifactTask.Key
				ioArtifact.Producer = artifactTask.Producer

				// Determine if this is an input or output artifact based on ArtifactTaskType
				switch artifactTask.Type {
				case apiv2beta1.IOType_COMPONENT_INPUT,
					apiv2beta1.IOType_ITERATOR_INPUT,
					apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
					apiv2beta1.IOType_COMPONENT_DEFAULT_INPUT,
					apiv2beta1.IOType_TASK_OUTPUT_INPUT,
					apiv2beta1.IOType_COLLECTED_INPUTS,
					apiv2beta1.IOType_ITERATOR_INPUT_RAW:
					inputArtifacts = append(inputArtifacts, ioArtifact)
				case apiv2beta1.IOType_OUTPUT,
					apiv2beta1.IOType_ITERATOR_OUTPUT,
					apiv2beta1.IOType_ONE_OF_OUTPUT,
					apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT:
					outputArtifacts = append(outputArtifacts, ioArtifact)
				}
			}
		}
	}

	// Set the artifacts on the task
	populatedTask.Inputs.Artifacts = inputArtifacts
	populatedTask.Outputs.Artifacts = outputArtifacts

	return populatedTask
}

func (m *MockAPI) CreateTask(_ context.Context, req *apiv2beta1.CreateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	task := req.Task
	if task.TaskId == "" {
		uuid, _ := uuid.NewRandom()
		task.TaskId = uuid.String()
	}
	m.tasks[task.TaskId] = task
	return task, nil
}

func (m *MockAPI) UpdateTask(_ context.Context, req *apiv2beta1.UpdateTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	if _, exists := m.tasks[req.TaskId]; !exists {
		return nil, fmt.Errorf("task not found: %s", req.TaskId)
	}
	task := req.Task
	task.TaskId = req.TaskId
	m.tasks[req.TaskId] = task
	task = m.hydrateTask(task)
	return task, nil
}

func (m *MockAPI) UpdateTasksBulk(_ context.Context, req *apiv2beta1.UpdateTasksBulkRequest) (*apiv2beta1.UpdateTasksBulkResponse, error) {
	response := &apiv2beta1.UpdateTasksBulkResponse{
		Tasks: make(map[string]*apiv2beta1.PipelineTaskDetail),
	}

	for taskID, task := range req.Tasks {
		if _, exists := m.tasks[taskID]; !exists {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		task.TaskId = taskID
		m.tasks[taskID] = task
		hydratedTask := m.hydrateTask(task)
		response.Tasks[taskID] = hydratedTask
	}

	return response, nil
}

func (m *MockAPI) GetTask(_ context.Context, req *apiv2beta1.GetTaskRequest) (*apiv2beta1.PipelineTaskDetail, error) {
	if _, exists := m.tasks[req.TaskId]; exists {
		task := m.hydrateTask(m.tasks[req.TaskId])
		return task, nil
	}

	return nil, fmt.Errorf("task not found: %s", req.TaskId)
}

func (m *MockAPI) ListTasks(_ context.Context, req *apiv2beta1.ListTasksRequest) (*apiv2beta1.ListTasksResponse, error) {
	var tasks []*apiv2beta1.PipelineTaskDetail

	var predicates []*apiv2beta1.Predicate
	if req.GetFilter() != "" {
		raw := strings.TrimSpace(req.GetFilter())
		filter := &apiv2beta1.Filter{}

		// First, try parsing as proto text format (matches filter.String()).
		if err := prototext.Unmarshal([]byte(raw), filter); err != nil {
			// Fallback to JSON. Support raw array of predicates by wrapping.
			if len(raw) > 0 && raw[0] == '[' {
				raw = `{"predicates":` + raw + `}`
			}
			if jerr := protojson.Unmarshal([]byte(raw), filter); jerr != nil {
				return nil, fmt.Errorf("failed to parse filter; textproto error: %v; json error: %v", err, jerr)
			}
		}
		predicates = filter.GetPredicates()
	}

	// Filter by run ID if specified
	if runID := req.GetRunId(); runID != "" {
		for _, task := range m.tasks {
			if task.RunId == runID {
				tasks = append(tasks, task)
			}
		}
	} else if parentID := req.GetParentId(); parentID != "" {
		// Filter by parent task ID
		for _, task := range m.tasks {
			if task.ParentTaskId != nil && *task.ParentTaskId == parentID {
				tasks = append(tasks, task)
			}
		}
	} else {
		// Return all tasks
		for _, task := range m.tasks {
			tasks = append(tasks, task)
		}
	}

	// Just handle cache case for now
	if len(predicates) == 2 {
		var statusPredicate *apiv2beta1.Predicate
		var fingerprintPredicate *apiv2beta1.Predicate

		switch {
		case predicates[0].Key == "status" && predicates[1].Key == "cache_fingerprint":
			statusPredicate = predicates[0]
			fingerprintPredicate = predicates[1]
		case predicates[1].Key == "status" && predicates[0].Key == "cache_fingerprint":
			statusPredicate = predicates[1]
			fingerprintPredicate = predicates[0]
		default:
			return nil, fmt.Errorf("only cache filter supported in mock library: %s", req.GetFilter())
		}

		var filtered []*apiv2beta1.PipelineTaskDetail
		status := statusPredicate.GetIntValue()
		fingerprint := fingerprintPredicate.GetStringValue()
		for _, t := range tasks {
			if int32(t.GetState().Number()) == status && t.GetCacheFingerprint() == fingerprint {
				filtered = append(filtered, t)
			}

		}
		tasks = filtered
	}

	var hydratedTasks []*apiv2beta1.PipelineTaskDetail
	for _, task := range tasks {
		hydratedTasks = append(hydratedTasks, m.hydrateTask(task))
	}

	return &apiv2beta1.ListTasksResponse{
		Tasks:     hydratedTasks,
		TotalSize: int32(len(tasks)),
	}, nil
}

func (m *MockAPI) CreateArtifact(_ context.Context, req *apiv2beta1.CreateArtifactRequest) (*apiv2beta1.Artifact, error) {
	artifact := req.Artifact
	if artifact.ArtifactId == "" {
		uuid, _ := uuid.NewRandom()
		artifact.ArtifactId = uuid.String()
	}
	m.artifacts[artifact.ArtifactId] = artifact

	task := m.tasks[req.TaskId]
	// Also create the artifact-task relationship
	// This mimics what the real API server does

	// Get the task name if the task exists, otherwise use empty string
	taskName := ""
	if task != nil {
		taskName = task.Name
	}

	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId: artifact.ArtifactId,
		TaskId:     req.TaskId,
		RunId:      req.RunId,
		Key:        req.ProducerKey,
		Type:       req.Type,
		Producer: &apiv2beta1.IOProducer{
			TaskName: taskName,
		},
	}
	if req.IterationIndex != nil {
		artifactTask.Producer.Iteration = req.IterationIndex
	}

	// Generate ID for artifact task
	atUUID, _ := uuid.NewRandom()
	artifactTask.Id = atUUID.String()
	m.artifactTasks[artifactTask.Id] = artifactTask

	return artifact, nil
}

func (m *MockAPI) CreateArtifactsBulk(_ context.Context, req *apiv2beta1.CreateArtifactsBulkRequest) (*apiv2beta1.CreateArtifactsBulkResponse, error) {
	response := &apiv2beta1.CreateArtifactsBulkResponse{
		Artifacts: make([]*apiv2beta1.Artifact, 0, len(req.Artifacts)),
	}

	for _, artifactReq := range req.Artifacts {
		artifact := artifactReq.Artifact
		if artifact.ArtifactId == "" {
			uuid, _ := uuid.NewRandom()
			artifact.ArtifactId = uuid.String()
		}
		m.artifacts[artifact.ArtifactId] = artifact

		// Get the task name if the task exists, otherwise use empty string
		task := m.tasks[artifactReq.TaskId]
		taskName := ""
		if task != nil {
			taskName = task.Name
		}

		// Also create the artifact-task relationship
		artifactTask := &apiv2beta1.ArtifactTask{
			ArtifactId: artifact.ArtifactId,
			TaskId:     artifactReq.TaskId,
			RunId:      artifactReq.RunId,
			Key:        artifactReq.ProducerKey,
			Type:       artifactReq.Type,
			Producer: &apiv2beta1.IOProducer{
				TaskName: taskName,
			},
		}
		if artifactReq.IterationIndex != nil {
			artifactTask.Producer.Iteration = artifactReq.IterationIndex
		}

		// Generate ID for artifact task
		atUUID, _ := uuid.NewRandom()
		artifactTask.Id = atUUID.String()
		m.artifactTasks[artifactTask.Id] = artifactTask

		response.Artifacts = append(response.Artifacts, artifact)
	}

	return response, nil
}

func (m *MockAPI) ListArtifactTasks(_ context.Context, _ *apiv2beta1.ListArtifactTasksRequest) (*apiv2beta1.ListArtifactTasksResponse, error) {
	var artifactTasks []*apiv2beta1.ArtifactTask
	for _, at := range m.artifactTasks {
		artifactTasks = append(artifactTasks, at)
	}
	return &apiv2beta1.ListArtifactTasksResponse{
		ArtifactTasks: artifactTasks,
		TotalSize:     int32(len(artifactTasks)),
	}, nil
}

func (m *MockAPI) ListArtifactsByURI(_ context.Context, uri string, namespace string) ([]*apiv2beta1.Artifact, error) {
	var artifacts []*apiv2beta1.Artifact
	for _, artifact := range m.artifacts {
		if artifact.GetUri() == uri && artifact.GetNamespace() == namespace {
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts, nil
}

func (m *MockAPI) CreateArtifactTask(_ context.Context, req *apiv2beta1.CreateArtifactTaskRequest) (*apiv2beta1.ArtifactTask, error) {
	artifactTask := req.ArtifactTask
	if artifactTask.Id == "" {
		uuid, _ := uuid.NewRandom()
		artifactTask.Id = uuid.String()
	}
	m.artifactTasks[artifactTask.Id] = artifactTask
	return artifactTask, nil
}

func (m *MockAPI) CreateArtifactTasks(_ context.Context, req *apiv2beta1.CreateArtifactTasksBulkRequest) (*apiv2beta1.CreateArtifactTasksBulkResponse, error) {
	var createdTasks []*apiv2beta1.ArtifactTask
	for _, at := range req.ArtifactTasks {
		if at.Id == "" {
			uuid, _ := uuid.NewRandom()
			at.Id = uuid.String()
		}
		m.artifactTasks[at.Id] = at
		createdTasks = append(createdTasks, at)
	}
	return &apiv2beta1.CreateArtifactTasksBulkResponse{
		ArtifactTasks: createdTasks,
	}, nil
}

func (m *MockAPI) GetPipelineVersion(_ context.Context, req *apiv2beta1.GetPipelineVersionRequest) (*apiv2beta1.PipelineVersion, error) {
	key := req.PipelineId + ":" + req.PipelineVersionId
	if pv, exists := m.pipelineVersions[key]; exists {
		return pv, nil
	}
	return nil, fmt.Errorf("pipeline version not found: %s", key)
}

func (m *MockAPI) FetchPipelineSpecFromRun(_ context.Context, run *apiv2beta1.Run) (*structpb.Struct, error) {
	var pipelineSpecStruct *structpb.Struct
	switch {
	case run.GetPipelineSpec() != nil:
		pipelineSpecStruct = run.GetPipelineSpec()
	case run.GetPipelineVersionReference() != nil:
		pvr := run.GetPipelineVersionReference()
		pipeline, err := m.GetPipelineVersion(context.Background(), &apiv2beta1.GetPipelineVersionRequest{
			PipelineId:        pvr.GetPipelineId(),
			PipelineVersionId: pvr.GetPipelineVersionId(),
		})
		if err != nil {
			return nil, err
		}
		pipelineSpecStruct = pipeline.GetPipelineSpec()
	default:
		return nil, fmt.Errorf("pipeline spec is not set")
	}
	if pipelineSpecStruct == nil {
		return nil, fmt.Errorf("pipeline spec is nil")
	}
	return pipelineSpecStruct, nil
}

// AddRun adds a run to the mock for testing
func (m *MockAPI) AddRun(run *apiv2beta1.Run) {
	if run.RunId == "" {
		uuid, _ := uuid.NewRandom()
		run.RunId = uuid.String()
	}
	m.runs[run.RunId] = run
}

// AddPipelineVersion adds a pipeline version to the mock for testing
func (m *MockAPI) AddPipelineVersion(pipelineID, versionID string, version *apiv2beta1.PipelineVersion) {
	key := pipelineID + ":" + versionID
	m.pipelineVersions[key] = version
}

func (m *MockAPI) UpdateStatuses(ctx context.Context, run *apiv2beta1.Run, pipelineSpec *structpb.Struct, currentTask *apiv2beta1.PipelineTaskDetail) error {
	return updateStatuses(ctx, run, m, pipelineSpec, currentTask)
}
