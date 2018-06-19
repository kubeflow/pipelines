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

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/googleprivate/ml/backend/src/model"
	"github.com/googleprivate/ml/backend/src/storage"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
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

func createPipeline(name string) *model.Pipeline {
	return &model.Pipeline{Name: name, PackageId: 1, Status: model.PipelineReady}
}

func TestCreatePipeline_NoSchedule(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: util.StringPointer("value1")},
					{Name: "param2", Value: util.StringPointer("value2")},
				},
			}}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")

	pipeline := &model.Pipeline{
		Name:       "MY_PIPELINE",
		PackageId:  1,
		Parameters: `[{"Name": "param1", "Value": "replacedvalue1"}]`}
	pipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	expected := model.Pipeline{
		ID:             1,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "",
		Enabled:        true,
		EnabledAtInSec: 2,
		Parameters:     `[{"Name": "param1", "Value": "replacedvalue1"}]`,
		Status:         model.PipelineReady}

	assert.Equalf(t, expected, *pipeline, "Unexpected pipeline structure. Expect %v. Got %v.",
		expected, *pipeline)
	assert.Equal(t, 1, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")

	expectedWorkflow := v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: util.StringPointer("replacedvalue1")},
					{Name: "param2", Value: util.StringPointer("value2")},
				},
			}}}
	var actualWorkflow v1alpha1.Workflow
	err = store.ObjectStore().GetFromYamlFile(&actualWorkflow, storage.PipelineFolder, "1")
	assert.Nil(t, err)
	assert.Equal(t, expectedWorkflow, actualWorkflow)
}

func TestCreatePipeline_FormatWorkflow(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	// Prepare store
	workflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{GenerateName: "workflow-name-"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: util.StringPointer("value1-[[schedule]]")},
					{Name: "param2", Value: util.StringPointer("value2-[[now]]-suffix")},
				},
			}}}
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")

	// Create pipeline
	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	pipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err)

	// Check pipeline
	expected := model.Pipeline{
		ID:             1,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "",
		Enabled:        true,
		EnabledAtInSec: 2,
		Status:         model.PipelineReady}
	assert.Equal(t, expected, *pipeline)

	// Check workflow
	assert.Equal(t, 1, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")

	expectedWorkflow := &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "workflow-name-0"},
		Spec: v1alpha1.WorkflowSpec{
			Arguments: v1alpha1.Arguments{
				Parameters: []v1alpha1.Parameter{
					{Name: "param1", Value: util.StringPointer("value1-19700101000003")},
					{Name: "param2", Value: util.StringPointer("value2-19700101000004-suffix")},
				},
			}}}

	jobDetail, err := store.JobStore().GetJob(1, "workflow-name-0")
	assert.Equal(t, expectedWorkflow, jobDetail.Workflow)
}

