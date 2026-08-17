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
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// MockAPI provides a mock implementation of API for testing
type MockAPI struct {
	runs             map[string]*apiv2beta1.Run
	tasks            map[string]*apiv2beta1.PipelineTask
	artifacts        map[string]*apiv2beta1.Artifact
	artifactTasks    map[string]*apiv2beta1.ArtifactTask
	pipelineVersions map[string]*apiv2beta1.PipelineVersion
}

// NewMockAPI creates a new mock API
func NewMockAPI() *MockAPI {
	return &MockAPI{
		runs:             make(map[string]*apiv2beta1.Run),
		tasks:            make(map[string]*apiv2beta1.PipelineTask),
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
			Tasks:          []*apiv2beta1.PipelineTask{},
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

func (m *MockAPI) ListRuns(_ context.Context, req *apiv2beta1.ListRunsRequest) (*apiv2beta1.ListRunsResponse, error) {
	runs := make([]*apiv2beta1.Run, 0, len(m.runs))
	for _, run := range m.runs {
		populatedRun, err := m.GetRun(context.Background(), &apiv2beta1.GetRunRequest{RunId: run.GetRunId()})
		if err != nil {
			return nil, err
		}
		runs = append(runs, populatedRun)
	}
	return &apiv2beta1.ListRunsResponse{
		Runs:      runs,
		TotalSize: int32(len(runs)),
	}, nil
}

func (m *MockAPI) hydrateTask(task *apiv2beta1.PipelineTask) *apiv2beta1.PipelineTask {
	// Create a copy of the task to populate with artifacts
	populatedTask := proto.Clone(task).(*apiv2beta1.PipelineTask)
	populatedTask.Inputs = &apiv2beta1.PipelineTask_InputOutputs{}
	populatedTask.Outputs = &apiv2beta1.PipelineTask_InputOutputs{}

	// Copy existing parameters if they exist
	if task.Inputs != nil {
		populatedTask.Inputs.Parameters = task.Inputs.Parameters
	}
	if task.Outputs != nil {
		populatedTask.Outputs.Parameters = task.Outputs.Parameters
	}

	type hydratedArtifact struct {
		artifact *apiv2beta1.Artifact
		task     *apiv2beta1.ArtifactTask
	}
	type groupKey struct {
		artifactKey  string
		ioType       apiv2beta1.IOType
		producerTask string
		hasIteration bool
		iterationVal int64
	}
	makeKey := func(artifactTask *apiv2beta1.ArtifactTask) groupKey {
		key := groupKey{
			artifactKey: artifactTask.GetKey(),
			ioType:      artifactTask.GetType(),
		}
		if producer := artifactTask.GetProducer(); producer != nil {
			key.producerTask = producer.GetTaskName()
			// Only split by iteration for ITERATOR_OUTPUT; ordinary outputs
			// consolidate all same-key artifacts into a single IOArtifact.
			if artifactTask.GetType() == apiv2beta1.IOType_ITERATOR_OUTPUT && producer.Iteration != nil {
				key.hasIteration = true
				key.iterationVal = producer.GetIteration()
			}
		}
		return key
	}

	inputGrouped := make(map[groupKey][]hydratedArtifact)
	outputGrouped := make(map[groupKey][]hydratedArtifact)
	for _, artifactTask := range m.artifactTasks {
		if artifactTask.TaskId != task.TaskId {
			continue
		}
		artifact, exists := m.artifacts[artifactTask.ArtifactId]
		if !exists {
			continue
		}
		entry := hydratedArtifact{artifact: artifact, task: artifactTask}
		key := makeKey(artifactTask)
		switch artifactTask.Type {
		case apiv2beta1.IOType_COMPONENT_INPUT,
			apiv2beta1.IOType_ITERATOR_INPUT,
			apiv2beta1.IOType_RUNTIME_VALUE_INPUT,
			apiv2beta1.IOType_COMPONENT_DEFAULT_INPUT,
			apiv2beta1.IOType_TASK_OUTPUT_INPUT,
			apiv2beta1.IOType_COLLECTED_INPUTS,
			apiv2beta1.IOType_ITERATOR_INPUT_RAW:
			inputGrouped[key] = append(inputGrouped[key], entry)
		case apiv2beta1.IOType_OUTPUT,
			apiv2beta1.IOType_ITERATOR_OUTPUT,
			apiv2beta1.IOType_ONE_OF_OUTPUT,
			apiv2beta1.IOType_TASK_FINAL_STATUS_OUTPUT:
			outputGrouped[key] = append(outputGrouped[key], entry)
		}
	}

	groupToIOArtifacts := func(grouped map[groupKey][]hydratedArtifact) []*apiv2beta1.PipelineTask_InputOutputs_IOArtifact {
		if len(grouped) == 0 {
			return nil
		}
		keys := make([]groupKey, 0, len(grouped))
		for key := range grouped {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			left := keys[i]
			right := keys[j]
			if left.artifactKey != right.artifactKey {
				return left.artifactKey < right.artifactKey
			}
			if left.ioType != right.ioType {
				return left.ioType < right.ioType
			}
			if left.producerTask != right.producerTask {
				return left.producerTask < right.producerTask
			}
			if left.hasIteration != right.hasIteration {
				return !left.hasIteration && right.hasIteration
			}
			return left.iterationVal < right.iterationVal
		})
		out := make([]*apiv2beta1.PipelineTask_InputOutputs_IOArtifact, 0, len(keys))
		for _, key := range keys {
			entries := grouped[key]
			artifacts := make([]*apiv2beta1.Artifact, 0, len(entries))
			for _, entry := range entries {
				artifacts = append(artifacts, entry.artifact)
			}
			first := entries[0].task
			ioArtifact := &apiv2beta1.PipelineTask_InputOutputs_IOArtifact{
				Artifacts:   artifacts,
				ArtifactKey: first.GetKey(),
				Type:        first.GetType(),
				Producer:    first.GetProducer(),
			}
			out = append(out, ioArtifact)
		}
		return out
	}

	populatedTask.Inputs.Artifacts = groupToIOArtifacts(inputGrouped)
	populatedTask.Outputs.Artifacts = groupToIOArtifacts(outputGrouped)

	return populatedTask
}

func taskIterationIndex(task *apiv2beta1.PipelineTask) *int64 {
	if task == nil || task.GetTypeAttributes() == nil || task.GetTypeAttributes().IterationIndex == nil {
		return nil
	}
	return task.GetTypeAttributes().IterationIndex
}

func sameLogicalTaskIdentity(existingTask, candidateTask *apiv2beta1.PipelineTask, runID string) bool {
	if existingTask == nil || candidateTask == nil {
		return false
	}
	if existingTask.GetRunId() != runID ||
		normalizedParentTaskID(existingTask) != normalizedParentTaskID(candidateTask) ||
		existingTask.GetScopePath() != candidateTask.GetScopePath() ||
		existingTask.GetName() != candidateTask.GetName() ||
		existingTask.GetType() != candidateTask.GetType() {
		return false
	}
	existingIterationIndex := taskIterationIndex(existingTask)
	candidateIterationIndex := taskIterationIndex(candidateTask)
	if existingIterationIndex == nil && candidateIterationIndex == nil {
		return true
	}
	if existingIterationIndex == nil || candidateIterationIndex == nil {
		return false
	}
	return *existingIterationIndex == *candidateIterationIndex
}

func normalizedParentTaskID(task *apiv2beta1.PipelineTask) string {
	if task == nil {
		return ""
	}
	return task.GetParentTaskId()
}

func (m *MockAPI) CreateTask(_ context.Context, req *apiv2beta1.CreateTaskRequest) (*apiv2beta1.PipelineTask, error) {
	task := req.Task
	if task != nil && task.GetRunId() != "" && task.GetScopePath() != "" && task.GetName() != "" && task.GetType() != apiv2beta1.PipelineTask_TASK_TYPE_UNSPECIFIED {
		for _, existingTask := range m.tasks {
			if sameLogicalTaskIdentity(existingTask, task, req.GetRunId()) {
				return existingTask, nil
			}
		}
	}
	if task.TaskId == "" {
		uuid, _ := uuid.NewRandom()
		task.TaskId = uuid.String()
	}
	m.tasks[task.TaskId] = task
	return task, nil
}

func (m *MockAPI) UpdateTask(_ context.Context, req *apiv2beta1.UpdateTaskRequest) (*apiv2beta1.PipelineTask, error) {
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
		Tasks: make(map[string]*apiv2beta1.PipelineTask),
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

func (m *MockAPI) GetTask(_ context.Context, req *apiv2beta1.GetTaskRequest) (*apiv2beta1.PipelineTask, error) {
	if _, exists := m.tasks[req.TaskId]; exists {
		task := m.hydrateTask(m.tasks[req.TaskId])
		return task, nil
	}

	return nil, fmt.Errorf("task not found: %s", req.TaskId)
}

func (m *MockAPI) ListTasks(_ context.Context, req *apiv2beta1.ListTasksRequest) (*apiv2beta1.ListTasksResponse, error) {
	var tasks []*apiv2beta1.PipelineTask

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

	// Filter by run ID and optional parent task ID. When both are set, require both.
	runID := req.GetRunId()
	parentID := req.GetParentId()
	for _, task := range m.tasks {
		if runID != "" && task.RunId != runID {
			continue
		}
		if parentID != "" && (task.ParentTaskId == nil || *task.ParentTaskId != parentID) {
			continue
		}
		tasks = append(tasks, task)
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

		var filtered []*apiv2beta1.PipelineTask
		status := statusPredicate.GetIntValue()
		fingerprint := fingerprintPredicate.GetStringValue()
		for _, t := range tasks {
			if int32(t.GetState().Number()) == status && t.GetCacheFingerprint() == fingerprint {
				filtered = append(filtered, t)
			}

		}
		tasks = filtered
	}

	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].GetTaskId() < tasks[j].GetTaskId()
	})

	totalSize := int32(len(tasks))
	start := 0
	if req.GetPageToken() != "" {
		tokenIndex, err := strconv.Atoi(req.GetPageToken())
		if err != nil {
			return nil, fmt.Errorf("invalid page token %q: %w", req.GetPageToken(), err)
		}
		start = tokenIndex
		if start < 0 {
			start = 0
		}
		if start > len(tasks) {
			start = len(tasks)
		}
	}

	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = len(tasks) - start
	}
	end := start + pageSize
	if end > len(tasks) {
		end = len(tasks)
	}

	page := tasks[start:end]
	var hydratedTasks []*apiv2beta1.PipelineTask
	for _, task := range page {
		hydratedTasks = append(hydratedTasks, m.hydrateTask(task))
	}

	nextPageToken := ""
	if end < len(tasks) {
		nextPageToken = strconv.Itoa(end)
	}

	return &apiv2beta1.ListTasksResponse{
		Tasks:         hydratedTasks,
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil
}

