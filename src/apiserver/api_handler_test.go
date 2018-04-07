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

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"ml/src/message"
	"ml/src/storage"
	"ml/src/util"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultScheduledAtInSec = 5
)

func createPkg(name string) *message.Package {
	return &message.Package{Name: name}
}

func createFakePipeline(name string, pkgId uint) *message.Pipeline {
	return &message.Pipeline{Name: name, PackageId: pkgId}
}

func createPipelineExpected1() message.Pipeline {
	return message.Pipeline{Metadata: &message.Metadata{ID: 1},
		Name:           "pipeline1",
		PackageId:      1,
		Enabled:        true,
		EnabledAtInSec: 1}
}

func createWorkflow(name string) *v1alpha1.Workflow {
	return &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: name},
		Status:     v1alpha1.WorkflowStatus{Phase: "Pending"}}
}

func createPipelineByte1() []byte {
	b, _ := json.Marshal(createFakePipeline("pipeline1", 1))
	return b
}

type FakeBadPackageStore struct{}

func (s *FakeBadPackageStore) ListPackages() ([]message.Package, error) {
	return nil, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) GetPackage(packageId uint) (*message.Package, error) {
	return nil, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) CreatePackage(*message.Package) error {
	return util.NewInternalError("bad package store", "")
}

type FakeBadJobStore struct{}

func (s *FakeBadJobStore) GetJob(pipelineId uint, name string) (*message.JobDetail, error) {
	return nil, util.NewInternalError("bad job store", "")
}

func (s *FakeBadJobStore) ListJobs(pipelineId uint) ([]message.Job, error) {
	return nil, util.NewInternalError("bad job store", "")
}

func (s *FakeBadJobStore) CreateJob(pipelineId uint, workflow *v1alpha1.Workflow,
	scheduledAtInSec int64) (*message.JobDetail, error) {
	return nil, util.NewInternalError("bad job store", "")
}

type FakeBadPackageManager struct{}

func (m *FakeBadPackageManager) CreatePackageFile(template []byte, fileName string) error {
	return util.NewInternalError("bad package manager", "")
}

func (m *FakeBadPackageManager) GetTemplate(fileName string) ([]byte, error) {
	return nil, util.NewInternalError("bad package manager", "")
}

type FakeBadPipelineStore struct{}

func (s *FakeBadPipelineStore) ListPipelines() ([]message.Pipeline, error) {
	return nil, util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) GetPipeline(id uint) (*message.Pipeline, error) {
	return nil, util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) CreatePipeline(*message.Pipeline) error {
	return util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) GetPipelineAndLatestJobIterator() (
	*storage.PipelineAndLatestJobIterator, error) {
	return nil, errors.New("Not implemented.")
}

func (s *FakeBadPipelineStore) EnablePipeline(id uint, enabled bool) error {
	return errors.New("Not implemented.")
}

func initApiHandlerTest(
	ps storage.PackageStoreInterface,
	js storage.JobStoreInterface,
	pls storage.PipelineStoreInterface,
	pm storage.PackageManagerInterface) *iris.Application {
	clientManager := ClientManager{
		packageStore:   ps,
		jobStore:       js,
		pipelineStore:  pls,
		packageManager: pm,
		time:           util.NewFakeTimeForEpoch()}
	return newApp(clientManager)
}

func TestListPackages(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	store.PackageStore.CreatePackage(createPkg("pkg2"))
	expectedPkg1 := createPkg("pkg1")
	expectedPkg1.Metadata = &message.Metadata{ID: 1}
	expectedPkg2 := createPkg("pkg2")
	expectedPkg2.Metadata = &message.Metadata{ID: 2}
	expectedResponse := []message.Package{*expectedPkg1, *expectedPkg2}
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, nil))

	response := e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusOK)
	var actual []message.Package
	util.MarshalOrFail(response.Body().Raw(), &actual)
	assert.Equal(t, expectedResponse, actual)
}

func TestListPackagesError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, nil))
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusInternalServerError)
}