func TestCreatePipeline_ValidSchedule(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.ObjectStore().AddFile([]byte("kind: Workflow"), storage.PackageFolder, "1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "1 0 * * *"}
	pipeline, err := manager.CreatePipeline(pipeline)

	assert.Nil(t, err, "There should not be an error: %v", err)

	expected := model.Pipeline{
		ID:             1,
		CreatedAtInSec: 2,
		UpdatedAtInSec: 2,
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "1 0 * * *",
		Enabled:        true,
		EnabledAtInSec: 2,
		Status:         model.PipelineReady}

	assert.Equalf(t, expected, *pipeline, "Unexpected pipeline structure. Expect %v. Got %v.",
		expected, *pipeline)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipeline_InvalidSchedule(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.ObjectStore().AddFile([]byte("kind: Workflow"), storage.PackageFolder, "1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "abcdef"}
	pipeline, err := manager.CreatePipeline(pipeline)

	assert.Contains(t, err.Error(),
		"InvalidInputError: The pipeline schedule cannot be parsed: abcdef: Expected 5 to 6 fields")
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipeline_CreatePipelineFileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.Pipeline{Name: "MY_PIPELINE", PackageId: 1}

	_, err := manager.CreatePipeline(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "bad object store")
	r := store.DB().First(&pipeline, "1")
	assert.Nil(t, r.Error)
	assert.Equal(t, model.PipelineCreating, pipeline.Status)
}

func TestCreateJobFromPipelineID(t *testing.T) {
	// Use a real UUID in this test case to guarantee job with unique name
	store, err := NewFakeClientManager(util.NewFakeTimeForEpoch(), util.NewUUIDGenerator())
	assert.Nil(t, err)
	defer store.Close()
	manager := NewResourceManager(store)
	// Create pipeline with a schedule.
	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "* * * * * *",
		Status:    model.PipelineReady}
	pipeline, err = store.PipelineStore().CreatePipeline(pipeline)
	assert.Nil(t, err)
	assert.Equal(t, 0, store.WorkflowClientFake().GetWorkflowCount())
	err = store.ObjectStore().AddFile([]byte("kind: Workflow"), storage.PipelineFolder, "1")
	assert.Nil(t, err)

	// Create job.
	scheduledAtInSec := int64(5)
	jobDetail1, err := manager.CreateJobFromPipelineID(pipeline.ID, scheduledAtInSec)
	assert.Nil(t, err)
	expectedJob1 := &model.Job{
		Name:             jobDetail1.Workflow.Name,
		CreatedAtInSec:   2,
		UpdatedAtInSec:   2,
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 5,
		PipelineID:       1,
	}
	assert.Equal(t, map[string]bool{jobDetail1.Workflow.Name: true},
		store.WorkflowClientFake().GetWorkflowKeys())
	assert.Equal(t, expectedJob1, jobDetail1.Job)

	_, err = store.JobStore().GetJob(pipeline.ID, jobDetail1.Workflow.Name)
	assert.Nil(t, err)

	jobs, _, err := store.JobStore().ListJobs(pipeline.ID, "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Len(t, jobs, 1)

	// Create another job.
	scheduledAtInSec = int64(6)
	jobDetail2, err := manager.CreateJobFromPipelineID(pipeline.ID, scheduledAtInSec)
	assert.Nil(t, err)
	expectedJob2 := &model.Job{
		Name:             jobDetail2.Workflow.Name,
		CreatedAtInSec:   3,
		UpdatedAtInSec:   3,
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 6,
		PipelineID:       1,
	}
	assert.Equal(t, map[string]bool{jobDetail1.Workflow.Name: true, jobDetail2.Workflow.Name: true},
		store.WorkflowClientFake().GetWorkflowKeys())
	assert.Equal(t, expectedJob2, jobDetail2.Job)

	_, err = store.JobStore().GetJob(pipeline.ID, jobDetail2.Workflow.Name)
	assert.Nil(t, err)

	jobs, _, err = store.JobStore().ListJobs(pipeline.ID, "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	assert.Nil(t, err)
	assert.Len(t, jobs, 2)
}

func TestCreateJobFromPipelineID_GetPipelineError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// We do not create the pipeline!

	// Create job.
	scheduledAtInSec := int64(5)
	_, err := manager.CreateJobFromPipelineID(55, scheduledAtInSec)
	assert.Contains(t, err.Error(), "Could not get pipeline from pipeline ID")
}

func TestListJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.ObjectStore().AddFile([]byte("kind: Workflow"), storage.PackageFolder, "1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	pipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err, "There should not be an error: %v", err)
	jobs, newToken, err := manager.ListJobs(pipeline.ID, "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	jobsExpected := []model.Job{{
		CreatedAtInSec:   4,
		UpdatedAtInSec:   4,
		Name:             "workflow-0",
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 3,
		PipelineID:       1,
	}}

	assert.Nil(t, err)
	assert.Equal(t, "", newToken)
	assert.Equal(t, jobsExpected, jobs)
}

func TestListJob_PipelineNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, _, err := manager.ListJobs(1, "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestGetJob(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	store.PackageStore().CreatePackage(createPkg("pkg1"))
	store.ObjectStore().AddFile([]byte(""), storage.PackageFolder, "1")

	pipeline := &model.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1}
	pipeline, err := manager.CreatePipeline(pipeline)
	assert.Nil(t, err, "There should not be an error: %v", err)

	job, err := manager.GetJob(pipeline.ID, "workflow-0")
	jobExpected := &model.Job{
		CreatedAtInSec:   4,
		UpdatedAtInSec:   4,
		Name:             "workflow-0",
		Status:           model.JobExecutionPending,
		ScheduledAtInSec: 3,
		PipelineID:       1,
	}
	assert.Equal(t, jobExpected, job.Job)
	assert.Equal(t, "workflow-0", job.Workflow.Name)
}

func TestGetJob_PipelineNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, err := manager.GetJob(1, "foo")
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestDeletePackage(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pkg, _ := store.PackageStore().CreatePackage(createPkg("pkg1"))
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, fmt.Sprint(pkg.ID))

	err := manager.DeletePackage(pkg.ID)
	assert.Nil(t, err)
	_, err = store.PackageStore().GetPackage(pkg.ID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Package 1 not found")
	_, err = store.ObjectStore().GetFile(storage.PackageFolder, fmt.Sprint(pkg.ID))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "object not found")
}

