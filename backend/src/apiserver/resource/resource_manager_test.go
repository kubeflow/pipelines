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
	"strings"
	"testing"
	"time"

	"encoding/json"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/src/apiserver/common"
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
	pipeline, err := manager.CreatePipeline("pipeline1", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
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

func TestCreatePipeline_ComplexPipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	createdPipeline, err := manager.CreatePipeline("pipeline1", []byte(strings.TrimSpace(
		complexPipeline)))
	assert.Nil(t, err)
	_, err = manager.GetPipeline(createdPipeline.UUID)
	assert.Nil(t, err)
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
	_, err := manager.CreatePipeline("pipeline1", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to add pipeline to pipeline table")
}

func TestCreatePipeline_CreatePipelineFileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	_, err := manager.CreatePipeline("pipeline1", []byte("apiVersion: argoproj.io/v1alpha1\nkind: Workflow"))
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
	store.ObjectStore().AddFile(template, storage.PipelineFolder, fmt.Sprint(pipeline.UUID))
	manager := NewResourceManager(store)
	actualTemplate, err := manager.GetPipelineTemplate(pipeline.UUID)
	assert.Nil(t, err)
	assert.Equal(t, template, actualTemplate)
}

func TestGetPipelineTemplate_PipelineMetadataNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	template := []byte("workflow: foo")
	store.ObjectStore().AddFile(template, storage.PipelineFolder, fmt.Sprint(1))
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
	runs, newToken, err := manager.ListRuns("1",
		&common.PaginationContext{
			PageSize:        1,
			KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
			SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
			IsDesc:          false,
		})
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

	_, _, err := manager.ListRuns("1", &common.PaginationContext{
		PageSize:        0,
		KeyFieldName:    model.GetRunTablePrimaryKeyColumn(),
		SortByFieldName: model.GetRunTablePrimaryKeyColumn(),
		IsDesc:          false,
	})
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

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1"},
				},
			},
		}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		DisplayName: "pp 1",
		PipelineId:  "1",
		Enabled:     true,
		Parameters:  "[{\"name\":\"param1\",\"value\":\"world\"}]",
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
		Parameters:     "[{\"name\":\"param1\",\"value\":\"world\"}]",
	}

	assert.Nil(t, err)
	assert.Equal(t, expectedJob, newJob)
}

func TestCreateJob_ExtraInputParameter(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1"},
				},
			},
		}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		DisplayName: "pp 1",
		PipelineId:  "1",
		Enabled:     true,
		Parameters:  "[{\"name\":\"param2\",\"value\":\"world\"}]",
	}
	_, err := manager.CreateJob(job)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Unrecognized input parameter: param2")
}

func TestCreateJob_InvalidParameterFormat(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1"},
				},
			},
		}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		DisplayName: "pp 1",
		PipelineId:  "1",
		Enabled:     true,
		Parameters:  "[I am invalid parameter format]",
	}
	_, err := manager.CreateJob(job)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to parse the parameter CRD")
}

