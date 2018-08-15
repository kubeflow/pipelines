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

package resource

import (
	"fmt"
	"testing"
	"time"

	"encoding/json"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/apiserver/storage"
	"github.com/googleprivate/ml/backend/src/common/util"
	swfapi "github.com/googleprivate/ml/backend/src/crd/pkg/apis/scheduledworkflow/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type FakeBadObjectStore struct{}

func (m *FakeBadObjectStore) AddFile(template []byte, bucket string, fileName string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) DeleteFile(bucket string, fileName string) error {
	return errors.New("Not implemented.")
}

func (m *FakeBadObjectStore) GetFile(bucket string, fileName string) ([]byte, error) {
	return []byte(""), nil
}

func (m *FakeBadObjectStore) AddAsYamlFile(o interface{}, bucket string, fileName string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func (m *FakeBadObjectStore) GetFromYamlFile(o interface{}, bucket string, fileName string) error {
	return util.NewInternalServerError(errors.New("Error"), "bad object store")
}

func createPipeline(name string) *model.Pipeline {
	return &model.Pipeline{Name: name, Status: model.PipelineReady}
}

func TestCreatePipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline, err := manager.CreatePipeline("pipeline1", []byte(""))
	pipelineExpected := &model.Pipeline{
		UUID:           DefaultFakeUUID,
		CreatedAtInSec: 1,
		Name:           "pipeline1",
		Parameters:     "[]",
		Status:         model.PipelineReady,
	}
	assert.Nil(t, err)
	assert.Equal(t, pipelineExpected, pipeline)
}

func TestCreatePipeline_GetParametersError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	_, err := manager.CreatePipeline("pipeline1", []byte("I am invalid yaml"))
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to parse the parameter")
}

func TestCreatePipeline_StorePipelineMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	manager := NewResourceManager(store)
	_, err := manager.CreatePipeline("pipeline1", []byte(""))
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to add pipeline to pipeline table")
}

func TestCreatePipeline_CreatePipelineFileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	_, err := manager.CreatePipeline("pipeline1", []byte(""))
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "bad object store")
	// Verify there is a pipeline in DB with status PipelineCreating.
	pipeline, err := manager.pipelineStore.GetPipelineWithStatus(DefaultFakeUUID, model.PipelineCreating)
	assert.Nil(t, err)
	assert.NotNil(t, pipeline)
}

func TestGetPipelineTemplate(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipeline("pipeline1"))
	template := []byte("workflow: foo")
	store.ObjectStore().AddFile(template, storage.JobFolder, fmt.Sprint(pipeline.UUID))
	manager := NewResourceManager(store)
	actualTemplate, err := manager.GetPipelineTemplate(pipeline.UUID)
	assert.Nil(t, err)
	assert.Equal(t, template, actualTemplate)
}

func TestGetPipelineTemplate_PipelineMetadataNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	template := []byte("workflow: foo")
	store.ObjectStore().AddFile(template, storage.JobFolder, fmt.Sprint(1))
	manager := NewResourceManager(store)
	_, err := manager.GetPipelineTemplate("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestGetPipelineTemplate_PipelineFileNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipeline("pipeline1"))
	manager := NewResourceManager(store)
	_, err := manager.GetPipelineTemplate(pipeline.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "object not found")
}

func TestListRun(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	job1 := &model.Job{
		UUID:           "1",
		Name:           "pp1",
		Namespace:      "n1",
		PipelineId:     "1",
		Enabled:        true,
		CreatedAtInSec: 0,
		UpdatedAtInSec: 0,
	}

	_, err := manager.jobStore.CreateJob(job1)
	assert.Nil(t, err)
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "run1",
			Namespace:         "n1",
			UID:               "1",
			CreationTimestamp: v1.Time{Time: time.Unix(1, 0)},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})
	err = manager.runStore.CreateOrUpdateRun(workflow)
	assert.Nil(t, err)
	runs, newToken, err := manager.ListRuns("1", "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/, false /*isDesc*/)
	runsExpected := []model.Run{{
		UUID:           "1",
		Name:           "run1",
		Namespace:      "n1",
		JobID:          "1",
		CreatedAtInSec: 1,
		Conditions:     ":",
	}}
	assert.Nil(t, err)
	assert.Equal(t, "", newToken)
	assert.Equal(t, runsExpected, runs)
}

func TestListRun_JobNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, _, err := manager.ListRuns("1", "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/, false /*isDesc*/)
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestGetRun(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	job1 := &model.Job{
		UUID:           "1",
		Name:           "pp1",
		Namespace:      "n1",
		PipelineId:     "1",
		Enabled:        true,
		CreatedAtInSec: 0,
		UpdatedAtInSec: 0,
	}
	_, err := manager.jobStore.CreateJob(job1)
	assert.Nil(t, err)
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:              "run1",
			Namespace:         "n1",
			UID:               "1",
			CreationTimestamp: v1.Time{Time: time.Unix(1, 0)},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID("1"),
			}},
		},
	})
	err = manager.runStore.CreateOrUpdateRun(workflow)
	run, err := manager.GetRun("1")
	workflowBytes, _ := json.Marshal(workflow)
	runExpected := &model.RunDetail{
		Run: model.Run{
			UUID:           "1",
			Name:           "run1",
			Namespace:      "n1",
			JobID:          "1",
			CreatedAtInSec: 1,
			Conditions:     ":",
		},
		Workflow: string(workflowBytes),
	}
	assert.Nil(t, err)
	assert.Equal(t, runExpected, run)
}

func TestCreateJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		DisplayName: "pp 1",
		PipelineId:  "1",
		Enabled:     true,
	}
	newJob, err := manager.CreateJob(job)
	expectedJob := &model.Job{
		UUID:           "123",
		DisplayName:    "pp 1",
		Name:           "job-pp1",
		Namespace:      "default",
		PipelineId:     "1",
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		Conditions:     "NO_STATUS:",
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, newJob)
}

func TestCreateJob_FailedToCreateScheduleWorkflow(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	_, err := manager.CreateJob(job)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to create a scheduled workflow")
}

func TestCreateJob_FailedToRetrieveYaml(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	_, err := manager.CreateJob(job)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "bad object store")
}

func TestEnableJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		DisplayName: "pp 1",
		PipelineId:  "1",
		Enabled:     true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)
	err = manager.EnableJob(newJob.UUID, false)
	assert.Nil(t, err)
	newJob, err = manager.GetJob(newJob.UUID)
	expectedJob := &model.Job{
		UUID:           "123",
		DisplayName:    "pp 1",
		Name:           "job-pp1",
		Namespace:      "default",
		PipelineId:     "1",
		Enabled:        false,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 2,
		Conditions:     "NO_STATUS:",
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedJob, newJob)
}

func TestEnableJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.EnableJob("1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestEnableJob_CrdFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	err = manager.EnableJob(newJob.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed: some error")
}

func TestEnableJob_DbFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)
	store.DB().Close()
	err = manager.EnableJob(newJob.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeleteJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	err = manager.DeleteJob(newJob.UUID)
	assert.Nil(t, err)

	job, err = manager.GetJob(newJob.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 123 not found")
}

func TestDeleteJob_JobNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeleteJob("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Job 1 not found")
}

func TestDeleteJob_CrdFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	err = manager.DeleteJob(newJob.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check job exist failed: some error")
}

func TestDeleteJob_DbFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.JobFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	store.DB().Close()
	err = manager.DeleteJob(newJob.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestReportWorkflowResource_Success(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create pipeline
	workflowForPipeline := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.JobFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "MY_JOB_ID",
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID(newJob.UUID),
			}},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
		},
	})
	err = manager.ReportWorkflowResource(workflow.ToStringForStore())
	assert.Nil(t, err)

	runDetail, err := manager.GetRun("MY_JOB_ID")
	assert.Nil(t, err)

	expectedRunDetail := &model.RunDetail{
		Run: model.Run{
			UUID:             "MY_JOB_ID",
			Name:             "MY_NAME",
			Namespace:        "MY_NAMESPACE",
			JobID:            "123",
			CreatedAtInSec:   11,
			ScheduledAtInSec: 0,
			Conditions:       ":",
		},
		Workflow: workflow.ToStringForStore(),
	}

	assert.Equal(t, expectedRunDetail, runDetail)
}

func TestReportWorkflowResource_UnmarshalError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	err := manager.ReportWorkflowResource("WRONG_WORKFLOW")
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestReportWorkflowResource_Error(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create pipeline
	workflowForPipeline := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.JobFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	store.Close()

	// report workflow
	workflow := util.NewWorkflow(&v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       "MY_JOB_ID",
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "kubeflow.org/v1alpha1",
				Kind:       "ScheduledWorkflow",
				Name:       "SCHEDULE_NAME",
				UID:        types.UID(newJob.UUID),
			}},
		},
	})
	err = manager.ReportWorkflowResource(workflow.ToStringForStore())
	assert.NotNil(t, err)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.(*util.UserError).String(), "database is closed")
}

func TestReportScheduledWorkflowResource_Success(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create pipeline
	workflowForPipeline := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.JobFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(newJob.UUID),
		},
	})
	err = manager.ReportScheduledWorkflowResource(swf.ToStringForStore())
	assert.Nil(t, err)

	runDetail, err := manager.GetJob(newJob.UUID)
	assert.Nil(t, err)

	expectedJob := &model.Job{
		Name:       "MY_NAME",
		Namespace:  "MY_NAMESPACE",
		PipelineId: "1",
		Enabled:    false,
		UUID:       newJob.UUID,
		Conditions: "NO_STATUS:",
		Trigger: model.Trigger{
			CronSchedule: model.CronSchedule{
				Cron: util.StringPointer(""),
			},
			PeriodicSchedule: model.PeriodicSchedule{
				IntervalSecond: util.Int64Pointer(0),
			},
		},
		Parameters:     "[]",
		CreatedAtInSec: 1,
		UpdatedAtInSec: 2,
	}
	assert.Equal(t, expectedJob, runDetail)
}

func TestReportScheduledWorkflowResource_UnmarshalError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	err := manager.ReportScheduledWorkflowResource("WRONG_SCHEDULED_WORKFLOW")
	assert.NotNil(t, err)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Could not unmarshal")
}

func TestReportScheduledWorkflowResource_Error(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create pipeline
	workflowForPipeline := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.JobFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
	}
	newJob, err := manager.CreateJob(job)
	assert.Nil(t, err)

	store.Close()

	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(newJob.UUID),
		},
	})
	err = manager.ReportScheduledWorkflowResource(swf.ToStringForStore())
	assert.NotNil(t, err)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.(*util.UserError).String(), "database is closed")
}

func TestToScheduledWorkflowName_SpecialCharsAndSpace(t *testing.T) {
	assert.Equal(t, toScheduledWorkflowName("! HaVe ä £unky name"), "job-haveunkyname")
}

func TestToScheduledWorkflowName_TruncateLongName(t *testing.T) {
	assert.Equal(t, toScheduledWorkflowName("AloooooooooooooooooongName"), "job-aloooooooooooooooooon")
}
