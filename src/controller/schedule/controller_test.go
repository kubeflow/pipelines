// Copyright 2018 The Kubeflow Authors
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

package schedule

import (
	"ml/src/model"
	"ml/src/resource"
	"ml/src/storage"
	"ml/src/util"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	timeFarInTheFutureSec = 99999999
)

func getDefaultPipelineAndLatestJob() *storage.PipelineAndLatestJob {
	return &storage.PipelineAndLatestJob{
		PipelineID:             1,
		PipelineName:           "PIPELINE_NAME",
		PipelineSchedule:       "* 1 * * * *",
		JobName:                util.StringPointer("JOB_NAME"),
		JobScheduledAtInSec:    util.Int64Pointer(10),
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 20,
	}
}

func createPkg(name string) *model.Package {
	return &model.Package{Name: name}
}

func TestGetNextStartTimeAfter(t *testing.T) {

	// Note: the epoch is "1970-01-01 00:00:00 +0000 UTC"
	epoch := time.Unix(0, 0).UTC()

	// Next start time as soon as possible
	actual, err := getNextStartTimeAfter("* * * * * *", epoch)
	assert.Nil(t, err)
	expected, err := time.Parse(time.RFC3339, "1970-01-01T00:00:01+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)

	// Seconds
	actual, err = getNextStartTimeAfter("30 * * * * *", epoch)
	assert.Nil(t, err)
	expected, err = time.Parse(time.RFC3339, "1970-01-01T00:00:30+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)

	actual, err = getNextStartTimeAfter("0 * * * * *", epoch)
	assert.Nil(t, err)
	expected, err = time.Parse(time.RFC3339, "1970-01-01T00:01:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)

	// Minutes
	actual, err = getNextStartTimeAfter("* 2 * * * *", epoch)
	assert.Nil(t, err)
	expected, err = time.Parse(time.RFC3339, "1970-01-01T00:02:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)

	// Hours
	actual, err = getNextStartTimeAfter("* * 3 * * *", epoch)
	assert.Nil(t, err)
	expected, err = time.Parse(time.RFC3339, "1970-01-01T03:00:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)

	// Day of the month
	actual, err = getNextStartTimeAfter("* * * 4 * *", epoch)
	assert.Nil(t, err)
	expected, err = time.Parse(time.RFC3339, "1970-01-04T00:00:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)

	// Month
	actual, err = getNextStartTimeAfter("* * * * 5 *", epoch)
	assert.Nil(t, err)
	expected, err = time.Parse(time.RFC3339, "1970-05-01T00:00:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)

	// Day of the week
	// Note: the epoch is a Thursday, the schedule requests a Saturday
	actual, err = getNextStartTimeAfter("* * * * * 6", epoch)
	assert.Nil(t, err)
	expected, err = time.Parse(time.RFC3339, "1970-01-03T00:00:00+00:00")
	assert.Nil(t, err)
	assert.Equal(t, expected.UTC(), actual)
}

func TestGetNextStartTimeAfterInvalidSchedule(t *testing.T) {
	actual, err := getNextStartTimeAfter("* * * * 0 *", time.Unix(0, 0))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unparseable schedule: * * * * 0 *")
	assert.Equal(t, time.Unix(0, 0).UTC(), actual)
}

func TestMustRun(t *testing.T) {

	// We use a schedule that runs jobs every hours.
	schedule := "* 0 * * * *"

	// Pipeline enabled after last job, current time is before next run.
	lastJobRunAt, _ := time.Parse(time.RFC3339, "1970-01-01T01:01:00+00:00")
	pipelineEnabledAt, _ := time.Parse(time.RFC3339, "1970-01-01T02:01:00+00:00")
	now, _ := time.Parse(time.RFC3339, "1970-01-01T00:00:00+00:00")
	expected, _ := time.Parse(time.RFC3339, "1970-01-01T03:00:00+00:00")

	run, scheduledTime, err := mustRun(schedule, lastJobRunAt.UTC(), now.UTC(),
		pipelineEnabledAt.UTC())
	assert.Nil(t, err)
	assert.False(t, run)
	assert.Equal(t, expected.UTC(), scheduledTime)

	// Pipeline enabled before last job, current time is before next run.
	lastJobRunAt, _ = time.Parse(time.RFC3339, "1970-01-01T11:01:00+00:00")
	pipelineEnabledAt, _ = time.Parse(time.RFC3339, "1970-01-01T10:01:00+00:00")
	now, _ = time.Parse(time.RFC3339, "1970-01-01T00:00:00+00:00")
	expected, _ = time.Parse(time.RFC3339, "1970-01-01T12:00:00+00:00")

	run, scheduledTime, err = mustRun(schedule, lastJobRunAt.UTC(), now.UTC(),
		pipelineEnabledAt.UTC())
	assert.Nil(t, err)
	assert.False(t, run)
	assert.Equal(t, expected.UTC(), scheduledTime)

	// Pipeline enabled after last job, current time is after next run.
	lastJobRunAt, _ = time.Parse(time.RFC3339, "1970-01-01T01:01:00+00:00")
	pipelineEnabledAt, _ = time.Parse(time.RFC3339, "1970-01-01T02:01:00+00:00")
	now, _ = time.Parse(time.RFC3339, "1970-01-10T00:00:00+00:00")
	expected, _ = time.Parse(time.RFC3339, "1970-01-01T03:00:00+00:00")

	run, scheduledTime, err = mustRun(schedule, lastJobRunAt.UTC(), now.UTC(),
		pipelineEnabledAt.UTC())
	assert.Nil(t, err)
	assert.True(t, run)
	assert.Equal(t, expected.UTC(), scheduledTime)

	// Pipeline enabled before last job, current time is after next run.
	lastJobRunAt, _ = time.Parse(time.RFC3339, "1970-01-01T11:01:00+00:00")
	pipelineEnabledAt, _ = time.Parse(time.RFC3339, "1970-01-01T10:01:00+00:00")
	now, _ = time.Parse(time.RFC3339, "1970-01-10T00:00:00+00:00")
	expected, _ = time.Parse(time.RFC3339, "1970-01-01T12:00:00+00:00")

	run, scheduledTime, err = mustRun(schedule, lastJobRunAt.UTC(), now.UTC(),
		pipelineEnabledAt.UTC())
	assert.Nil(t, err)
	assert.True(t, run)
	assert.Equal(t, expected.UTC(), scheduledTime)

}

func TestMustRunWrongSchedule(t *testing.T) {

	// We use a schedule that runs jobs every hours.
	schedule := "WRONG_SCHEDULE"

	// Pipeline enabled after last job, current time is before next run.
	lastJobRunAt, _ := time.Parse(time.RFC3339, "1970-01-01T01:01:00+00:00")
	pipelineEnabledAt, _ := time.Parse(time.RFC3339, "1970-01-01T02:01:00+00:00")
	now, _ := time.Parse(time.RFC3339, "1970-01-01T00:00:00+00:00")
	expected := time.Unix(0, 0).UTC()

	run, scheduledTime, err := mustRun(schedule, lastJobRunAt.UTC(), now.UTC(),
		pipelineEnabledAt.UTC())
	assert.Contains(t, err.Error(),
		"Could not determine the next time to run a job. Schedule: WRONG_SCHEDULE")
	assert.False(t, run)
	assert.Equal(t, expected, scheduledTime)
}

func TestRunForSingleRowJobDoesNotRun(t *testing.T) {

	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	controller := NewController(resource.NewResourceManager(store))

	hasRun, scheduledTime, err := controller.runForSingleRow(&storage.PipelineAndLatestJob{
		PipelineID:             100,
		PipelineName:           "PIPELINE_NAME",
		PipelineSchedule:       "* 1 * * * *",
		JobName:                util.StringPointer("JOB_NAME"),
		JobScheduledAtInSec:    util.Int64Pointer(10),
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 20,
	})

	assert.False(t, hasRun)
	assert.Equal(t, util.ParseTimeOrFatal("1970-01-01T00:01:00+00:00"), scheduledTime)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())
}

func TestRunForSingleRowJobRuns(t *testing.T) {

	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTime(
		time.Unix(timeFarInTheFutureSec, 0)))
	defer store.Close()
	manager := resource.NewResourceManager(store)
	controller := NewController(manager)

	// Create package and pipeline.
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		Schedule:  "* * * * * *",
		PackageId: 1}
	_, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Create job.
	hasRun, scheduledTime, err := controller.runForSingleRow(getDefaultPipelineAndLatestJob())

	assert.Nil(t, err)
	assert.True(t, hasRun)
	assert.Equal(t, util.ParseTimeOrFatal("1970-01-01T00:01:00+00:00"), scheduledTime)
	assert.Equal(t, 1, store.WorkflowClientFake().GetWorkflowCount())
}

func TestRunForSingleRowInvalidPipelineParameters(t *testing.T) {

	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTime(time.Unix(0, 0)))
	defer store.Close()

	controller := NewController(resource.NewResourceManager(store))

	pipeline := getDefaultPipelineAndLatestJob()
	pipeline.PipelineSchedule = ""
	hasRun, scheduledTime, err := controller.runForSingleRow(pipeline)
	assert.False(t, hasRun)
	assert.Equal(t, time.Unix(0, 0).UTC(), scheduledTime)
	assert.Contains(t, err.Error(), "The schedule should not be empty")

	pipeline = getDefaultPipelineAndLatestJob()
	pipeline.PipelineSchedule = "INVALID_SCHEDULE"
	hasRun, scheduledTime, err = controller.runForSingleRow(pipeline)
	assert.False(t, hasRun)
	assert.Equal(t, time.Unix(0, 0).UTC(), scheduledTime)
	assert.Contains(t, err.Error(), "Could not figure out whether a job should be created")

	pipeline = getDefaultPipelineAndLatestJob()
	pipeline.PipelineEnabledAtInSec = 0
	hasRun, scheduledTime, err = controller.runForSingleRow(pipeline)
	assert.False(t, hasRun)
	assert.Equal(t, time.Unix(0, 0).UTC(), scheduledTime)
	assert.Contains(t, err.Error(), "PipelineEnabledAtInSec should not be 0")
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())
}

func TestRunForSingleRowNoPreviousJobAndDoesNotRun(t *testing.T) {

	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTime(time.Unix(0, 0)))
	defer store.Close()

	controller := NewController(resource.NewResourceManager(store))

	hasRun, scheduledTime, err := controller.runForSingleRow(&storage.PipelineAndLatestJob{
		PipelineID:             100,
		PipelineName:           "PIPELINE_NAME",
		PipelineSchedule:       "* 1 * * * *",
		JobName:                nil,
		JobScheduledAtInSec:    nil,
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 20,
	})

	assert.Nil(t, err)
	assert.False(t, hasRun)
	assert.Equal(t, util.ParseTimeOrFatal("1970-01-01T00:01:00+00:00"), scheduledTime)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())
}