func (m *MockAPI) FindCachedTask(_ context.Context, req *apiv2beta1.FindCachedTaskRequest) (*apiv2beta1.FindCachedTaskResponse, error) {
	var matchedTask *apiv2beta1.PipelineTask
	for _, task := range m.tasks {
		if task.GetCacheFingerprint() != req.GetCacheFingerprint() {
			continue
		}
		if task.GetState() != apiv2beta1.PipelineTask_SUCCEEDED {
			continue
		}
		if matchedTask == nil || task.GetCreateTime().GetSeconds() > matchedTask.GetCreateTime().GetSeconds() {
			matchedTask = task
		}
	}
	if matchedTask == nil {
		return &apiv2beta1.FindCachedTaskResponse{}, nil
	}
	return &apiv2beta1.FindCachedTaskResponse{Task: m.hydrateTask(matchedTask)}, nil
}

func (m *MockAPI) CreateArtifact(_ context.Context, req *apiv2beta1.CreateArtifactRequest) (*apiv2beta1.Artifact, error) {
	artifact := req.Artifact
	if req.GetReuseIfExists() {
		for _, existing := range m.artifacts {
			if mockArtifactsEqualForReuse(artifact, existing) {
				taskName := ""
				if task, exists := m.tasks[req.TaskId]; exists && task != nil {
					taskName = task.Name
				}
				outputType := util.OutputIOTypeForIteration(req.IterationIndex)
				for _, existingLink := range m.artifactTasks {
					if existingLink.GetArtifactId() == existing.ArtifactId &&
						existingLink.GetTaskId() == req.TaskId &&
						existingLink.GetKey() == req.ProducerKey &&
						existingLink.GetType() == outputType &&
						mockIterationEqual(existingLink.GetProducer(), req.IterationIndex) {
						return existing, nil
					}
				}
				artifactTask := &apiv2beta1.ArtifactTask{
					ArtifactId: existing.ArtifactId,
					TaskId:     req.TaskId,
					RunId:      req.RunId,
					Type:       outputType,
					Key:        req.ProducerKey,
					Producer: &apiv2beta1.IOProducer{
						TaskName: taskName,
					},
				}
				if req.IterationIndex != nil {
					artifactTask.Producer.Iteration = req.IterationIndex
				}
				if artifactTask.Id == "" {
					id, _ := uuid.NewRandom()
					artifactTask.Id = id.String()
				}
				m.artifactTasks[artifactTask.Id] = artifactTask
				return existing, nil
			}
		}
	}

	if artifact.ArtifactId == "" {
		id, _ := uuid.NewRandom()
		artifact.ArtifactId = id.String()
	}
	m.artifacts[artifact.ArtifactId] = artifact

	taskName := ""
	if task, exists := m.tasks[req.TaskId]; exists && task != nil {
		taskName = task.Name
	}
	artifactTask := &apiv2beta1.ArtifactTask{
		ArtifactId: artifact.ArtifactId,
		TaskId:     req.TaskId,
		RunId:      req.RunId,
		Type:       util.OutputIOTypeForIteration(req.IterationIndex),
		Key:        req.ProducerKey,
		Producer: &apiv2beta1.IOProducer{
			TaskName: taskName,
		},
	}
	if req.IterationIndex != nil {
		artifactTask.Producer.Iteration = req.IterationIndex
	}
	if artifactTask.Id == "" {
		id, _ := uuid.NewRandom()
		artifactTask.Id = id.String()
	}
	m.artifactTasks[artifactTask.Id] = artifactTask

	return artifact, nil
}

