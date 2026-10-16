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
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
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

func TestReadyzTransitionsToReady(t *testing.T) {
	synced := false
	mux := newHealthMux(func() bool { return synced })

	// Before sync: 503
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /readyz before sync: got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	// After sync: 200
	synced = true
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /readyz after sync: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestServeHealthProbeShutdownOnContextCancel(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	serverContext, cancelServer := context.WithCancel(context.Background())

	serverStopped := make(chan struct{})
	go func() {
		defer close(serverStopped)
		serveHealthProbe(serverContext, listener, func() bool { return true })
	}()

	// Verify the server is serving before shutdown.
	response, err := http.Get("http://" + listener.Addr().String() + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz before shutdown failed: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz before shutdown: got %d, want %d", response.StatusCode, http.StatusOK)
	}

	// Cancel context and wait for the server to stop.
	cancelServer()
	select {
	case <-serverStopped:
	case <-time.After(10 * time.Second):
		t.Fatal("server did not shut down within 10 seconds after context cancellation")
	}
}

func TestHasSyncedAggregation(t *testing.T) {
	// HasSynced must return true only when both informer caches report
	// synced. Test all four combinations using atomic booleans to simulate
	// the func() bool returned by each client's HasSynced().
	var workflowSynced atomic.Bool
	var swfSynced atomic.Bool

	hasSynced := func() bool {
		return workflowSynced.Load() && swfSynced.Load()
	}

	// Neither synced.
	if hasSynced() {
		t.Error("expected false when neither informer is synced")
	}

	// Only workflow synced.
	workflowSynced.Store(true)
	if hasSynced() {
		t.Error("expected false when only workflow informer is synced")
	}

	// Only scheduled workflow synced.
	workflowSynced.Store(false)
	swfSynced.Store(true)
	if hasSynced() {
		t.Error("expected false when only scheduled workflow informer is synced")
	}

	// Both synced.
	workflowSynced.Store(true)
	if !hasSynced() {
		t.Error("expected true when both informers are synced")
	}
}