func TestCreateJob_FailedToCreateScheduleWorkflow(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		DisplayName: "pp 1",
		PipelineId:  "1",
		Enabled:     true,
		Parameters:  "[]",
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
		Parameters:     "[]",
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
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, "1")
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.PipelineFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.PipelineFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.PipelineFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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
	store.ObjectStore().AddAsYamlFile(workflowForPipeline, storage.PipelineFolder, "1")

	// Create job
	job := &model.Job{
		Name:       "pp1",
		PipelineId: "1",
		Enabled:    true,
		Parameters: "[]",
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

const (
	complexPipeline = `
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: tfmataxicabclassificationpipelineexample-
spec:
  arguments:
    parameters:
    - name: output
    - name: project
    - name: schema
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/schema.json
    - name: train
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/train.csv
    - name: evaluation
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/eval.csv
    - name: preprocess-mode
      value: local
    - name: preprocess-module
      value: gs://ml-pipeline-playground/tfma/taxi-cab-classification/preprocessing.py
    - name: target
      value: tips
    - name: learning-rate
      value: '0.1'
    - name: hidden-layer-size
      value: '1500'
    - name: steps
      value: '3000'
    - name: workers
      value: '0'
    - name: pss
      value: '0'
    - name: predict-mode
      value: local
    - name: analyze-mode
      value: local
    - name: analyze-slice-column
      value: trip_start_hour
  entrypoint: tfmataxicabclassificationpipelineexample
  templates:
  - container:
      args:
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/analysis'
      - --model
      - '{{inputs.parameters.training-train}}'
      - --eval
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --project
      - '{{inputs.parameters.project}}'
      - --mode
      - '{{inputs.parameters.analyze-mode}}'
      - --slice-columns
      - '{{inputs.parameters.analyze-slice-column}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tfma
    inputs:
      parameters:
      - name: analyze-mode
      - name: analyze-slice-column
      - name: evaluation
      - name: output
      - name: project
      - name: schema
      - name: training-train
    name: analysis
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.default:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: analysis-analysis
        valueFrom:
          path: /output.txt
  - container:
      args:
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/predict'
      - --data
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --target
      - '{{inputs.parameters.target}}'
      - --model
      - '{{inputs.parameters.training-train}}'
      - --mode
      - '{{inputs.parameters.predict-mode}}'
      - --project
      - '{{inputs.parameters.project}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tf-predict
    inputs:
      parameters:
      - name: evaluation
      - name: output
      - name: predict-mode
      - name: project
      - name: schema
      - name: target
      - name: training-train
    name: prediction
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.default:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: prediction-predict
        valueFrom:
          path: /output.txt
  - container:
      args:
      - --train
      - '{{inputs.parameters.train}}'
      - --eval
      - '{{inputs.parameters.evaluation}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --output
      - '{{inputs.parameters.output}}/{{workflow.name}}/transformed'
      - --project
      - '{{inputs.parameters.project}}'
      - --mode
      - '{{inputs.parameters.preprocess-mode}}'
      - --preprocessing-module
      - '{{inputs.parameters.preprocess-module}}'
      image: gcr.io/ml-pipeline/ml-pipeline-dataflow-tft
    inputs:
      parameters:
      - name: evaluation
      - name: output
      - name: preprocess-mode
      - name: preprocess-module
      - name: project
      - name: schema
      - name: train
    name: preprocess
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.default:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: preprocess-transformed
        valueFrom:
          path: /output.txt
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: analyze-mode
            value: '{{inputs.parameters.analyze-mode}}'
          - name: analyze-slice-column
            value: '{{inputs.parameters.analyze-slice-column}}'
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: training-train
            value: '{{tasks.training.outputs.parameters.training-train}}'
        dependencies:
        - training
        name: analysis
        template: analysis
      - arguments:
          parameters:
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: predict-mode
            value: '{{inputs.parameters.predict-mode}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: target
            value: '{{inputs.parameters.target}}'
          - name: training-train
            value: '{{tasks.training.outputs.parameters.training-train}}'
        dependencies:
        - training
        name: prediction
        template: prediction
      - arguments:
          parameters:
          - name: evaluation
            value: '{{inputs.parameters.evaluation}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: preprocess-mode
            value: '{{inputs.parameters.preprocess-mode}}'
          - name: preprocess-module
            value: '{{inputs.parameters.preprocess-module}}'
          - name: project
            value: '{{inputs.parameters.project}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: train
            value: '{{inputs.parameters.train}}'
        name: preprocess
        template: preprocess
      - arguments:
          parameters:
          - name: hidden-layer-size
            value: '{{inputs.parameters.hidden-layer-size}}'
          - name: learning-rate
            value: '{{inputs.parameters.learning-rate}}'
          - name: output
            value: '{{inputs.parameters.output}}'
          - name: preprocess-module
            value: '{{inputs.parameters.preprocess-module}}'
          - name: preprocess-transformed
            value: '{{tasks.preprocess.outputs.parameters.preprocess-transformed}}'
          - name: pss
            value: '{{inputs.parameters.pss}}'
          - name: schema
            value: '{{inputs.parameters.schema}}'
          - name: steps
            value: '{{inputs.parameters.steps}}'
          - name: target
            value: '{{inputs.parameters.target}}'
          - name: workers
            value: '{{inputs.parameters.workers}}'
        dependencies:
        - preprocess
        name: training
        template: training
    inputs:
      parameters:
      - name: analyze-mode
      - name: analyze-slice-column
      - name: evaluation
      - name: hidden-layer-size
      - name: learning-rate
      - name: output
      - name: predict-mode
      - name: preprocess-mode
      - name: preprocess-module
      - name: project
      - name: pss
      - name: schema
      - name: steps
      - name: target
      - name: train
      - name: workers
    name: tfmataxicabclassificationpipelineexample
  - container:
      args:
      - --job-dir
      - '{{inputs.parameters.output}}/{{workflow.name}}/train'
      - --transformed-data-dir
      - '{{inputs.parameters.preprocess-transformed}}'
      - --schema
      - '{{inputs.parameters.schema}}'
      - --learning-rate
      - '{{inputs.parameters.learning-rate}}'
      - --hidden-layer-size
      - '{{inputs.parameters.hidden-layer-size}}'
      - --steps
      - '{{inputs.parameters.steps}}'
      - --target
      - '{{inputs.parameters.target}}'
      - --workers
      - '{{inputs.parameters.workers}}'
      - --pss
      - '{{inputs.parameters.pss}}'
      - --preprocessing-module
      - '{{inputs.parameters.preprocess-module}}'
      - --tfjob-timeout-minutes
      - '60'
      image: gcr.io/ml-pipeline/ml-pipeline-kubeflow-tf
    inputs:
      parameters:
      - name: hidden-layer-size
      - name: learning-rate
      - name: output
      - name: preprocess-module
      - name: preprocess-transformed
      - name: pss
      - name: schema
      - name: steps
      - name: target
      - name: workers
    name: training
    outputs:
      artifacts:
      - name: mlpipeline-ui-metadata
        path: /mlpipeline-ui-metadata.json
        s3:
          accessKeySecret:
            key: accesskey
            name: mlpipeline-minio-artifact
          bucket: mlpipeline
          endpoint: minio-service.default:9000
          insecure: true
          key: runs/{{workflow.uid}}/{{pod.name}}/mlpipeline-ui-metadata.tgz
          secretKeySecret:
            key: secretkey
            name: mlpipeline-minio-artifact
      parameters:
      - name: training-train
        valueFrom:
          path: /output.txt`
)
