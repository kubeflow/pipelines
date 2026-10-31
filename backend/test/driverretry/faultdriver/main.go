// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The faultdriver runs the production executor plugin against a real KFP API.
// Its only substitution is at the HTTP response boundary, after the production
// handler has committed the driver's side effects. It is built only for E2E.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/driver/executorplugin"
)

func main() {
	handler, err := executorplugin.NewHandler(context.Background(), executorplugin.DefaultTokenPath, nil)
	if err != nil {
		log.Fatal(err)
	}
	faults := &faultHandler{next: handler, directory: "/kfp/log", crash: func() { os.Exit(86) }}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/template.execute", faults)
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Fatal(server.ListenAndServe())
}

type faultHandler struct {
	next      http.Handler
	directory string
	crash     func()
	mu        sync.Mutex
}

func (h *faultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()
	r.Body = io.NopCloser(bytes.NewReader(body))
	var request struct {
		Template struct {
			Plugin json.RawMessage `json:"plugin"`
		} `json:"template"`
	}
	_ = json.Unmarshal(body, &request)
	args, owned, parseErr := driverapi.ParseDriverPluginArgs(request.Template.Plugin)
	response := httptest.NewRecorder()
	h.next.ServeHTTP(response, r)
	var result driverapi.DriverResponse
	if owned && parseErr == nil && !args.DriverRetryFinalize && response.Code == http.StatusOK && json.Unmarshal(response.Body.Bytes(), &result) == nil && result.Node.Phase == "Succeeded" {
		if args.TaskName == "restart-driver" {
			h.mu.Lock()
			// The shared Argo emptyDir survives a container restart. O_EXCL also
			// prevents overlapping polls from crashing more than once.
			marker := filepath.Join(h.directory, "lost-response-"+args.RunID)
			f, createErr := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			switch {
			case createErr == nil:
				_, writeErr := f.Write(response.Body.Bytes())
				syncErr := f.Sync()
				closeErr := f.Close()
				if writeErr != nil || syncErr != nil || closeErr != nil {
					log.Fatal("cannot persist committed response marker")
				}
				log.Printf("driver-retry E2E: committed task %s; dropping successful response and restarting plugin", args.TaskName)
				h.crash()
			case !os.IsExist(createErr):
				log.Fatal(createErr)
			default:
				committed, readErr := os.ReadFile(marker)
				if readErr != nil {
					log.Fatal(readErr)
				}
				if !bytes.Equal(committed, response.Body.Bytes()) {
					result.Node.Phase = "Failed"
					result.Node.Message = "driver-retry E2E: replay changed the committed task ID or handoff outputs"
					response.Body.Reset()
					if err := json.NewEncoder(response.Body).Encode(result); err != nil {
						log.Fatal(err)
					}
				}
			}
			h.mu.Unlock()
		}
		if args.TaskName == "exhaust-driver" && args.DriverRetryGeneration == "0" {
			result.Node.Phase = "Failed"
			result.Node.Message = fmt.Sprintf("driver-retry E2E: lost committed response, attempt %s", args.DriverRetryAttempt)
			// Returning Failed exercises Argo's actual retry and exit-hook ordering.
			// The next manual generation passes through the production response.
			response.Body.Reset()
			if err := json.NewEncoder(response.Body).Encode(result); err != nil {
				log.Fatal(err)
			}
		}
	}
	for key, values := range response.Header() {
		w.Header()[key] = values
	}
	w.WriteHeader(response.Code)
	_, _ = w.Write(response.Body.Bytes())
}
