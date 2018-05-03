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
	"ml/backend/src/model"
	"ml/backend/src/util"
	"net/http"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createPipeline(name string, packageId uint) *model.Pipeline {
	return &model.Pipeline{Name: name, PackageId: packageId, Parameters: []model.Parameter{}, Status: model.PipelineReady}
}

func pipelineExpected1() model.Pipeline {
	return model.Pipeline{
		ID:             1,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		Name:           "pipeline1",
		PackageId:      1,
		Enabled:        true,
		EnabledAtInSec: 1,
		Parameters:     []model.Parameter{},
		Status:         model.PipelineReady}
}

func TestListPipelines(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore().CreatePipeline(createPipeline("pipeline1", 1))
	store.PipelineStore().CreatePipeline(createPipeline("pipeline2", 2))
	store.PipelineStore().CreatePipeline(&model.Pipeline{Name: "pipeline3", PackageId: 3, Status: model.PipelineCreating})
	pipelinesExpected := []model.Pipeline{
		pipelineExpected1(),
		{
			ID:             2,
			CreatedAtInSec: 2,
			UpdatedAtInSec: 2,
			Name:           "pipeline2",
			PackageId:      2,
			Enabled:        true,
			EnabledAtInSec: 2,
			Parameters:     []model.Parameter{},
			Status:         model.PipelineReady,
		}}

	pipelines, err := store.PipelineStore().ListPipelines()
	assert.Nil(t, err)
	assert.Equal(t, pipelinesExpected, pipelines, "Got unexpected pipelines")
}

func TestListPipelinesError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	_, err := store.PipelineStore().ListPipelines()

	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode(),
		"Expected to list pipeline to return error")
}

func TestGetPipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore().CreatePipeline(createPipeline("pipeline1", 1))

	pipeline, err := store.PipelineStore().GetPipeline(1)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected1(), *pipeline, "Got unexpected pipelines")
}

func TestGetPipeline_NotFound_Creating(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore().CreatePipeline(&model.Pipeline{Name: "pipeline3", PackageId: 3, Status: model.PipelineCreating})

	_, err := store.PipelineStore().GetPipeline(1)
	assert.Equal(t, http.StatusNotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found error")
}

func TestGetPipeline_NotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	_, err := store.PipelineStore().GetPipeline(1)
	assert.Equal(t, http.StatusNotFound, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return not found error")
}

func TestGetPipeline_InternalError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	_, err := store.PipelineStore().GetPipeline(1)
	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode(),
		"Expected get pipeline to return internal error")
}

func TestDeletePipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore().CreatePipeline(createPipeline("pipeline1", 1))

	err := store.PipelineStore().DeletePipeline(1)
	assert.Nil(t, err)
	_, err = store.PipelineStore().GetPipeline(1)
	assert.Equal(t, http.StatusNotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestDeletePipeline_InternalError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	err := store.PipelineStore().DeletePipeline(1)
	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode(),
		"Expected delete pipeline to return internal error")
}

func TestCreatePipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline := createPipeline("pipeline1", 1)
	pipeline, err := store.PipelineStore().CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected1(), *pipeline, "Got unexpected pipelines")
}

func TestCreatePipelineError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()

	pipeline := createPipeline("pipeline1", 1)
	pipeline, err := store.PipelineStore().CreatePipeline(pipeline)
	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode(),
		"Expected create pipeline to return error")
}

func TestEnablePipeline(t *testing.T) {

	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	// Creating a pipeline. It is enabled by default.
	createdPipeline := &model.Pipeline{Name: "Pipeline123", Status: model.PipelineReady}
	createdPipeline, err := store.PipelineStore().CreatePipeline(createdPipeline)
	assert.Nil(t, err)
	pipelineID := createdPipeline.ID

	// Verify that the created pipeline is enabled.
	createdPipeline, err = store.PipelineStore().GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, createdPipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(1), createdPipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")
	assert.Equal(t, int64(1), createdPipeline.UpdatedAtInSec, "Unexpected value of UpdatedAtInSec.")

	// Verify that enabling the pipeline has no effect. In particular, EnabledAtInSec should
	// not change.
	err = store.PipelineStore().EnablePipeline(pipelineID, true)
	assert.Nil(t, err)
	pipeline, err := store.PipelineStore().GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(1), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that disabling the pipeline changes both Enabled and EnabledAtInSec
	err = store.PipelineStore().EnablePipeline(pipelineID, false)
	assert.Nil(t, err)
	pipeline, err = store.PipelineStore().GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, false, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(2), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that disabling again as no effect.
	err = store.PipelineStore().EnablePipeline(pipelineID, false)
	assert.Nil(t, err)
	pipeline, err = store.PipelineStore().GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, false, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(2), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that enabling the pipeline changes both Enabled and EnabledAtInSec
	err = store.PipelineStore().EnablePipeline(pipelineID, true)
	assert.Nil(t, err)
	pipeline, err = store.PipelineStore().GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, pipeline.Enabled, "The pipeline must be enabled.")
	assert.Equal(t, int64(3), pipeline.EnabledAtInSec, "Unexpected value of EnabledAtInSec.")

	// Verify that none of the fields of the pipeline have changed.
	createdPipeline.EnabledAtInSec = pipeline.EnabledAtInSec
	createdPipeline.UpdatedAtInSec = pipeline.UpdatedAtInSec
	assert.Equal(t, createdPipeline, pipeline)
}

func TestEnablePipelineRecordNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	err := store.PipelineStore().EnablePipeline(12, true)
	assert.IsType(t, &util.UserError{}, err)
	assert.Equal(t, http.StatusNotFound, err.(*util.UserError).ExternalStatusCode())
}

func TestEnablePipelineDatabaseError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	// Creating a pipeline. It is enabled by default.
	createdPipeline := &model.Pipeline{Name: "Pipeline123"}
	createdPipeline, err := store.PipelineStore().CreatePipeline(createdPipeline)
	assert.Nil(t, err)
	pipelineID := createdPipeline.ID

	// Closing the DB.
	store.Close()

	// Enabling the pipeline.
	err = store.PipelineStore().EnablePipeline(pipelineID, true)
	assert.Contains(t, err.Error(), "Error when enabling pipeline 1 to true: sql: database is closed")
}

func TestUpdatePipelineStatus(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline, err := store.PipelineStore().CreatePipeline(&model.Pipeline{Name: "pipeline1", PackageId: 1, Parameters: []model.Parameter{}, Status: model.PipelineCreating})
	assert.Nil(t, err)
	err = store.PipelineStore().UpdatePipelineStatus(pipeline.ID, model.PipelineReady)

	store.DB().First(&pipeline, pipeline.ID)
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected1(), *pipeline, "Got unexpected pipelines")
}

func TestUpdatePipelineStatusError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	err := store.PipelineStore().UpdatePipelineStatus(1, model.PipelineReady)
	assert.Equal(t, http.StatusInternalServerError, err.(*util.UserError).ExternalStatusCode())
}

func TestGetPipelineAndLatestJobIteratorPipelineWithoutJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	pipeline1 := &model.Pipeline{
		Name:      "MY_PIPELINE_1",
		PackageId: 123,
		Schedule:  "1 0 * * *",
		Status:    model.PipelineReady}

	pipeline2 := &model.Pipeline{
		Name:      "MY_PIPELINE_2",
		PackageId: 123,
		Schedule:  "1 0 * * 1",
		Status:    model.PipelineReady}

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

	store.PipelineStore().CreatePipeline(pipeline1)
	store.PipelineStore().CreatePipeline(pipeline2)
	store.JobStore().CreateJob(1, &workflow1, defaultScheduledAtInSec, defaultCreatedAtInSec)
	store.JobStore().CreateJob(1, &workflow2, defaultScheduledAtInSec+5, defaultCreatedAtInSec)

	// Checking the first row, which does not have a job.
	iterator, err := store.PipelineStore().GetPipelineAndLatestJobIterator()

	assert.Nil(t, err)
	assert.True(t, iterator.Next())

	result, err := iterator.Get()
	assert.Nil(t, err)

	expected := &PipelineAndLatestJob{
		PipelineID:             2,
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

	expected = &PipelineAndLatestJob{
		PipelineID:             1,
		PipelineName:           pipeline1.Name,
		PipelineSchedule:       pipeline1.Schedule,
		JobName:                &workflow2.Name,
		JobScheduledAtInSec:    util.Int64Pointer(defaultScheduledAtInSec + 5),
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 1,
	}

	assert.Equal(t, expected, result)

	// Checking that there are no rows left.
	assert.False(t, iterator.Next())

}

func TestGetPipelineAndLatestJobIteratorPipelineWithoutSchedule(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	pipeline1 := &model.Pipeline{
		Name:      "MY_PIPELINE_1",
		PackageId: 123,
		Schedule:  "1 0 * * *",
		Status:    model.PipelineReady}

	pipeline2 := &model.Pipeline{
		Name:      "MY_PIPELINE_2",
		PackageId: 123,
		Schedule:  "",
		Status:    model.PipelineReady}

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

	store.PipelineStore().CreatePipeline(pipeline1)
	store.PipelineStore().CreatePipeline(pipeline2)
	store.JobStore().CreateJob(1, &workflow1, defaultScheduledAtInSec+5, defaultCreatedAtInSec)
	store.JobStore().CreateJob(1, &workflow2, defaultScheduledAtInSec, defaultCreatedAtInSec)

	// Checking the first row, which does not have a job.
	iterator, err := store.PipelineStore().GetPipelineAndLatestJobIterator()

	assert.Nil(t, err)
	assert.True(t, iterator.Next())

	result, err := iterator.Get()
	assert.Nil(t, err)

	expected := &PipelineAndLatestJob{
		PipelineID:             1,
		PipelineName:           pipeline1.Name,
		PipelineSchedule:       pipeline1.Schedule,
		JobName:                &workflow1.Name,
		JobScheduledAtInSec:    util.Int64Pointer(defaultScheduledAtInSec + 5),
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 1,
	}

	assert.Equal(t, expected, result)

	// Checking that there are no rows left.
	assert.False(t, iterator.Next())

}
