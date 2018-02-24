package main

import (
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/util"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
)

type FakePackageDao struct{}

func (dao *FakePackageDao) ListPackages() ([]pipelinemanager.Package, error) {
	packages := []pipelinemanager.Package{
		{Id: 123, Description: "first description"},
		{Id: 456, Description: "second description"}}
	return packages, nil
}

func (dao *FakePackageDao) GetPackage(packageId string) (pipelinemanager.Package, error) {
	pkg := pipelinemanager.Package{Id: 123, Description: "first description"}
	return pkg, nil
}

type FakeBadPackageDao struct {
}

func (dao *FakeBadPackageDao) ListPackages() ([]pipelinemanager.Package, error) {
	return nil, util.NewInternalError("there is no package here")
}

func (dao *FakeBadPackageDao) GetPackage(packageId string) (pipelinemanager.Package, error) {
	return pipelinemanager.Package{}, util.NewInternalError("there is no package here")
}

type FakeJobDao struct{}

func (dao *FakeJobDao) ListJobs() ([]pipelinemanager.Job, error) {
	jobs := []pipelinemanager.Job{
		{Name: "job1", Status: "Failed"},
		{Name: "job2", Status: "Succeeded"}}
	return jobs, nil
}

type FakeBadJobDao struct{}

func (dao *FakeBadJobDao) ListJobs() ([]pipelinemanager.Job, error) {
	return nil, util.NewInternalError("there is no job here")
}

func initApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		packageDao: &FakePackageDao{},
		jobDao:     &FakeJobDao{},
	}
	return newApp(clientManager)
}

func initBadApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		packageDao: &FakeBadPackageDao{},
		jobDao:     &FakeBadJobDao{},
	}
	return newApp(clientManager)
}

func TestListPackages(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"id\":123,\"description\":\"first description\"},{\"id\":456,\"description\":\"second description\"}]")
}

func TestListPackagesReturnError(t *testing.T) {
	e := httptest.New(t, initBadApiHandlerTest())
	e.GET("/apis/v1alpha1/packages").Expect().Status(httptest.StatusInternalServerError)
}

func TestGetPackage(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/packages/123").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"id\":123,\"description\":\"first description\"}")
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
