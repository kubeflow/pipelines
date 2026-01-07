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

package storage

import (
	"fmt"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

const (
	linkUUID1   = "123e4567-e89b-12d3-a456-426655441011"
	linkUUID2   = "123e4567-e89b-12d3-a456-426655441012"
	linkUUID3   = "123e4567-e89b-12d3-a456-426655441013"
	artifactID1 = "123e4567-e89b-12d3-a456-426655441013"
	artifactID2 = "123e4567-e89b-12d3-a456-426655441014"
	taskID1     = "123e4567-e89b-12d3-a456-426655441015"
	taskID2     = "123e4567-e89b-12d3-a456-426655441016"
	runID1      = "123e4567-e89b-12d3-a456-426655441017"
	runID2      = "123e4567-e89b-12d3-a456-426655441018"
)

// initializeArtifactTaskDeps sets up a fake DB and returns stores needed for artifact-task tests.
func initializeArtifactTaskDeps() (*DB, *ArtifactStore, *TaskStore, *RunStore, *ArtifactTaskStore) {
	db := NewFakeDBOrFatal()
	fakeTime := util.NewFakeTimeForEpoch()

	artifactStore := NewArtifactStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(artifactID1, nil))
	taskStore := NewTaskStore(db, fakeTime, util.NewFakeUUIDGeneratorOrFatal(taskID1, nil))
	runStore := NewRunStore(db, fakeTime)
	linkStore := NewArtifactTaskStore(db, util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil))

	// Seed runs to satisfy Task FK
	_, _ = runStore.CreateRun(&model.Run{UUID: runID1, ExperimentId: "exp-1", K8SName: "r1", DisplayName: "r1", StorageState: model.StorageStateAvailable, Namespace: "ns1", RunDetails: model.RunDetails{CreatedAtInSec: 1, ScheduledAtInSec: 1, State: model.RuntimeStateRunning}})
	_, _ = runStore.CreateRun(&model.Run{UUID: runID2, ExperimentId: "exp-2", K8SName: "r2", DisplayName: "r2", StorageState: model.StorageStateAvailable, Namespace: "ns2", RunDetails: model.RunDetails{CreatedAtInSec: 2, ScheduledAtInSec: 2, State: model.RuntimeStateSucceeded}})

	return db, artifactStore, taskStore, runStore, linkStore
}

func TestArtifactTaskAPIFieldMap(t *testing.T) {
	for _, modelField := range (&model.ArtifactTask{}).APIToModelFieldMap() {
		assert.Contains(t, artifactTaskColumns, fmt.Sprintf("%s.%s", artifactTaskTableName, modelField))
	}
}

func TestCreateArtifactTask_Success(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Create an artifact and a task to link
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID1, nil)
	art, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("s3://b/p1"),
		Name:      "a1",
		Metadata:  map[string]interface{}{"k": "v"},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskID1, nil)
	task, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          runID1,
		Name:             "t1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp1",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Link as INPUT
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	link, err := linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art.UUID,
		TaskID:      task.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input-key",
	})
	assert.NoError(t, err)
	assert.Equal(t, linkUUID1, link.UUID)
	assert.Equal(t, art.UUID, link.ArtifactID)
	assert.Equal(t, task.UUID, link.TaskID)
	assert.Equal(t, model.IOType(apiv2beta1.IOType_COMPONENT_INPUT), link.Type)

	// Fetch back
	got, err := linkStore.GetArtifactTask(link.UUID)
	assert.NoError(t, err)
	assert.Equal(t, link.UUID, got.UUID)
	assert.Equal(t, link.ArtifactID, got.ArtifactID)
	assert.Equal(t, link.TaskID, got.TaskID)
	assert.Equal(t, link.Type, got.Type)
}

