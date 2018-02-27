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
	"ml/apiserver/src/storage/packagemanager"
)

type FakePackageStore struct{}

func (s *FakePackageStore) ListPackages() ([]pipelinemanager.Package, error) {
	packages := []pipelinemanager.Package{
		{Id: "123", Name: "Package123"},
		{Id: "456", Name: "Package456"}}
	return packages, nil
}

func (s *FakePackageStore) GetPackage(packageId string) (pipelinemanager.Package, error) {
	pkg := pipelinemanager.Package{Id: "123", Name: "Package123"}
	return pkg, nil
}
func (s *FakePackageStore) CreatePackage(pipelinemanager.Package) error {
	return nil
}

type FakeBadPackageStore struct{}

func (s *FakeBadPackageStore) ListPackages() ([]pipelinemanager.Package, error) {
	return nil, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) GetPackage(packageId string) (pipelinemanager.Package, error) {
	return pipelinemanager.Package{}, util.NewInternalError("bad package store", "")
}

func (s *FakeBadPackageStore) CreatePackage(pipelinemanager.Package) error {
	return util.NewInternalError("bad package store", "")
}

type FakeJobStore struct{}

func (s *FakeJobStore) ListJobs() ([]pipelinemanager.Job, error) {
	jobs := []pipelinemanager.Job{
		{Name: "job1", Status: "Failed"},
		{Name: "job2", Status: "Succeeded"}}
	return jobs, nil
}

type FakeBadJobStore struct{}

func (s *FakeBadJobStore) ListJobs() ([]pipelinemanager.Job, error) {
	return nil, util.NewInternalError("there is no job here", "")
}

type FakePackageManager struct{}

func (m *FakePackageManager) CreatePackageFile(file multipart.File, fileHeader *multipart.FileHeader) error {
	return nil
}

func (m *FakePackageManager) GetPackageFile(fileName string) ([]byte, error) {
	return []byte("I am a template"), nil
}

type FakeBadPackageManager struct{}

func (m *FakeBadPackageManager) CreatePackageFile(file multipart.File, fileHeader *multipart.FileHeader) error {
	return util.NewInternalError("bad package manager", "")
}

func (m *FakeBadPackageManager) GetPackageFile(fileName string) ([]byte, error) {
	return nil, util.NewInternalError("bad package manager", "")
}

func initApiHandlerTest(ps storage.PackageStoreInterface, js storage.JobStoreInterface, pm packagemanager.PackageManagerInterface) *iris.Application {
	clientManager := ClientManager{packageStore: ps, jobStore: js, packageManager: pm}
	return newApp(clientManager)
}

func TestListPackages(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil))
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusOK).
			Body().Equal("[{\"id\":\"123\",\"name\":\"Package123\"},{\"id\":\"456\",\"name\":\"Package456\"}]")
}

func TestListPackagesReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil))
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusInternalServerError)
}

func TestGetPackage(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, nil))
	e.GET("/apis/v1alpha1/packages/123").Expect().Status(httptest.StatusOK).
			Body().Equal("{\"id\":\"123\",\"name\":\"Package123\"}")
}

func TestGetPackageReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, nil))
	e.GET("/apis/v1alpha1/packages/123").Expect().Status(httptest.StatusInternalServerError)
}

func TestUploadPackage(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, &FakePackageManager{}))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	e.POST("/apis/v1alpha1/packages/upload").
			WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
			Expect().Status(httptest.StatusOK).Body().Contains("\"name\":\"hello-world.yaml\"")
}

func TestUploadPackageCreatePackageFileError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, &FakeBadPackageManager{}))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	e.POST("/apis/v1alpha1/packages/upload").
			WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
			Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad package manager")
}

func TestUploadPackageCreatePackageError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, &FakePackageManager{}))
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	w.CreateFormFile("uploadfile", "hello-world.yaml")
	w.Close()
	e.POST("/apis/v1alpha1/packages/upload").
			WithHeader("Content-Type", w.FormDataContentType()).WithBytes(b.Bytes()).
			Expect().Status(httptest.StatusInternalServerError).Body().Equal("bad package store")
}

func TestGetTemplate(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/packages/123/templates").Expect().Status(httptest.StatusOK).
			Body().Equal("I am a template")
}

func TestGetTemplateGetPackageError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakeBadPackageStore{}, nil, &FakePackageManager{}))
	e.GET("/apis/v1alpha1/packages/123/templates").Expect().Status(httptest.StatusInternalServerError).
			Body().Equal("bad package store")
}

func TestGetTemplateGetPackageFileError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(&FakePackageStore{}, nil, &FakeBadPackageManager{}))
	e.GET("/apis/v1alpha1/packages/123/templates").Expect().Status(httptest.StatusInternalServerError).
			Body().Equal("bad package manager")

}

func TestListJobs(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeJobStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusOK).
			Body().Equal("[{\"name\":\"job1\",\"status\":\"Failed\"},{\"name\":\"job2\",\"status\":\"Succeeded\"}]")
}

func TestListJobsReturnError(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest(nil, &FakeBadJobStore{}, nil))
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusInternalServerError)
}
