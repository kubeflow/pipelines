package main

import (
	"testing"

	"errors"

	"github.com/googleprivate/ml/apiserver/src/message"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
)

type FakeTemplateDao struct {
}

func (dao *FakeTemplateDao) ListTemplate() ([]message.Template, error) {
	templates := []message.Template{
		{Id: 123, Description: "first description"},
		{Id: 456, Description: "second description"}}
	return templates, nil
}

func (dao *FakeTemplateDao) GetTemplate(templateId string) (message.Template, error) {
	template := message.Template{Id: 123, Description: "first description"}
	return template, nil
}

type FakeBadTemplateDao struct {
}

func (dao *FakeBadTemplateDao) ListTemplate() ([]message.Template, error) {
	return nil, errors.New("there is no template here")
}

func (dao *FakeBadTemplateDao) GetTemplate(templateId string) (message.Template, error) {
	return message.Template{}, errors.New("there is no template here")
}

func initApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		templateDao: &FakeTemplateDao{},
	}
	return newApp(clientManager)
}

func initBadApiHandlerTest() *iris.Application {
	clientManager := ClientManager{
		templateDao: &FakeBadTemplateDao{},
	}
	return newApp(clientManager)
}

func TestListTemplate(t *testing.T) {
	e := httptest.New(t, initApiHandlerTest())
	e.GET("/apis/v1alpha1/templates").Expect().Status(httptest.StatusOK).
		Body().Equal("[{\"id\":123,\"description\":\"first description\"},{\"id\":456,\"description\":\"second description\"}]")
}

func TestListTemplateReturnError(t *testing.T) {
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