func TestDeletePackage_PackageNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	err := manager.DeletePackage(1)
	assert.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Package 1 not found")
}

func TestDeletePackage_UpdatePackageMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pkg, _ := store.PackageStore().CreatePackage(createPkg("pkg1"))
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, fmt.Sprint(pkg.ID))

	store.DB().Close()
	err := manager.DeletePackage(pkg.ID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to get package")
}

func TestDeletePackage_DeletePackageFileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pkg, _ := store.PackageStore().CreatePackage(createPkg("pkg1"))
	// Not store the template yaml file.

	err := manager.DeletePackage(pkg.ID)
	assert.Nil(t, err)
	r := store.DB().First(&pkg, pkg.ID)
	assert.Nil(t, r.Error)
	assert.Equal(t, model.PackageDeleting, pkg.Status)
}

func TestDeletePipeline(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipeline("MY_PIPELINE"))
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, fmt.Sprint(pipeline.ID))

	err := manager.DeletePipeline(pipeline.ID)
	assert.Nil(t, err)
	_, err = store.PipelineStore().GetPipeline(pipeline.ID)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
	_, err = store.ObjectStore().GetFile(storage.PipelineFolder, fmt.Sprint(pipeline.ID))
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "object not found")
}

func TestDeletePipeline_PipelineNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	err := manager.DeletePipeline(1)
	assert.NotNil(t, err)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestDeletePipeline_pdatePipelineMetadataError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipeline("MY_PIPELINE"))
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PipelineFolder, fmt.Sprint(pipeline.ID))

	store.DB().Close() // Close DB early
	err := manager.DeletePipeline(pipeline.ID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to get pipeline")
}

func TestDeletePipeline_DeletePipelineFileError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline, _ := store.PipelineStore().CreatePipeline(createPipeline("MY_PIPELINE"))
	// Not store the workflow yaml file.

	err := manager.DeletePipeline(pipeline.ID)
	assert.Nil(t, err)
	r := store.DB().First(&pipeline, pipeline.ID)
	assert.Nil(t, r.Error)
	assert.Equal(t, model.PipelineDeleting, pipeline.Status)
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

func TestListJobV2(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline1 := &model.PipelineDetailV2{
		PipelineV2: model.PipelineV2{
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
	job1 := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Condition:        "running",
		},
		Workflow: "workflow1",
	}
	store.DB().Create(pipeline1)
	store.DB().Create(job1)
	jobs, newToken, err := manager.ListJobsV2(pipeline1.UUID, "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	jobsExpected := []model.JobV2{{
		UUID:             "1",
		Name:             "job1",
		Namespace:        "n1",
		PipelineID:       "1",
		CreatedAtInSec:   1,
		ScheduledAtInSec: 1,
		Condition:        "running",
	}}
	assert.Nil(t, err)
	assert.Equal(t, "", newToken)
	assert.Equal(t, jobsExpected, jobs)
}

func TestListJobV2_PipelineNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, _, err := manager.ListJobsV2("1", "" /*pageToken*/, 0 /*pageSize*/, "" /*sortByFieldName*/)
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestGetJobV2(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	pipeline1 := &model.PipelineDetailV2{
		PipelineV2: model.PipelineV2{
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
	job1 := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Condition:        "running",
		},
		Workflow: "workflow1",
	}
	store.DB().Create(pipeline1)
	store.DB().Create(job1)
	job, err := manager.GetJobV2(pipeline1.UUID, job1.UUID)
	jobExpected := &model.JobDetailV2{
		JobV2: model.JobV2{
			UUID:             "1",
			Name:             "job1",
			Namespace:        "n1",
			PipelineID:       "1",
			CreatedAtInSec:   1,
			ScheduledAtInSec: 1,
			Condition:        "running",
		},
		Workflow: "workflow1",
	}
	assert.Nil(t, err)
	assert.Equal(t, jobExpected, job)
}