func TestListArtifactTasks_Filters(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Create 2 artifacts
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID1, nil)
	art1, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u1"),
		Name:      "a1",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID2, nil)
	art2, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u2"),
		Name:      "a2",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Create 2 tasks across 2 runs
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskID1, nil)
	t1, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          runID1,
		Name:             "t1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-1",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)
	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskID2, nil)
	t2, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		RunUUID:          runID2,
		Name:             "t2",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p2", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-2",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Create links: art1<->t1 (INPUT), art2<->t1 (OUTPUT), art2<->t2 (INPUT)
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art1.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input1",
	})
	assert.NoError(t, err)
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID2, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art2.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output1",
	})
	assert.NoError(t, err)
	// another link with a fresh random UUID
	linkStore.uuid = util.NewUUIDGenerator()
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art2.UUID,
		TaskID:      t2.UUID,
		RunUUID:     runID2,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input2",
	})
	assert.NoError(t, err)

	opts, _ := list.NewOptions(&model.ArtifactTask{}, 20, "", nil)

	// List all
	all, total, npt, err := linkStore.ListArtifactTasks(nil, nil, opts)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(all))
	assert.Equal(t, 3, total)
	assert.Equal(t, "", npt)

	// Filter by task t1
	byTask, totalTask, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: t1.UUID}}}, nil, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(byTask))
	assert.Equal(t, 2, totalTask)

	// Filter by artifact art2
	byArtifact, totalArt, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.ArtifactResourceType, ID: art2.UUID}}}, nil, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(byArtifact)) // art2 is linked twice
	assert.Equal(t, 2, totalArt)

	// Filter by run runID2 (should return only links for tasks in run-2)
	byRun, totalRun, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: runID2}}}, nil, opts)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(byRun))
	assert.Equal(t, 1, totalRun)
	assert.Equal(t, art2.UUID, byRun[0].ArtifactID)
	assert.Equal(t, t2.UUID, byRun[0].TaskID)
	assert.Equal(t, model.IOType(apiv2beta1.IOType_COMPONENT_INPUT), byRun[0].Type)
}

func TestListArtifactsForTask_UsingArtifactTasks(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Seed artifacts and a single task
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID1, nil)
	art1, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u1"),
		Name:      "a1",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID2, nil)
	art2, err := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u2"),
		Name:      "a2",
		Metadata:  map[string]interface{}{},
	})
	assert.NoError(t, err)

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskID1, nil)
	t1, err := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          runID1,
		Name:             "t1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-1",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})
	assert.NoError(t, err)

	// Link both artifacts to t1
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art1.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input1",
	})
	assert.NoError(t, err)
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID2, nil)
	_, err = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art2.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output1",
	})
	assert.NoError(t, err)

	// Use artifactTasks to list artifacts for task t1
	opts, _ := list.NewOptions(&model.ArtifactTask{}, 20, "", nil)
	rows, total, _, err := linkStore.ListArtifactTasks([]*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: t1.UUID}}}, nil, opts)
	assert.NoError(t, err)
	assert.Equal(t, 2, total)

	// Collect artifact IDs and verify set equals {art1, art2}
	ids := map[string]bool{}
	for _, r := range rows {
		ids[r.ArtifactID] = true
	}
	assert.True(t, ids[art1.UUID])
	assert.True(t, ids[art2.UUID])
	assert.Equal(t, 2, len(ids))
}

func TestListArtifactTasks_Pagination_PageSizeAndNextPageToken(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Seed artifacts and a task
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID1, nil)
	art1, _ := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u1"),
		Name:      "a1",
		Metadata:  map[string]interface{}{},
	})

	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID2, nil)
	art2, _ := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u2"),
		Name:      "a2",
		Metadata:  map[string]interface{}{},
	})

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskID1, nil)
	t1, _ := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          runID1,
		Name:             "t1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-1",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})

	// Create 3 links with deterministic UUID order
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	_, _ = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art1.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input1",
	})

	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID2, nil)
	_, _ = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art1.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output1",
	})

	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID3, nil)
	_, _ = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art2.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input2",
	})

	// Page 1: size 2
	opts1, _ := list.NewOptions(&model.ArtifactTask{}, 2, "", nil)
	page1, total, token1, err := linkStore.ListArtifactTasks(nil, nil, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(page1))
	assert.Equal(t, 3, total)
	assert.NotEmpty(t, token1)
	// should be ordered by UUID asc by default
	assert.Equal(t, linkUUID1, page1[0].UUID)
	assert.Equal(t, linkUUID2, page1[1].UUID)

	// Page 2: use token
	opts2, err := list.NewOptionsFromToken(token1, 2)
	assert.NoError(t, err)
	page2, total2, token2, err := linkStore.ListArtifactTasks(nil, nil, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(page2))
	assert.Equal(t, 3, total2)
	assert.Equal(t, "", token2)
	assert.Equal(t, linkUUID3, page2[0].UUID)
}

