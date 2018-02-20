package main

import (
	"errors"
	"testing"

	"ml/apiserver/src/message/pipelinemanager"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
)

type FakeTemplateDao struct{}

func (dao *FakeTemplateDao) ListTemplates() ([]pipelinemanager.Template, error) {
	templates := []pipelinemanager.Template{
		{Id: 123, Description: "first description"},
		{Id: 456, Description: "second description"}}
	return templates, nil
}

func (dao *FakeTemplateDao) GetTemplate(templateId string) (pipelinemanager.Template, error) {
	template := pipelinemanager.Template{Id: 123, Description: "first description"}
	return template, nil
}

type FakeBadTemplateDao struct {
}

func (dao *FakeBadTemplateDao) ListTemplates() ([]pipelinemanager.Template, error) {
	return nil, errors.New("there is no template here")
}

func (dao *FakeBadTemplateDao) GetTemplate(templateId string) (pipelinemanager.Template, error) {
	return pipelinemanager.Template{}, errors.New("there is no template here")
}

type FakeRunDao struct{}

func (dao *FakeRunDao) ListRuns() ([]pipelinemanager.Run, error) {
	runs := []pipelinemanager.Run{
		{Name: "run1", Status: "Failed"},
		{Name: "run2", Status: "Succeeded"}}
	return runs, nil
}

type FakeBadRunDao struct{}

func (dao *FakeBadRunDao) ListRuns() ([]pipelinemanager.Run, error) {
	return nil, errors.New("there is no run here")
}

func initApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		templateDao: &FakeTemplateDao{},
		runDao:      &FakeRunDao{},
	}
	return newApp(clientManager)
}

func initBadApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		templateDao: &FakeBadTemplateDao{},
		runDao:      &FakeBadRunDao{},
	}
	return newApp(clientManager)
}

func TestListTemplates(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/templates").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"id\":123,\"description\":\"first description\"},{\"id\":456,\"description\":\"second description\"}]")
}

func TestListTemplatesReturnError(t *testing.T) {
	e := httptest.New(t, initBadApiHandlerTest())
	e.GET("/apis/v1alpha1/templates").Expect().Status(httptest.StatusInternalServerError)
}

func TestGetTemplate(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/templates/123").Expect().Status(httptest.StatusOK).
		Body().Equal("{\"id\":123,\"description\":\"first description\"}")
}

func TestGetTemplateReturnError(t *testing.T) {
	e := httptest.New(t, initBadApiHandlerTest())
	e.GET("/apis/v1alpha1/templates/123").Expect().Status(httptest.StatusInternalServerError)
}

func TestListRuns(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/runs").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"name\":\"run1\",\"createdAt\":\"\",\"startedAt\":\"\",\"finishedAt\":\"\",\"status\":\"Failed\"},{\"name\":\"run2\",\"createdAt\":\"\",\"startedAt\":\"\",\"finishedAt\":\"\",\"status\":\"Succeeded\"}]")
}

func TestListRunsReturnError(t *testing.T) {
	e := httptest.New(t, initBadApiHandlerTest())
	e.GET("/apis/v1alpha1/runs").Expect().Status(httptest.StatusInternalServerError)
}
