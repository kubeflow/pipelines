// Copyright 2020 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"net/http"
)

// These are valid conditions of a ScheduledWorkflow.
const (
	RunKey  = "run_id"
	NodeKey = "node_id"
	Follow  = "follow"
)

type RunLogServer struct {
	resourceManager *resource.ResourceManager
	httpClient      *http.Client
}

// Log streaming endpoint
// This endpoint is not exposed through grpc endpoint, since grpc-gateway cannot handle native HTTP content streaming.
func (s *RunLogServer) ReadRunLog(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Read run log called")

	vars := mux.Vars(r)

	runId, ok := vars[RunKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf("missing path parameter: '%s')", RunKey))
		return
	}

	nodeId, ok := vars[NodeKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf("missing path parameter: '%s')", NodeKey))
		return
	}

	follow := vars[Follow] == "true" // defaults to false

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache, private")

	err := s.resourceManager.ReadLog(runId, nodeId, follow, w)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, err)
	}
}

func (s *RunLogServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to read run log. Error: %+v", err)
	w.WriteHeader(code)
	errorResponse := api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte("Error reading run log"))
	}
	w.Write(errBytes)
}

func NewRunLogServer(resourceManager *resource.ResourceManager) *RunLogServer {
	return &RunLogServer{resourceManager: resourceManager, httpClient: http.DefaultClient}
}