func TestGetJobV2_PipelineNotFoundError(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	_, err := manager.GetJobV2("1", "foo")
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestCreatePipelineV2(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)

	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipelineV2(pipeline)
	expectedPipeline := &model.PipelineV2{
		UUID:           "123",
		Name:           "pp1",
		Namespace:      "default",
		PackageId:      1,
		Enabled:        true,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 1,
		Condition:      "NO_STATUS",
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedPipeline, newPipeline)
}

func TestCreatePipelineV2_FailedToCreateScheduleWorkflow(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	_, err := manager.CreatePipelineV2(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Failed to create a scheduled workflow")
}

func TestCreatePipelineV2_FailedToRetrieveYaml(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	// Use a bad object store
	manager.objectStore = &FakeBadObjectStore{}
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	_, err := manager.CreatePipelineV2(pipeline)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "bad object store")
}

func TestEnablePipelineV2(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipelineV2(pipeline)
	assert.Nil(t, err)
	err = manager.EnablePipelineV2(newPipeline.UUID, false)
	assert.Nil(t, err)
	newPipeline, err = manager.GetPipelineV2(newPipeline.UUID)
	expectedPipeline := &model.PipelineV2{
		UUID:           "123",
		Name:           "pp1",
		Namespace:      "default",
		PackageId:      1,
		Enabled:        false,
		CreatedAtInSec: 1,
		UpdatedAtInSec: 2,
		Condition:      "NO_STATUS",
	}
	assert.Nil(t, err)
	assert.Equal(t, expectedPipeline, newPipeline)
}

func TestEnablePipelineV2_PipelineNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.EnablePipelineV2("1", false)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestEnablePipelineV2_CrdFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipelineV2(pipeline)
	assert.Nil(t, err)

	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	err = manager.EnablePipelineV2(newPipeline.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check pipeline exist failed: some error")
}

func TestEnablePipelineV2_DbFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipelineV2(pipeline)
	assert.Nil(t, err)
	store.DB().Close()
	err = manager.EnablePipelineV2(newPipeline.UUID, false)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}

func TestDeletePipelineV2(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipelineV2(pipeline)
	assert.Nil(t, err)

	err = manager.DeletePipelineV2(newPipeline.UUID)
	assert.Nil(t, err)

	pipeline, err = manager.GetPipelineV2(newPipeline.UUID)
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 123 not found")
}

func TestDeletePipelineV2_PipelineNotExist(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	err := manager.DeletePipelineV2("1")
	assert.Equal(t, codes.NotFound, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Pipeline 1 not found")
}

func TestDeletePipelineV2_CrdFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipelineV2(pipeline)
	assert.Nil(t, err)

	manager.scheduledWorkflow = &FakeBadScheduledWorkflowClient{}
	err = manager.DeletePipelineV2(newPipeline.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "Check pipeline exist failed: some error")
}

func TestDeletePipelineV2_DbFailure(t *testing.T) {
	store := NewFakeClientManagerOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	manager := NewResourceManager(store)
	workflow := &v1alpha1.Workflow{ObjectMeta: v1.ObjectMeta{Name: "workflow-name"}}
	store.ObjectStore().AddAsYamlFile(workflow, storage.PackageFolder, "1")
	pipeline := &model.PipelineV2{
		Name:      "pp1",
		PackageId: 1,
		Enabled:   true,
	}
	newPipeline, err := manager.CreatePipelineV2(pipeline)
	assert.Nil(t, err)

	store.DB().Close()
	err = manager.DeletePipelineV2(newPipeline.UUID)
	assert.Equal(t, codes.Internal, err.(*util.UserError).ExternalStatusCode())
	assert.Contains(t, err.Error(), "database is closed")
}