func TestGetPackage(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, nil))
	expectedPkg1 := createPkg("pkg1")
	expectedPkg1.Metadata = &message.Metadata{ID: 1}

	response := e.GET("/apis/v1alpha1/packages/1").Expect().Status(httptest.StatusOK)
	var actual message.Package
	util.MarshalOrFail(response.Body().Raw(), &actual)
	assert.Equal(t, *expectedPkg1, actual)
}

func TestGetPackage_InternalError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, nil))
	e.GET("/apis/v1alpha1/packages/1").Expect().Status(httptest.StatusInternalServerError)
}

func TestGetPackage_NotFound(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, nil))

	e.GET("/apis/v1alpha1/packages/1").Expect().Status(httptest.StatusBadRequest).
		Body().Contains("Package 1 not found")
}

func TestUploadPackage(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pm := storage.NewFakePackageManager()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world")
	w.Close()
	pkgsExpect := []message.Package{
		{Metadata: &message.Metadata{ID: 1}, Name: "hello-world", Parameters: []message.Parameter{}}}
	// Check response
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, pm))
	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusOK).Body().Contains("\"name\":\"hello-world\"")

	// Verify stored in package manager
	template, err := pm.GetTemplate("hello-world")
	assert.Nil(t, err)
	assert.NotNil(t, template)

	// Verify metadata in db
	pkg, err := store.PackageStore.ListPackages()
	assert.Nil(t, err)
	assert.Equal(t, pkg, pkgsExpect)
}

func TestUploadPackage_CreatePackageFileError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, &FakeBadPackageManager{}))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()

	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad package manager")
}

func TestUploadPackage_GetFormFileError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, storage.NewFakePackageManager()))
	var b bytes.Buffer
	b.WriteString("I am invalid file")
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()

	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusBadRequest).Body().Equal("Failed to read package.")
}

func TestUploadPackage_CreatePackageError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, storage.NewFakePackageManager()))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()

	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad package store")
}

func TestUploadPackage_GetParametersError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, storage.NewFakePackageManager()))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, _ := w.CreateFormFile("uploadfile", "hello-world.yaml")
	part.Write([]byte("I am invalid yaml"))
	w.Close()

	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusBadRequest).Body().Equal("Failed to parse the parameter.")
}

func TestGetTemplate(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	pm := storage.NewFakePackageManager()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	pm.CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, pm))

	e.GET("/apis/v1alpha1/packages/1/templates").Expect().Status(httptest.StatusOK).
		Body().Equal("kind: Workflow")
}

func TestGetTemplate_GetPackageError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, storage.NewFakePackageManager()))
	e.GET("/apis/v1alpha1/packages/1/templates").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad package store")
}

func TestGetTemplate_GetPackageFileError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, nil, nil, &FakeBadPackageManager{}))

	e.GET("/apis/v1alpha1/packages/1/templates").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad package manager")
}

func TestListPipelines(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore.CreatePipeline(createFakePipeline("pipeline1", 1))
	store.PipelineStore.CreatePipeline(createFakePipeline("pipeline2", 2))
	e := httptest.New(t, initApiHandlerTest(nil, nil, store.PipelineStore, nil))

	response := e.GET("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusOK)
	expectedResponse := []message.Pipeline{
		createPipelineExpected1(),
		{Metadata: &message.Metadata{ID: 2},
			Name:           "pipeline2",
			PackageId:      2,
			Enabled:        true,
			EnabledAtInSec: 2,
			Parameters:     []message.Parameter(nil)}}
	var actual []message.Pipeline
	util.MarshalOrFail(response.Body().Raw(), &actual)
	assert.Equal(t, expectedResponse, actual)
}

func TestListPipelinesError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakeBadPipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad pipeline store")
}

func TestGetPipeline(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PipelineStore.CreatePipeline(createFakePipeline("pipeline1", 1))
	e := httptest.New(t, initApiHandlerTest(nil, nil, store.PipelineStore, nil))

	response := e.GET("/apis/v1alpha1/pipelines/1").Expect().Status(httptest.StatusOK)
	var actual message.Pipeline
	util.MarshalOrFail(response.Body().Raw(), &actual)
	assert.Equal(t, createPipelineExpected1(), actual)
}

func TestGetPipeline_InternalError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakeBadPipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines/1").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad pipeline store")
}

