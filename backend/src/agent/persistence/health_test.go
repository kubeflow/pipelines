// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzReturns200(t *testing.T) {
	mux := newHealthMux(func() bool { return false })
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz: got %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("GET /healthz body: got %q, want %q", rec.Body.String(), "ok")
	}
}

func TestReadyzNotReady(t *testing.T) {
	mux := newHealthMux(func() bool { return false })
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz (not synced): got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestReadyzReady(t *testing.T) {
	mux := newHealthMux(func() bool { return true })
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /readyz (synced): got %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("GET /readyz body: got %q, want %q", rec.Body.String(), "ok")
	}
}
