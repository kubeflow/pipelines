package util

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

type FooStruct struct {
	Foo string
	Bar string
}

func TestHandleJSONPayload(t *testing.T) {
	w := httptest.NewRecorder()
	f := FooStruct{Foo: "abc", Bar: "def"}
	HandleJSONPayload(w, f)
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Response header is not set up correctly")
	}
	if w.Body.String() != "{\"Foo\":\"abc\",\"Bar\":\"def\"}\n" {
		t.Errorf("JSON payload is not generated correctly")
	}
}

func TestHandleJSONPayloadErr(t *testing.T) {
	w := httptest.NewRecorder()
	HandleJSONPayload(w, math.Inf(1))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Response status is not set up correctly. Expecting an internal server error")
	}
}
