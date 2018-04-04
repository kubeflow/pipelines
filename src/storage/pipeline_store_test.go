// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"ml/src/message"
	"ml/src/util"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createPipeline(name string, pkgId uint) *message.Pipeline {
	return &message.Pipeline{Name: name, PackageId: pkgId, Parameters: []message.Parameter{}}
}

func pipelineExpected1() message.Pipeline {
	return message.Pipeline{Metadata: &message.Metadata{ID: 1},
		Name:           "pipeline1",
		PackageId:      1,
		Enabled:        true,
		EnabledAtInSec: 1,
		Parameters:     []message.Parameter{}}
}

func TestListPipelines(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore.CreatePipeline(createPipeline("pipeline1", 1))
	store.PipelineStore.CreatePipeline(createPipeline("pipeline2", 2))
	pipelinesExpected := []message.Pipeline{
		pipelineExpected1(),
		{Metadata: &message.Metadata{ID: 2},
			Name:           "pipeline2",
			PackageId:      2,
			Enabled:        true,
			EnabledAtInSec: 2,
			Parameters:     []message.Parameter{}}}

	pipelines, err := store.PipelineStore.ListPipelines()
	assert.Nil(t, err)
	assert.Equal(t, pipelinesExpected, pipelines, "Got unexpected pipelines")
}

func TestListPipelinesError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB.Close()
	_, err := store.PipelineStore.ListPipelines()

	assert.IsType(t, new(util.InternalError), err, "Expected to list pipeline to return error")
}

func TestGetPipeline(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore.CreatePipeline(createPipeline("pipeline1", 1))

	pipeline, err := store.PipelineStore.GetPipeline(1)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected1(), *pipeline, "Got unexpected pipelines")
}

func TestGetPipeline_NotFoundError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	_, err := store.PipelineStore.GetPipeline(1)
	assert.IsType(t, new(util.ResourceNotFoundError), err, "Expected get pipeline to return not found error")
}

func TestGetPipeline_InternalError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB.Close()
	_, err := store.PipelineStore.GetPipeline(1)
	assert.IsType(t, new(util.InternalError), err, "Expected get pipeline to return internal error")
}

func TestCreatePipeline(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline := createPipeline("pipeline1", 1)
	err := store.PipelineStore.CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected1(), *pipeline, "Got unexpected pipelines")
}

func TestCreatePipelineError(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB.Close()

	pipeline := createPipeline("pipeline1", 1)
	err := store.PipelineStore.CreatePipeline(pipeline)
	assert.IsType(t, new(util.InternalError), err, "Expected create pipeline to return error")
}

func TestEnablePipeline(t *testing.T) {

	store, err := NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	// Creating a pipeline. It is enabled by default.
	createdPipeline := &message.Pipeline{Name: "Pipeline123"}
	err = store.PipelineStore.CreatePipeline(createdPipeline)
	assert.Nil(t, err)
	pipelineID := createdPipeline.ID

	// Verify that the created pipeline is enabled.
	createdPipeline, err = store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, createdPipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(1), createdPipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")
	assert.Equal(t, int64(0), createdPipeline.UpdatedAtInSec, "Unexpected value of UpdatedAtInSec.")

	// Verify that enabling the pipeline has no effect. In particular, EnabledAtInSec should
	// not change.
	err = store.PipelineStore.EnablePipeline(pipelineID, true)
	assert.Nil(t, err)
	pipeline, err := store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(1), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that disabling the pipeline changes both Enabled and EnabledAtInSec
	err = store.PipelineStore.EnablePipeline(pipelineID, false)
	assert.Nil(t, err)
	pipeline, err = store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, false, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(2), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that disabling again as no effect.
	err = store.PipelineStore.EnablePipeline(pipelineID, false)
	assert.Nil(t, err)
	pipeline, err = store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, false, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(2), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that enabling the pipeline changes both Enabled and EnabledAtInSec
	err = store.PipelineStore.EnablePipeline(pipelineID, true)
	assert.Nil(t, err)
	pipeline, err = store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(3), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that none of the fields of the pipeline have changed.
	createdPipeline.UpdatedAt = pipeline.UpdatedAt
	createdPipeline.EnabledAtInSec = pipeline.EnabledAtInSec
	createdPipeline.UpdatedAtInSec = pipeline.UpdatedAtInSec
	assert.Equal(t, createdPipeline, pipeline)
}