func TestGetPipeline_NotFound(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(nil, nil, store.PipelineStore, nil))

	e.GET("/apis/v1alpha1/pipelines/1").Expect().Status(httptest.StatusBadRequest).
		Body().Contains("Pipeline 1 not found")
}

func TestCreatePipeline_NoSchedule(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	pm := storage.NewFakePackageManager()
	pm.CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore, pm))

	response := e.POST("/apis/v1alpha1/pipelines").WithBytes(createPipelineByte1()).Expect().Status(httptest.StatusOK)
	var actual message.Pipeline
	util.MarshalOrFail(response.Body().Raw(), &actual)
	assert.Equal(t, createPipelineExpected1(), actual)
	assert.Equal(t, 1, store.WorkflowClientFake.GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipeline_BadPipelineFormatError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, store.JobStore, store.PipelineStore, storage.NewFakePackageManager()))

	e.POST("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusBadRequest).
		Body().Contains("The pipeline has invalid format.")
}

func TestCreatePipeline_PackageNotExistError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, store.JobStore, store.PipelineStore, storage.NewFakePackageManager()))

	e.POST("/apis/v1alpha1/pipelines").WithBytes(createPipelineByte1()).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad package store")
}

func TestCreatePipeline_MetadataError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	pm := storage.NewFakePackageManager()
	pm.CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, &FakeBadPipelineStore{}, pm))

	e.POST("/apis/v1alpha1/pipelines").WithBytes(createPipelineByte1()).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad pipeline store")
}

func TestCreatePipeline_GetTemplateError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore, &FakeBadPackageManager{}))

	e.POST("/apis/v1alpha1/pipelines").WithBytes(createPipelineByte1()).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad package manager")
}

func TestCreatePipeline_CreateJobError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	pm := storage.NewFakePackageManager()
	pm.CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, &FakeBadJobStore{}, store.PipelineStore, pm))

	e.POST("/apis/v1alpha1/pipelines").WithBytes(createPipelineByte1()).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad job store")
}

func TestCreatePipeline_ValidSchedule(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	pm := storage.NewFakePackageManager()
	pm.CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore, pm))
	b, _ := json.Marshal(message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "1 0 * * *"})
	response := e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte(b)).Expect().Status(httptest.StatusOK)

	expectedResponse := message.Pipeline{
		Metadata:       &message.Metadata{ID: 1},
		Name:           "MY_PIPELINE",
		PackageId:      1,
		Schedule:       "1 0 * * *",
		Enabled:        true,
		EnabledAtInSec: 1}
	var actual message.Pipeline
	util.MarshalOrFail(response.Body().Raw(), &actual)
	assert.Equal(t, expectedResponse, actual)
	assert.Equal(t, 0, store.WorkflowClientFake.GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipeline_InvalidSchedule(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	store.PackageStore.CreatePackage(createPkg("pkg1"))
	pm := storage.NewFakePackageManager()
	pm.CreatePackageFile([]byte("kind: Workflow"), "pkg1")
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore, pm))
	b, _ := json.Marshal(message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 1,
		Schedule:  "abcdef"})

	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte(b)).Expect().
		Status(httptest.StatusBadRequest).Body().
		Equal("The pipeline schedule cannot be parsed: abcdef: Expected 5 to 6 fields, found 1: abcdef")
	assert.Equal(t, 0, store.WorkflowClientFake.GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestEnablePipeline(t *testing.T) {
	store, err := storage.NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	// Creating a pipeline
	createdPipeline := &message.Pipeline{Name: "Pipeline123"}
	err = store.PipelineStore.CreatePipeline(createdPipeline)
	assert.Nil(t, err)
	pipelineID := createdPipeline.ID

	// Verify that the created pipeline is enabled.
	createdPipeline, err = store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, createdPipeline.Enabled, "The pipeline must be enabled.")

	// Disabling the pipeline
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore,
		storage.NewFakePackageManager()))
	response := e.POST(fmt.Sprintf("/apis/v1alpha1/pipelines/%v/disable", pipelineID)).Expect().Status(
		httptest.StatusOK)
	assert.Empty(t, response.Body().Raw())

	// Verify that the created pipeline is disabled.
	createdPipeline, err = store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, false, createdPipeline.Enabled, "The pipeline must be disabled.")

	// Enabling the pipeline
	e = httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore,
		storage.NewFakePackageManager()))
	response = e.POST(fmt.Sprintf("/apis/v1alpha1/pipelines/%v/enable", pipelineID)).Expect().Status(
		httptest.StatusOK)
	assert.Empty(t, response.Body().Raw())

	// Verify that the created pipeline is enabled.
	createdPipeline, err = store.PipelineStore.GetPipeline(pipelineID)
	assert.Nil(t, err)
	assert.Equal(t, true, createdPipeline.Enabled, "The pipeline must be enabled.")
}

