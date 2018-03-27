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
	"errors"
	"mime/multipart"
	"ml/src/message"
	"ml/src/storage"
	"ml/src/util"
	"testing"
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakePackageStore struct{}

func (s *FakePackageStore) ListPackages() ([]message.Package, error) {
	packages := []message.Package{
		{Name: "Package123"},
		{Name: "Package456"}}
	return packages, nil
}

func (s *FakePackageStore) GetPackage(packageId uint) (message.Package, error) {
	pkg := message.Package{Name: "package123"}
	return pkg, nil
}
func (s *FakePackageStore) CreatePackage(pkg message.Package) (message.Package, error) {
	return pkg, nil
}

type FakeBadPackageStore struct{}

func (s *FakeBadPackageStore) ListPackages() ([]message.Package, error) {
	return nil, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) GetPackage(packageId uint) (message.Package, error) {
	return message.Package{}, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) CreatePackage(message.Package) (message.Package, error) {
	return message.Package{}, util.NewInternalError("bad package store", "")
}

type FakeJobStore struct{}

func (s *FakeJobStore) GetJob(pipelineId uint, name string) (message.JobDetail, error) {
	return message.JobDetail{Workflow: &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "abc"},
		Status:     v1alpha1.WorkflowStatus{Phase: "Pending"}}}, nil
}

func (s *FakeJobStore) ListJobs(pipelineId uint) ([]message.Job, error) {
	jobs := []message.Job{{Name: "job1"}, {Name: "job2"}}
	return jobs, nil
}

func (s *FakeJobStore) CreateJob(pipelineId uint, workflow *v1alpha1.Workflow) (message.JobDetail, error) {
	return message.JobDetail{Workflow: &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "xyz"},
		Status:     v1alpha1.WorkflowStatus{Phase: "Pending"}}}, nil
}

type FakeBadJobStore struct{}

func (s *FakeBadJobStore) GetJob(pipelineId uint, name string) (message.JobDetail, error) {
	return message.JobDetail{}, util.NewInternalError("bad job store", "")
}

func (s *FakeBadJobStore) ListJobs(pipelineId uint) ([]message.Job, error) {
	return nil, util.NewInternalError("bad job store", "")
}

func (s *FakeBadJobStore) CreateJob(pipelineId uint, workflow *v1alpha1.Workflow) (message.JobDetail, error) {
	return message.JobDetail{}, util.NewInternalError("bad job store", "")
}

type FakePackageManager struct{}

func (m *FakePackageManager) CreatePackageFile(template []byte, fileName string) error {
	return nil
}

func (m *FakePackageManager) GetTemplate(fileName string) ([]byte, error) {
	return []byte("kind: Workflow"), nil
}

type FakeBadPackageManager struct{}

func (m *FakeBadPackageManager) CreatePackageFile(template []byte, fileName string) error {
	return util.NewInternalError("bad package manager", "")
}

func (m *FakeBadPackageManager) GetTemplate(fileName string) ([]byte, error) {
	return nil, util.NewInternalError("bad package manager", "")
}

type FakePipelineStore struct{}

func (s *FakePipelineStore) ListPipelines() ([]message.Pipeline, error) {
	pipelines := []message.Pipeline{
		{Name: "p1", PackageId: 123},
		{Name: "p2", PackageId: 345}}
	return pipelines, nil
}

func (s *FakePipelineStore) GetPipeline(id uint) (message.Pipeline, error) {
	return message.Pipeline{Name: "p", PackageId: 123}, nil
}

func (s *FakePipelineStore) CreatePipeline(p message.Pipeline) (message.Pipeline, error) {
	pipeline := message.Pipeline{
		Metadata:  &message.Metadata{ID: 1},
		Name:      "p",
		PackageId: 123,
		Schedule:  p.Schedule}
	return pipeline, nil
}

func (s *FakePipelineStore) GetPipelineAndLatestJobIterator() (
	*storage.PipelineAndLatestJobIterator, error) {
	return nil, errors.New("Not implemented.")
}

type FakeBadPipelineStore struct{}

