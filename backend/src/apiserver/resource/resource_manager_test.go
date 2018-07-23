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

func createPkg(name string) *model.Package {
	return &model.Package{Name: name, Status: model.PackageReady}
}

func TestCreatePackage(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pkg, err := manager.CreatePackage("package1", []byte(""))
	pkgExpected := &model.Package{
		ID:             1,
		CreatedAtInSec: 1,
		Name:           "package1",
		Parameters:     "[]",
		Status:         model.PackageReady,
	}
	assert.Nil(t, err)
	assert.Equal(t, pkgExpected, pkg)
}

func TestCreatePackage_GetParametersError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	_, err := manager.CreatePackage("package1", []byte("I am invalid yaml"))
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to parse the parameter")
}

func TestCreatePackage_StorePackageMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.DB().Close()
	manager := NewResourceManager(store)
	_, err := manager.CreatePackage("package1", []byte(""))
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to add package to package table")
}

func TestCreatePackage_CreatePackageFileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	_, err := manager.CreatePackage("package1", []byte(""))
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "bad object store")
	var pkg model.Package
	r := store.DB().First(&pkg, "1")
	assert.Nil(t, r.Error)
	assert.Equal(t, model.PackageCreating, pkg.Status)
}

func TestGetPackageTemplate(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pkg, _ := store.PackageStore().CreatePackage(createPkg("pkg1"))
	template := []byte("workflow: foo")
	store.ObjectStore().AddFile(template, storage.PackageFolder, fmt.Sprint(pkg.ID))
	manager := NewResourceManager(store)
	actualTemplate, err := manager.GetPackageTemplate(pkg.ID)
	assert.Nil(t, err)
	assert.Equal(t, template, actualTemplate)
}

func TestGetPackageTemplate_PackageMetadataNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	template := []byte("workflow: foo")
	store.ObjectStore().AddFile(template, storage.PackageFolder, fmt.Sprint(1))
	manager := NewResourceManager(store)
	_, err := manager.GetPackageTemplate(1)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Package 1 not found")
}

func TestGetPackageTemplate_PackageFileNotFound(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pkg, _ := store.PackageStore().CreatePackage(createPkg("pkg1"))
	manager := NewResourceManager(store)
	_, err := manager.GetPackageTemplate(pkg.ID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to get 1 from packages: object not found")
}

func TestListJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline1 := &model.PipelineDetail{
		Pipeline: model.Pipeline{
			UUID:           "1",
			Name:           "pp1",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 0,
			UpdatedAtInSec: 0,
		},
		ScheduledWorkflow: "scheduledworkflow1",
	}
	job1 := &model.JobDetail{
		Job: model.Job{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		Workflow: "workflow1",
	}
	store.DB().Create(pipeline1)
	store.DB().Create(job1)
	jobs, newToken, err := manager.ListJobs(pipeline1.UUID, "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	jobsExpected := []model.Job{{
		UUID:             "1",
		Name:             "job1",
		Namespace:        "n1",
		PipelineID:       "1",
		CreatedAtInSec:   1,
		ScheduledAtInSec: 1,
		Conditions:       "running",
	}}
	assert.Nil(t, err)
	assert.Equal(t, "", newToken)
	assert.Equal(t, jobsExpected, jobs)
}

func TestListJob_PipelineNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, _, err := manager.ListJobs("1", "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestGetJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline1 := &model.PipelineDetail{
		Pipeline: model.Pipeline{
			UUID:           "1",
			Name:           "pp1",
			Namespace:      "n1",
			PackageId:      1,
			Enabled:        true,
			CreatedAtInSec: 0,
			UpdatedAtInSec: 0,
		},
		ScheduledWorkflow: "scheduledworkflow1",
	}
	job1 := &model.JobDetail{
		Job: model.Job{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		Workflow: "workflow1",
	}
	store.DB().Create(pipeline1)
	store.DB().Create(job1)
	job, err := manager.GetJob(pipeline1.UUID, job1.UUID)
	jobExpected := &model.JobDetail{
		Job: model.Job{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Conditions:       "running",
		},
		Workflow: "workflow1",
	}
	assert.Nil(t, err)
	assert.Equal(t, jobExpected, job)
}