func mockIterationEqual(producer *apiv2beta1.IOProducer, iterationIndex *int64) bool {
	var existing *int64
	if producer != nil {
		existing = producer.Iteration
	}
	if existing == nil && iterationIndex == nil {
		return true
	}
	if existing == nil || iterationIndex == nil {
		return false
	}
	return *existing == *iterationIndex
}

func mockArtifactsEqualForReuse(left, right *apiv2beta1.Artifact) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.GetNamespace() != right.GetNamespace() {
		return false
	}
	if left.GetType() != right.GetType() {
		return false
	}
	if left.GetUri() != right.GetUri() {
		return false
	}
	if left.GetName() != right.GetName() {
		return false
	}
	if left.GetDescription() != right.GetDescription() {
		return false
	}
	leftMetadata := left.GetMetadata()
	rightMetadata := right.GetMetadata()
	if len(leftMetadata) != len(rightMetadata) {
		return false
	}
	for key, leftValue := range leftMetadata {
		rightValue, exists := rightMetadata[key]
		if !exists || !proto.Equal(leftValue, rightValue) {
			return false
		}
	}
	return true
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

		taskName := ""
		if task, exists := m.tasks[artifactReq.TaskId]; exists && task != nil {
			taskName = task.Name
		}
		artifactTask := &apiv2beta1.ArtifactTask{
			ArtifactId: artifact.ArtifactId,
			TaskId:     artifactReq.TaskId,
			RunId:      artifactReq.RunId,
			Type:       util.OutputIOTypeForIteration(artifactReq.IterationIndex),
			Key:        artifactReq.ProducerKey,
			Producer: &apiv2beta1.IOProducer{
				TaskName: taskName,
			},
		}
		if artifactReq.IterationIndex != nil {
			artifactTask.Producer.Iteration = artifactReq.IterationIndex
		}
		if artifactTask.Id == "" {
			uuid, _ := uuid.NewRandom()
			artifactTask.Id = uuid.String()
		}
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

func (m *MockAPI) UpdateStatuses(ctx context.Context, run *apiv2beta1.Run, pipelineSpec *structpb.Struct, currentTask *apiv2beta1.PipelineTask) error {
	return updateStatuses(ctx, run, m, pipelineSpec, currentTask)
}