func (s *FakeBadPipelineStore) ListPipelines() ([]message.Pipeline, error) {
	return nil, util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) GetPipeline(id uint) (message.Pipeline, error) {
	return message.Pipeline{}, util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) CreatePipeline(message.Pipeline) (message.Pipeline, error) {
	return message.Pipeline{}, util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) GetPipelineAndLatestJobIterator() (
		*storage.PipelineAndLatestJobIterator, error) {
	return nil, errors.New("Not implemented.")
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
		packageManager: pm}
	return newApp(clientManager)
}

func createAPIHandler(
	pkgStore storage.PackageStoreInterface,
	jobStore storage.JobStoreInterface,
	pipelineStore storage.PipelineStoreInterface,
	pkgManager storage.PackageManagerInterface) *APIHandler {
	clientManager := ClientManager{
		packageStore:   pkgStore,
		jobStore:       jobStore,
		pipelineStore:  pipelineStore,
		packageManager: pkgManager}
	return newAPIHandler(clientManager)
}

func TestListPackages(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil, nil))
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"name\":\"Package123\"},{\"name\":\"Package456\"}]")
}

func TestListPackagesReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, nil))
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusInternalServerError)
}

func TestGetPackage(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil, nil))
	e.GET("/apis/v1alpha1/packages/123").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"name\":\"package123\"}")
}

func TestGetPackageReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, nil))
	e.GET("/apis/v1alpha1/packages/123").Expect().Status(httptest.StatusInternalServerError)
}

func TestUploadPackage(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil, &FakePackageManager{}))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusOK).Body().Contains("\"name\":\"hello-world.yaml\"")
}

func TestUploadPackageCreatePackageFileError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil, &FakeBadPackageManager{}))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad package manager")
}

func TestUploadPackageGetFormFileError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil, &FakePackageManager{}))
	var b bytes.Buffer
	b.WriteString("I am invalid file")
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusBadRequest).Body().Equal("Failed to read package.")
}

func TestUploadPackageCreatePackageError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, &FakePackageManager{}))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	e.POST("/apis/v1alpha1/packages/upload").
		WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
		Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad package store")
}

func TestUploadPackageGetParametersError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, &FakePackageManager{}))
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
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/packages/123/templates").Expect().Status(httptest.StatusOK).
		Body().Equal("kind: Workflow")
}

func TestGetTemplateGetPackageError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/packages/123/templates").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad package store")
}

func TestGetTemplateGetPackageFileError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil, &FakeBadPackageManager{}))
	e.GET("/apis/v1alpha1/packages/123/templates").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad package manager")
}

func TestListPipelines(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakePipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"name\":\"p1\",\"packageId\":123,\"schedule\":\"\",\"enabled\":false,\"enabledAt\":0},{\"name\":\"p2\",\"packageId\":345,\"schedule\":\"\",\"enabled\":false,\"enabledAt\":0}]")
}

func TestListPipelinesError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakeBadPipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad pipeline store")
}

func TestGetPipeline(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakePipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines/1").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"name\":\"p\",\"packageId\":123,\"schedule\":\"\",\"enabled\":false,\"enabledAt\":0}")
}

func TestGetPipelineError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakeBadPipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines/1").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad pipeline store")
}

func TestCreatePipeline(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte("{}")).Expect().Status(httptest.StatusOK).
		Body().Equal("{\"id\":1,\"createdAt\":\"0001-01-01T00:00:00Z\",\"name\":\"p\",\"packageId\":123,\"schedule\":\"\",\"enabled\":false,\"enabledAt\":0}")
}

func TestCreatePipelineBadPipelineFormatError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusBadRequest).
		Body().Contains("The pipeline has invalid format.")
}

func TestCreatePipelinePackageNotExistError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte("{}")).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad package store")
}

func TestCreatePipelineMetadataError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeJobStore{}, &FakeBadPipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte("{}")).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad pipeline store")
}

func TestCreatePipelineGetTemplateError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakeBadPackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte("{}")).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad package manager")
}

func TestCreatePipelineCreateJobError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeBadJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte("{}")).Expect().Status(httptest.StatusInternalServerError).
		Body().Contains("bad job store")
}

