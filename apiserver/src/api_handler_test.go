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
	"mime/multipart"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/storage"
	"ml/apiserver/src/util"
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakePackageStore struct{}

func (s *FakePackageStore) ListPackages() ([]pipelinemanager.Package, error) {
	packages := []pipelinemanager.Package{
		{Name: "Package123"},
		{Name: "Package456"}}
	return packages, nil
}

func (s *FakePackageStore) GetPackage(packageId uint) (pipelinemanager.Package, error) {
	pkg := pipelinemanager.Package{Name: "package123"}
	return pkg, nil
}
func (s *FakePackageStore) CreatePackage(pkg pipelinemanager.Package) (pipelinemanager.Package, error) {
	return pkg, nil
}

type FakeBadPackageStore struct{}

func (s *FakeBadPackageStore) ListPackages() ([]pipelinemanager.Package, error) {
	return nil, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) GetPackage(packageId uint) (pipelinemanager.Package, error) {
	return pipelinemanager.Package{}, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) CreatePackage(pipelinemanager.Package) (pipelinemanager.Package, error) {
	return pipelinemanager.Package{}, util.NewInternalError("bad package store", "")
}

type FakeJobStore struct{}

func (s *FakeJobStore) GetJob(name string) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "abc"},
		Status:     v1alpha1.WorkflowStatus{Phase: "Pending"}}, nil
}

func (s *FakeJobStore) ListJobs() ([]v1alpha1.Workflow, error) {
	jobs := []v1alpha1.Workflow{{ObjectMeta: v1.ObjectMeta{Name: "opq"},
		Status: v1alpha1.WorkflowStatus{Phase: "Failed"}}}
	return jobs, nil
}

func (s *FakeJobStore) CreateJob(workflow *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{Name: "xyz"},
		Status:     v1alpha1.WorkflowStatus{Phase: "Pending"}}, nil
}

type FakeBadJobStore struct{}

func (s *FakeBadJobStore) GetJob(name string) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{}, util.NewInternalError("bad job store", "")
}

func (s *FakeBadJobStore) ListJobs() ([]v1alpha1.Workflow, error) {
	return nil, util.NewInternalError("bad job store", "")
}

func (s *FakeBadJobStore) CreateJob(workflow *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return &v1alpha1.Workflow{}, util.NewInternalError("bad job store", "")
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

func (s *FakePipelineStore) ListPipelines() ([]pipelinemanager.Pipeline, error) {
	pipelines := []pipelinemanager.Pipeline{
		{Name: "p1", PackageId: 123},
		{Name: "p2", PackageId: 345}}
	return pipelines, nil
}

func (s *FakePipelineStore) GetPipeline(id uint) (pipelinemanager.Pipeline, error) {
	return pipelinemanager.Pipeline{Name: "p", PackageId: 123}, nil
}

func (s *FakePipelineStore) CreatePipeline(p pipelinemanager.Pipeline) (pipelinemanager.Pipeline, error) {
	return pipelinemanager.Pipeline{Name: "p", PackageId: 123}, nil
}

type FakeBadPipelineStore struct{}

func (s *FakeBadPipelineStore) ListPipelines() ([]pipelinemanager.Pipeline, error) {
	return nil, util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) GetPipeline(id uint) (pipelinemanager.Pipeline, error) {
	return pipelinemanager.Pipeline{}, util.NewInternalError("bad pipeline store", "")
}

func (s *FakeBadPipelineStore) CreatePipeline(pipelinemanager.Pipeline) (pipelinemanager.Pipeline, error) {
	return pipelinemanager.Pipeline{}, util.NewInternalError("bad pipeline store", "")
}

func initApiHandlerTest(
	ps storage.PackageStoreInterface,
	js storage.JobStoreInterface,
	pls storage.PipelineStoreInterface,
	pm storage.PackageManagerInterface) *iris.Application {
	clientManager := ClientManager{packageStore: ps, jobStore: js, pipelineStore: pls, packageManager: pm}
	return newApp(clientManager)
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
		Body().Equal("[{\"name\":\"p1\",\"packageId\":123},{\"name\":\"p2\",\"packageId\":345}]")
}

func TestListPipelinesError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakeBadPipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad pipeline store")
}

func TestGetPipeline(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakePipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines/1").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"name\":\"p\",\"packageId\":123}")

}

func TestGetPipelineError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, nil, &FakeBadPipelineStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines/1").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad pipeline store")
}

func TestCreatePipeline(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte("{}")).Expect().Status(httptest.StatusOK).
		Body().Equal("{\"name\":\"p\",\"packageId\":123}")
}

func TestCreatePipelineError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeBadJobStore{}, &FakeBadPipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusBadRequest).
		Body().Contains("The pipeline has invalid format.")
}

func TestListJobs(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeJobStore{}, nil, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"metadata\":{\"name\":\"opq\",\"creationTimestamp\":null}," +
		"\"spec\":{\"templates\":null,\"entrypoint\":\"\",\"arguments\":{}}," +
		"\"status\":{\"phase\":\"Failed\",\"startedAt\":null,\"finishedAt\":null,\"nodes\":null}}]")
}

func TestListJobsReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeBadJobStore{}, nil, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad job store")
}

func TestGetJob(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/pipelines/1/jobs/job1").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"metadata\":{\"name\":\"abc\",\"creationTimestamp\":null}," +
		"\"spec\":{\"templates\":null,\"entrypoint\":\"\",\"arguments\":{}}," +
		"\"status\":{\"phase\":\"Pending\",\"startedAt\":null,\"finishedAt\":null,\"nodes\":null}}")
}

func TestGetJobError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeBadJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/pipelines/1/jobs/job1").Expect().Status(httptest.StatusInternalServerError).
		Body().Equal("bad job store")
}