func TestRunForSingleRowNoPreviousJobAndRuns(t *testing.T) {
	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTime(
		time.Unix(timeFarInTheFutureSec, 0)))
	defer store.Close()
	manager := resource.NewResourceManager(store)
	controller := NewController(manager)

	// Create package and pipeline.
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.PackageManager().CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		Schedule:  "* * * * * *",
		PackageId: 1}
	_, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())

	// Run job.
	hasRun, scheduledTime, err := controller.runForSingleRow(&storage.PipelineAndLatestJob{
		PipelineID:             1,
		PipelineName:           "PIPELINE_NAME",
		PipelineSchedule:       "* 1 * * * *",
		JobName:                nil,
		JobScheduledAtInSec:    nil,
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: 20,
	})

	assert.Nil(t, err)
	assert.True(t, hasRun)
	assert.Equal(t, util.ParseTimeOrFatal("1970-01-01T00:01:00+00:00"), scheduledTime)
	assert.Equal(t, 1, store.WorkflowClientFake().GetWorkflowCount())
}

func TestRunForSingleRowPreviousJobAndDoesNotRun(t *testing.T) {

	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTime(
		util.ParseTimeOrFatal("1970-10-01T01:00:00+00:00")))
	defer store.Close()

	controller := NewController(resource.NewResourceManager(store))

	hasRun, scheduledTime, err := controller.runForSingleRow(&storage.PipelineAndLatestJob{
		PipelineID:       100,
		PipelineName:     "PIPELINE_NAME",
		PipelineSchedule: "* 5 * * * *",
		JobName:          util.StringPointer("JOB_NAME"),
		JobScheduledAtInSec: util.Int64Pointer(
			util.ParseTimeOrFatal("1970-10-01T00:59:00+00:00").Unix()),
		PipelineEnabled:        true,
		PipelineEnabledAtInSec: util.ParseTimeOrFatal("1970-01-01T01:00:00+00:00").Unix(),
	})

	assert.Nil(t, err)
	assert.False(t, hasRun)
	assert.Equal(t, util.ParseTimeOrFatal("1970-10-01T01:05:00+00:00"), scheduledTime)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())
}

func TestRunForQueryNothingToDo(t *testing.T) {

	store := storage.NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()

	controller := NewController(resource.NewResourceManager(store))

	err := controller.runForQuery()
	assert.Nil(t, err)

}