func TestCreatePipelineInternalNoSchedule(t *testing.T) {
	store, err := storage.NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()
	apiHandler := createAPIHandler(&FakePackageStore{}, store.JobStore, store.PipelineStore,
		&FakePackageManager{})

	result, err, errPrefix := apiHandler.createPipelineInternal(message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 123})

	assert.Nil(t, err, "There should not be an error: %v", err)
	result.Metadata.CreatedAt = time.Unix(0, 0) // Not testing this field
	result.Metadata.UpdatedAt = time.Unix(0, 0) // Not testing this field

	expected := &message.Pipeline{
		Metadata: &message.Metadata{
			ID:        1,
			CreatedAt: time.Unix(0, 0),
			UpdatedAt: time.Unix(0, 0)},
		Name:       "MY_PIPELINE",
		PackageId:  123,
		Schedule:   "",
		Enabled:    true,
		EnabledAtInSec: 1}

	assert.Equalf(t, expected, result, "Unexpected pipeline structure. Expect %v. Got %v.",
		expected, result)
	assert.Empty(t, errPrefix)
	assert.Equal(t, 1, store.WorkflowClientFake.GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipelineInternalValidSchedule(t *testing.T) {
	store, err := storage.NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()
	apiHandler := createAPIHandler(&FakePackageStore{}, store.JobStore, store.PipelineStore,
		&FakePackageManager{})

	result, err, errPrefix := apiHandler.createPipelineInternal(message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 123,
		Schedule:  "1 0 * * *"})

	assert.Nil(t, err, "There should not be an error: %v", err)

	result.Metadata.CreatedAt = time.Unix(0, 0) // Not testing this field
	result.Metadata.UpdatedAt = time.Unix(0, 0) // Not testing this field

	expected := &message.Pipeline{
		Metadata: &message.Metadata{
			ID:        1,
			CreatedAt: time.Unix(0, 0),
			UpdatedAt: time.Unix(0, 0)},
		Name:       "MY_PIPELINE",
		PackageId:  123,
		Schedule:   "1 0 * * *",
		Enabled:    true,
		EnabledAtInSec: 1}

	assert.Equalf(t, expected, result, "Unexpected pipeline structure. Expect %v. Got %v.",
		expected, result)
	assert.Empty(t, errPrefix)
	assert.Equal(t, 0, store.WorkflowClientFake.GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestCreatePipelineInternalInvalidSchedule(t *testing.T) {
	store, err := storage.NewFakeStore(util.NewFakeTimeForEpoch())
	assert.Nil(t, err)
	defer store.Close()
	apiHandler := createAPIHandler(&FakePackageStore{}, store.JobStore, store.PipelineStore,
		&FakePackageManager{})

	result, err, errPrefix := apiHandler.createPipelineInternal(message.Pipeline{
		Name:      "MY_PIPELINE",
		PackageId: 123,
		Schedule:  "abcdef"})

	assert.Contains(t, err.Error(),
		"Invalid input: The pipeline schedule cannot be parsed: abcdef: Expected 5 to 6 fields")
	assert.Nil(t, result)
	assert.Equal(t, errPrefix, "CreatePipeline_ValidSchedule")
	assert.Equal(t, 0, store.WorkflowClientFake.GetWorkflowCount(), "Unexpected number of workflows.")
}

func TestListJobs(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeJobStore{}, nil, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"name\":\"job1\",\"scheduledAt\":0},{\"name\":\"job2\",\"scheduledAt\":0}]")
}

func TestListJobsReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeBadJobStore{}, nil, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad job store")
}

func TestGetJob(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/pipelines/1/jobs/job1").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"job\":{\"metadata\":{\"name\":\"abc\",\"creationTimestamp\":null}," +
		"\"spec\":{\"templates\":null,\"entrypoint\":\"\",\"arguments\":{}},\"status\":" +
		"{\"phase\":\"Pending\",\"startedAt\":null,\"finishedAt\":null,\"nodes\":null}}}")
}

func TestGetJobError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeBadJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/pipelines/1/jobs/job1").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad job store")
}