func TestEnablePipelineDatabaseError(t *testing.T) {
	store, err := storage.NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	err = store.Close()
	assert.Nil(t, err)

	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore,
		storage.NewFakePackageManager()))

	response := e.POST("/apis/v1alpha1/pipelines/1/enable").Expect().Status(
		httptest.StatusInternalServerError)
	assert.Contains(t, response.Body().Raw(), "Internal error: sql: database is closed")
}

func TestEnablePipelineNotFound(t *testing.T) {
	store, err := storage.NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()

	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore,
		storage.NewFakePackageManager()))

	response := e.POST("/apis/v1alpha1/pipelines/1/enable").Expect().Status(httptest.StatusNotFound)
	assert.Equal(t, "Pipeline 1 not found.", response.Body().Raw())
}

func TestListJobs(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	jobDetail1, err := store.JobStore.CreateJob(1, createWorkflow("wf1"), defaultScheduledAtInSec)
	assert.Nil(t, err)
	_, err = store.JobStore.CreateJob(2, createWorkflow("wf2"), defaultScheduledAtInSec)
	assert.Nil(t, err)

	expectedResponse := []message.Job{
		{Metadata: &message.Metadata{ID: 1},
			Name:             jobDetail1.Job.Name,
			ScheduledAtInSec: defaultScheduledAtInSec,
		}}
	e := httptest.New(t, initApiHandlerTest(nil, store.JobStore, nil, nil))

	response := e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusOK)
	var actual []message.Job
	util.MarshalOrFail(response.Body().Raw(), &actual)
	assert.Equal(t, expectedResponse, actual)
}

func TestListJobsReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeBadJobStore{}, nil, nil))

	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad job store")
}

func TestGetJob(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	jobDetail, err := store.JobStore.CreateJob(1, createWorkflow("wf1"), defaultScheduledAtInSec)
	assert.Nil(t, err)
	expectedResponse := message.JobDetail{
		Workflow: createWorkflow(jobDetail.Job.Name),
		Job: &message.Job{
			Name:             jobDetail.Job.Name,
			ScheduledAtInSec: defaultScheduledAtInSec,
			PipelineID:       0,
		}}
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore,
		storage.NewFakePackageManager()))

	response := e.GET(fmt.Sprintf("/apis/v1alpha1/pipelines/1/jobs/%v", jobDetail.Job.Name)).
		Expect().Status(httptest.StatusOK)
	var actual message.JobDetail
	util.MarshalOrFail(response.Body().Raw(), &actual)
	actual.Job.Metadata = nil
	assert.Equal(t, expectedResponse, actual)
}

func TestGetJob_InternalError(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, &FakeBadJobStore{}, store.PipelineStore,
		storage.NewFakePackageManager()))

	e.GET("/apis/v1alpha1/pipelines/1/jobs/job1").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad job store")
}

func TestGetJob_NotFound(t *testing.T) {
	store := storage.NewFakeStoreOrFatal(util.NewFakeTimeForEpoch())
	defer store.Close()
	e := httptest.New(t, initApiHandlerTest(store.PackageStore, store.JobStore, store.PipelineStore,
		storage.NewFakePackageManager()))

	e.GET("/apis/v1alpha1/pipelines/1/jobs/wf1").Expect().Status(httptest.StatusBadRequest).
		Body().Contains("Job wf1 not found")
}