func TestListArtifactTasks_Pagination_WithFilter(t *testing.T) {
	db, artifactStore, taskStore, _, linkStore := initializeArtifactTaskDeps()
	defer db.Close()

	// Seed artifacts and tasks
	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID1, nil)
	art1, _ := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u1"),
		Name:      "a1",
		Metadata:  map[string]interface{}{},
	})

	artifactStore.uuid = util.NewFakeUUIDGeneratorOrFatal(artifactID2, nil)
	art2, _ := artifactStore.CreateArtifact(&model.Artifact{
		Namespace: "ns1",
		Type:      1,
		URI:       strPTR("u2"),
		Name:      "a2",
		Metadata:  map[string]interface{}{},
	})

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskID1, nil)
	t1, _ := taskStore.CreateTask(&model.Task{
		Namespace:        "ns1",
		RunUUID:          runID1,
		Name:             "t1",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-1",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})

	taskStore.uuid = util.NewFakeUUIDGeneratorOrFatal(taskID2, nil)
	t2, _ := taskStore.CreateTask(&model.Task{
		Namespace:        "ns2",
		RunUUID:          runID2,
		Name:             "t2",
		Pods:             createTaskPodsAsJSONSlice(createTaskPod("p1", "uid1", apiv2beta1.PipelineTaskDetail_EXECUTOR)),
		Fingerprint:      "fp-2",
		State:            1,
		StateHistory:     model.JSONSlice{},
		InputParameters:  model.JSONSlice{},
		OutputParameters: model.JSONSlice{},
		Type:             0,
		TypeAttrs:        map[string]interface{}{},
	})

	// Links: 2 for t1, 1 for t2
	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID1, nil)
	_, _ = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art1.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input1",
	})

	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID2, nil)
	_, _ = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art2.UUID,
		TaskID:      t1.UUID,
		RunUUID:     runID1,
		Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
		ArtifactKey: "output1",
	})

	linkStore.uuid = util.NewFakeUUIDGeneratorOrFatal(linkUUID3, nil)
	_, _ = linkStore.CreateArtifactTask(&model.ArtifactTask{
		ArtifactID:  art2.UUID,
		TaskID:      t2.UUID,
		RunUUID:     runID2,
		Type:        model.IOType(apiv2beta1.IOType_COMPONENT_INPUT),
		ArtifactKey: "input2",
	})

	filterByT1 := []*model.FilterContext{{ReferenceKey: &model.ReferenceKey{Type: model.TaskResourceType, ID: t1.UUID}}}

	// Page size 1 for filtered list
	opts1, _ := list.NewOptions(&model.ArtifactTask{}, 1, "", nil)
	p1, total1, tok1, err := linkStore.ListArtifactTasks(filterByT1, nil, opts1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(p1))
	assert.Equal(t, 2, total1)
	assert.NotEmpty(t, tok1)

	// Second page
	opts2, err := list.NewOptionsFromToken(tok1, 1)
	assert.NoError(t, err)
	p2, total2, tok2, err := linkStore.ListArtifactTasks(filterByT1, nil, opts2)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(p2))
	assert.Equal(t, 2, total2)
	assert.Equal(t, "", tok2)

	// Ensure the two pages are disjoint and together contain the two t1 links
	ids := map[string]bool{p1[0].UUID: true}
	assert.False(t, ids[p2[0].UUID])
	ids[p2[0].UUID] = true
	assert.Len(t, ids, 2)
}