func TestEnablePipelineRecordNotFound(t *testing.T) {
	store, err := NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	err = store.PipelineStore.EnablePipeline(12, true)
	assert.IsType(t, &util.UserError{}, err)
	assert.Contains(t, err.(*util.UserError).Internal().Error(), "record not found")
	assert.IsType(t, &util.ResourceNotFoundError{}, err.(*util.UserError).External())
}

func TestEnablePipelineDatabaseError(t *testing.T) {
	store, err := NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	// Creating a pipeline. It is enabled by default.
	createdPipeline := &message.Pipeline{Name: "Pipeline123"}
	err = store.PipelineStore.CreatePipeline(createdPipeline)
	assert.Nil(t, err)
	pipelineID := createdPipeline.ID

	// Closing the DB.
	store.Close()

	// Enabling the pipeline.
	err = store.PipelineStore.EnablePipeline(pipelineID, true)
	assert.Contains(t, err.Error(), "Error when enabling pipeline 1 to true: sql: database is closed")
}

func TestGetPipelineAndLatestJobIteratorPipelineWithoutJob(t *testing.T) {
	store := NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	pipeline1 := &message.Pipeline{
		Name:      "MY_PIPELINE_1",
		PackageId: 123,
		Schedule:  "1 0 * * *"}

	pipeline2 := &message.Pipeline{
		Name:      "MY_PIPELINE_2",
		PackageId: 123,
		Schedule:  "1 0 * * 1"}

	workflow1 := v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_1",
		},
	}

	workflow2 := v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_2",
		},
	}

	store.PipelineStore.CreatePipeline(pipeline1)
	store.PipelineStore.CreatePipeline(pipeline2)
	store.JobStore.CreateJob(1, &workflow1)
	store.JobStore.CreateJob(1, &workflow2)

	// Checking the first row, which does not have a job.
	iterator, err := store.PipelineStore.GetPipelineAndLatestJobIterator()

	assert.Nil(t, err)
	assert.True(t, iterator.Next())

	result, err := iterator.Get()
	assert.Nil(t, err)

	expected := &PipelineAndLatestJob{
		PipelineID:             "2",
		PipelineName:           pipeline2.Name,
		PipelineSchedule:       pipeline2.Schedule,
		JobName:                nil,
		JobScheduledAtInSec:    nil,
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 2,
	}

	assert.Equal(t, expected, result)

	// Checking the second row, which has a job.
	assert.True(t, iterator.Next())

	result, err = iterator.Get()
	assert.Nil(t, err)

	scheduledSec := int64(4)
	expected = &PipelineAndLatestJob{
		PipelineID:             "1",
		PipelineName:           pipeline1.Name,
		PipelineSchedule:       pipeline1.Schedule,
		JobName:                &workflow2.Name,
		JobScheduledAtInSec:    &scheduledSec,
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 1,
	}

	assert.Equal(t, expected, result)

	// Checking that there are no rows left.
	assert.False(t, iterator.Next())

}

func TestGetPipelineAndLatestJobIteratorPipelineWithoutSchedule(t *testing.T) {
	store, err := NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	pipeline1 := &message.Pipeline{
		Name:      "MY_PIPELINE_1",
		PackageId: 123,
		Schedule:  "1 0 * * *"}

	pipeline2 := &message.Pipeline{
		Name:      "MY_PIPELINE_2",
		PackageId: 123,
		Schedule:  ""}

	workflow1 := v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_1",
		},
	}

	workflow2 := v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "MY_WORKFLOW_NAME_2",
		},
	}

	store.PipelineStore.CreatePipeline(pipeline1)
	store.PipelineStore.CreatePipeline(pipeline2)
	store.JobStore.CreateJob(1, &workflow1)
	store.JobStore.CreateJob(1, &workflow2)

	// Checking the first row, which does not have a job.
	iterator, err := store.PipelineStore.GetPipelineAndLatestJobIterator()

	assert.Nil(t, err)
	assert.True(t, iterator.Next())

	result, err := iterator.Get()
	assert.Nil(t, err)

	scheduledSec := int64(4)
	expected := &PipelineAndLatestJob{
		PipelineID:             "1",
		PipelineName:           pipeline1.Name,
		PipelineSchedule:       pipeline1.Schedule,
		JobName:                &workflow2.Name,
		JobScheduledAtInSec:    &scheduledSec,
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 1,
	}

	assert.Equal(t, expected, result)

	// Checking that there are no rows left.
	assert.False(t, iterator.Next())

}
