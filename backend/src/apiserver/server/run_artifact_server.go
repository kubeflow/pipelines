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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	authorizationv1 "k8s.io/api/authorization/v1"
)

const (
	ArtifactNameKey = "artifact_name"
)

type RunArtifactServer struct {
	resourceManager *resource.ResourceManager
}

// Artifact streaming endpoint that streams artifacts directly from object storage
// to the HTTP response without buffering the entire content in memory.
// No size limits are imposed - the streaming approach itself provides the security benefit.
func (s *RunArtifactServer) StreamArtifactV1(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Stream artifact v1 called")

	vars := mux.Vars(r)

	runId, ok := vars[RunKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf("missing path parameter: '%s'", RunKey))
		return
	}

	nodeId, ok := vars[NodeKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf("missing path parameter: '%s'", NodeKey))
		return
	}

	artifactName, ok := vars[ArtifactNameKey]
	if !ok {
		s.writeErrorToResponse(w, http.StatusBadRequest, fmt.Errorf("missing path parameter: '%s'", ArtifactNameKey))
		return
	}

	// Perform authorization check
	err := s.canAccessRun(r.Context(), runId, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReadArtifact})
	if err != nil {
		s.writeErrorToResponse(w, http.StatusForbidden, fmt.Errorf("unauthorized to read artifact: %v", err))
		return
	}

	// Set headers for binary content streaming
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache, private")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", artifactName))
	w.WriteHeader(http.StatusOK)

	// Stream the artifact directly to the response
	err = s.resourceManager.StreamArtifact(r.Context(), runId, nodeId, artifactName, w)
	if err != nil {
		glog.Errorf("Failed to stream artifact: %v", err)
		// Since we've already started writing the response, we can't change the status code
		// Just log the error and close the connection
		return
	}
}

// StreamArtifact handles v2beta1 artifact streaming (same implementation as v1)
func (s *RunArtifactServer) StreamArtifact(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Stream artifact v2 called")
	s.StreamArtifactV1(w, r)
}

// canAccessRun checks if the user can access the specified run
func (s *RunArtifactServer) canAccessRun(ctx context.Context, runId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	// This is a simplified authorization check. In a real implementation,
	// you would need to integrate with the proper authorization system.
	// For now, we'll just return nil to allow access.
	// TODO: Implement proper authorization check similar to run_server.go
	return nil
}

func (s *RunArtifactServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to stream artifact. Error: %+v", err)
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	errorResponse := &api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte(`{"error_message": "Error streaming artifact"}`))
		return
	}
	w.Write(errBytes)
}

func NewRunArtifactServer(resourceManager *resource.ResourceManager) *RunArtifactServer {
	return &RunArtifactServer{resourceManager: resourceManager}
}