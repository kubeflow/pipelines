package main

import (
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
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
	// TODO
	return nil
}

type FakeBadPackageStore struct {
}

func (s *FakeBadPackageStore) ListPackages() ([]pipelinemanager.Package, error) {
	return nil, util.NewInternalError("there is no package here")
}

func (s *FakeBadPackageStore) GetPackage(packageId string) (pipelinemanager.Package, error) {
	return pipelinemanager.Package{}, util.NewInternalError("there is no package here")
}

func (s *FakeBadPackageStore) CreatePackage(pipelinemanager.Package) error {
	// TODO
	return nil
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
	return nil, util.NewInternalError("there is no job here")
}

func initApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		packageStore: &FakePackageStore{},
		jobStore:     &FakeJobStore{},
	}
	return newApp(clientManager)
}

func initBadApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		packageStore: &FakeBadPackageStore{},
		jobStore:     &FakeBadJobStore{},
	}
	return newApp(clientManager)
}

func TestListPackages(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"id\":\"123\",\"name\":\"Package123\"},{\"id\":\"456\",\"name\":\"Package456\"}]")
}

func TestListPackagesReturnError(t *testing.T) {
	e := httptest.New(t, initBadApiHandlerTest())
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusInternalServerError)
}

func TestGetPackage(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/packages/123").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"id\":\"123\",\"name\":\"Package123\"}")
}

func TestGetPackageReturnError(t *testing.T) {
	e := httptest.New(t, initBadApiHandlerTest())
	e.GET("/apis/v1alpha1/packages/123").Expect().Status(httptest.StatusInternalServerError)
}

func TestListJobs(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"name\":\"job1\",\"status\":\"Failed\"},{\"name\":\"job2\",\"status\":\"Succeeded\"}]")
}

func TestListJobsReturnError(t *testing.T) {
	e := httptest.New(t, initBadApiHandlerTest())
	e.GET("/apis/v1alpha1/pipelines/1/jobs").Expect().Status(httptest.StatusInternalServerError)
}