func TestGetJob_PipelineNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, err := manager.GetJob("1", "foo")
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestCreatePipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	expectedPipeline := &model.Pipeline{
		UUID:           "123",
		Name:           "pp1",
		Namespace:      "default",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		Conditions:     "NO_STATUS:",
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedPipeline, newPipeline)
}

func TestCreatePipeline_FailedToCreateScheduleWorkflow(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	_, err := manager.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to create a scheduled workflow")
}

func TestCreatePipeline_FailedToRetrieveYaml(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	_, err := manager.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "bad object store")
}

func TestEnablePipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)
	err = manager.EnablePipeline(newPipeline.UUID, false)
	assert.Nil(t, err)
	newPipeline, err = manager.GetPipeline(newPipeline.UUID)
	expectedPipeline := &model.Pipeline{
		UUID:           "123",
		Name:           "pp1",
		Namespace:      "default",
		PackageId:      1,
		Enabled:        false,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 2,
		Conditions:     "NO_STATUS:",
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedPipeline, newPipeline)
}

func TestEnablePipeline_PipelineNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.EnablePipeline("1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestEnablePipeline_CrdFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	err = manager.EnablePipeline(newPipeline.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check pipeline exist failed: some error")
}

func TestEnablePipeline_DbFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)
	store.DB().Close()
	err = manager.EnablePipeline(newPipeline.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeletePipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	err = manager.DeletePipeline(newPipeline.UUID)
	assert.Nil(t, err)

	pipeline, err = manager.GetPipeline(newPipeline.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 123 not found")
}

func TestDeletePipeline_PipelineNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeletePipeline("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestDeletePipeline_CrdFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	err = manager.DeletePipeline(newPipeline.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check pipeline exist failed: some error")
}

func TestDeletePipeline_DbFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	store.DB().Close()
	err = manager.DeletePipeline(newPipeline.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestReportWorkflowResource_Success(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	// Create package
	workflowForPackage := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPackage, storage.PackageFolder, "1")

	// Create pipeline
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
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
				UID:        types.UID(newPipeline.UUID),
			}},
			CreationTimestamp: v1.NewTime(time.Unix(11, 0).UTC()),
		},
	})
	err = manager.ReportWorkflowResource(workflow.ToStringForStore())
	assert.Nil(t, err)

	jobDetail, err := manager.GetJob(newPipeline.UUID, "MY_JOB_ID")
	assert.Nil(t, err)

	expectedJobDetail := &model.JobDetail{
		Job: model.Job{
			UUID:             "MY_JOB_ID",
			Name:             "MY_NAME",
			Namespace:        "MY_NAMESPACE",
			PipelineID:       "123",
			CreatedAtInSec:   11,
			ScheduledAtInSec: 0,
			Conditions:       ":",
		},
		Workflow: workflow.ToStringForStore(),
	}

	assert.Equal(t, expectedJobDetail, jobDetail)
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

	// Create package
	workflowForPackage := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPackage, storage.PackageFolder, "1")

	// Create pipeline
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
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
				UID:        types.UID(newPipeline.UUID),
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

	// Create package
	workflowForPackage := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPackage, storage.PackageFolder, "1")

	// Create pipeline
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(newPipeline.UUID),
		},
	})
	err = manager.ReportScheduledWorkflowResource(swf.ToStringForStore())
	assert.Nil(t, err)

	jobDetail, err := manager.GetPipeline(newPipeline.UUID)
	assert.Nil(t, err)

	expectedPipeline := &model.Pipeline{
		Name:       "MY_NAME",
		Namespace:  "MY_NAMESPACE",
		PackageId:  1,
		Enabled:    false,
		UUID:       newPipeline.UUID,
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
	assert.Equal(t, expectedPipeline, jobDetail)
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

	// Create package
	workflowForPackage := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflowForPackage, storage.PackageFolder, "1")

	// Create pipeline
	pipeline := &model.Pipeline{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	store.Close()

	// report scheduled workflow
	swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "MY_NAME",
			Namespace: "MY_NAMESPACE",
			UID:       types.UID(newPipeline.UUID),
		},
	})
	err = manager.ReportScheduledWorkflowResource(swf.ToStringForStore())
	assert.NotNil(t, err)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.(*util.UserError).String(), "database is closed")

}
