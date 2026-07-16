// Copyright 2024 The Kubeflow Authors
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
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

// RunIntermediateInputsServer handles the HTTP endpoint for supplying
// human-provided parameter values to a paused (suspend) pipeline node.
//
// Endpoint: POST /apis/v2beta1/runs/{run_id}/nodes/{node_id}:setIntermediateInputs
type RunIntermediateInputsServer struct {
	*BaseRunServer
}

// setIntermediateInputsBody is the JSON request body for the endpoint.
type setIntermediateInputsBody struct {
	// Parameters maps parameter name → human-supplied string value.
	Parameters map[string]string `json:"parameters"`
}

// SetIntermediateInputs handles POST /apis/v2beta1/runs/{run_id}/nodes/{node_id}:setIntermediateInputs.
//
// The caller supplies the run ID and node display name in the path, plus a JSON
// body of the form:
//
//	{"parameters": {"decision": "YES"}}
//
// On success the node's output parameters are persisted and the Argo workflow
// is resumed. The response is 200 OK with an empty JSON object {}.
func (s *RunIntermediateInputsServer) SetIntermediateInputs(w http.ResponseWriter, r *http.Request) {
	glog.Infof("SetIntermediateInputs called")

	vars := mux.Vars(r)

	runID, ok := vars[RunKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf(missingParamErrorMessage, RunKey))
		return
	}

	nodeID, ok := vars[NodeKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf(missingParamErrorMessage, NodeKey))
		return
	}

	// Authorization: caller must have update permission on the run.
	if err := s.canAccessRun(r.Context(), runID,
		&authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbUpdate}); err != nil {
		s.writeErrorToResponse(w, http.StatusForbidden,
			util.Wrap(err, "Failed to authorize the request"))
		return
	}

	// Read and parse the request body.
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest,
			fmt.Errorf("failed to read request body: %v", err))
		return
	}
	defer func() { _ = r.Body.Close() }()

	var body setIntermediateInputsBody
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest,
			fmt.Errorf("invalid JSON body: %v", err))
		return
	}

	if len(body.Parameters) == 0 {
		s.writeErrorToResponse(w, http.StatusBadRequest,
			fmt.Errorf("request body must contain at least one parameter"))
		return
	}

	req := resource.SetRunIntermediateInputsRequest{
		RunID:           runID,
		NodeDisplayName: nodeID,
		Parameters:      body.Parameters,
	}

	if err := s.resourceManager.SetRunIntermediateInputs(r.Context(), req); err != nil {
		httpStatus := httpStatusFromError(err)
		s.writeErrorToResponse(w, httpStatus, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write([]byte("{}")); writeErr != nil {
		glog.Warningf("SetIntermediateInputs: failed to write response: %v", writeErr)
	}
}

// GetIntermediateInputsStatus handles GET /apis/v2beta1/runs/{run_id}/nodes/{node_id}/intermediateInputs.
//
// Returns the current state of the suspend node: whether it is waiting for
// input and the parameter metadata (description, default, enum choices, any
// already-supplied value).  This is a read-only endpoint and does not modify
// the workflow.
func (s *RunIntermediateInputsServer) GetIntermediateInputsStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	runID, ok := vars[RunKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf(missingParamErrorMessage, RunKey))
		return
	}

	nodeID, ok := vars[NodeKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf(missingParamErrorMessage, NodeKey))
		return
	}

	// Authorization: read permission is sufficient for a status query.
	if err := s.canAccessRun(r.Context(), runID,
		&authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet}); err != nil {
		s.writeErrorToResponse(w, http.StatusForbidden,
			util.Wrap(err, "Failed to authorize the request"))
		return
	}

	status, err := s.resourceManager.GetRunIntermediateInputsStatus(r.Context(), runID, nodeID)
	if err != nil {
		s.writeErrorToResponse(w, httpStatusFromError(err), err)
		return
	}

	respBytes, err := json.Marshal(status)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError,
			fmt.Errorf("failed to marshal response: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, writeErr := w.Write(respBytes); writeErr != nil {
		glog.Warningf("GetIntermediateInputsStatus: failed to write response: %v", writeErr)
	}
}

// writeErrorToResponse writes a JSON error response with the given HTTP status code.
func (s *RunIntermediateInputsServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("SetIntermediateInputs failed. Error: %+v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	errResp := &api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, marshalErr := json.Marshal(errResp)
	if marshalErr != nil {
		if _, writeErr := w.Write([]byte(`{"error_message":"internal error"}`)); writeErr != nil {
			glog.Errorf("SetIntermediateInputs: failed to write fallback error response: %v", writeErr)
		}
		return
	}
	if _, writeErr := w.Write(errBytes); writeErr != nil {
		glog.Errorf("SetIntermediateInputs: failed to write error response: %v", writeErr)
	}
}

// httpStatusFromError maps KFP util error types to HTTP status codes.
func httpStatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	userError, ok := err.(*util.UserError)
	if !ok {
		return http.StatusInternalServerError
	}
	// Map gRPC status codes to HTTP equivalents.
	switch userError.ExternalStatusCode() {
	case 3: // codes.InvalidArgument
		return http.StatusBadRequest
	case 5: // codes.NotFound
		return http.StatusNotFound
	case 7: // codes.PermissionDenied
		return http.StatusForbidden
	case 9: // codes.FailedPrecondition
		return http.StatusBadRequest
	case 10: // codes.Aborted
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// NewRunIntermediateInputsServer creates a new RunIntermediateInputsServer.
func NewRunIntermediateInputsServer(resourceManager *resource.ResourceManager) *RunIntermediateInputsServer {
	return &RunIntermediateInputsServer{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager,
			options:         &RunServerOptions{CollectMetrics: false},
		},
	}
}
