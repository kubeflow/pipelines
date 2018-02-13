package main

import (
	"github.com/googleprivate/ml/webserver/src/message"
	"net/http"
	"net/http/httptest"
	"testing"
	"errors"
)

type FakeTemplateDao struct {
}

func (dao *FakeTemplateDao) ListTemplate() ([]message.Template, error) {
	templates := []message.Template{{Id: 123, Description: "first description"}, {Id: 456, Description: "second description"}}
	return templates, nil
}

func (dao *FakeTemplateDao) GetTemplate(templateId string) (message.Template, error) {
	template := message.Template{Id: 123, Description: "first description"}
	return template, nil
}

type FakeBadTemplateDao struct {
}

func (dao *FakeBadTemplateDao ) ListTemplate() ([]message.Template, error) {
	return nil, errors.New("there is no template here")
}

func (dao *FakeBadTemplateDao ) GetTemplate(templateId string) (message.Template, error) {
	return message.Template{}, errors.New("there is no template here")
}

func TestListTemplate(t *testing.T) {
	handler := &APIHandler{
		templateDao: &FakeTemplateDao{},
	}
	req, err := http.NewRequest("GET", "/templates", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler.ListTemplates(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if rr.Body.String() != "[{\"id\":123,\"description\":\"first description\"},{\"id\":456,\"description\":\"second description\"}]\n" {
		t.Errorf("JSON payload is not generated correctly")
	}
}

func TestListTemplateReturnError(t *testing.T) {

	handler := &APIHandler{
		templateDao: &FakeBadTemplateDao{},
	}
	req, err := http.NewRequest("GET", "/templates", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler.ListTemplates(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}

func TestGetTemplate(t *testing.T) {
	handler := &APIHandler{
		templateDao: &FakeTemplateDao{},
	}
	req, err := http.NewRequest("GET", "/templates/123", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler.GetTemplates(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if rr.Body.String() != "{\"id\":123,\"description\":\"first description\"}\n" {
		t.Errorf("JSON payload is not generated correctly")
	}
}

func TestGetTemplateReturnError(t *testing.T) {
	handler := &APIHandler{
		templateDao: &FakeBadTemplateDao{},
	}
	req, err := http.NewRequest("GET", "/templates/123", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler.GetTemplates(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusInternalServerError)
	}
}
