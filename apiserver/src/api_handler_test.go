package main

import (
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"mime/multipart"
	"bytes"
	"ml/apiserver/src/storage"
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

func (s *FakeJobStore) ListJobs() ([]pipelinemanager.Job, error) {
	jobs := []pipelinemanager.Job{
		{Name: "job1", Status: "Failed"},
		{Name: "job2", Status: "Succeeded"}}
	return jobs, nil
}

func (s *FakeJobStore) CreateJob([]byte) (pipelinemanager.Job, error) {
	return pipelinemanager.Job{Name: "job1", Status: "Failed"}, nil
}

type FakeBadJobStore struct{}

func (s *FakeBadJobStore) ListJobs() ([]pipelinemanager.Job, error) {
	return nil, util.NewInternalError("bad job store", "")
}

func (s *FakeBadJobStore) CreateJob([]byte) (pipelinemanager.Job, error) {
	return pipelinemanager.Job{}, util.NewInternalError("bad job store", "")
}

type FakePackageManager struct{}

func (m *FakePackageManager) CreatePackageFile(template []byte, fileName string) error {
	return nil
}

func (m *FakePackageManager) GetPackageFile(fileName string) ([]byte, error) {
	return []byte("kind: Workflow"), nil
}

type FakeBadPackageManager struct{}

func (m *FakeBadPackageManager) CreatePackageFile(template []byte, fileName string) error {
	return util.NewInternalError("bad package manager", "")
}

func (m *FakeBadPackageManager) GetPackageFile(fileName string) ([]byte, error) {
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
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, &FakePipelineStore{}, nil))
	e.POST("/apis/v1alpha1/pipelines").WithBytes([]byte("{}")).Expect().Status(httptest.StatusOK).
			Body().Equal("{\"name\":\"p\",\"packageId\":123}")
}

func TestCreatePipelineError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, &FakeBadPipelineStore{}, nil))
	e.POST("/apis/v1alpha1/pipelines").Expect().Status(httptest.StatusBadRequest).
			Body().Contains("Invalid input")
}

func TestListJobs(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeJobStore{}, nil, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusOK).
			Body().Equal("[{\"name\":\"job1\",\"status\":\"Failed\"},{\"name\":\"job2\",\"status\":\"Succeeded\"}]")
}

func TestListJobsReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeBadJobStore{}, nil, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad job store")
}

func TestCreateJob(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusOK).
			Body().Equal("{\"name\":\"job1\",\"status\":\"Failed\"}")
}

func TestCreateJobError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, &FakeBadJobStore{}, &FakePipelineStore{}, &FakePackageManager{}))
	e.POST("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad job store")
}
