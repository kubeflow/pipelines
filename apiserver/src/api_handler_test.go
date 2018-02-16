package main

import (
	"github.com/googleprivate/ml/webserver/src/dao"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListTemplate(t *testing.T) {
	handler := &APIHandler{
		templateDao: &dao.FakeTemplateDao{},
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
		templateDao: &dao.FakeBadTemplateDao{},
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
		templateDao: &dao.FakeTemplateDao{},
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
		templateDao: &dao.FakeBadTemplateDao{},
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
