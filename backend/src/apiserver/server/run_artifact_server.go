// Copyright 2025 The Kubeflow Authors
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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

const (
	ArtifactNameKey          = "artifact_name"
	missingParamErrorMessage = "missing path parameter: '%s'"
)

type RunArtifactServer struct {
	*BaseRunServer
}

// ReadArtifact is an artifact reading endpoint that streams artifacts from object storage,
// encodes them to base64 on-the-fly, and returns them as JSON.
// The streaming approach allows handling large artifacts without buffering everything in memory.
func (s *RunArtifactServer) ReadArtifact(response http.ResponseWriter, r *http.Request) {
	glog.Infof("Read artifact v2 called")

	vars := mux.Vars(r)

	runID, ok := vars[RunKey]

	if !ok {
		s.writeErrorToResponse(response, http.StatusBadRequest, fmt.Errorf(missingParamErrorMessage, RunKey))
		return
	}

	nodeID, ok := vars[NodeKey]
	if !ok {
		s.writeErrorToResponse(response, http.StatusBadRequest, fmt.Errorf(missingParamErrorMessage, NodeKey))
		return
	}

	artifactName, ok := vars[ArtifactNameKey]
	if !ok {
		s.writeErrorToResponse(response, http.StatusBadRequest, fmt.Errorf(missingParamErrorMessage, ArtifactNameKey))
		return
	}

	err := s.canAccessRun(r.Context(), runID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReadArtifact})
	if err != nil {
		s.writeErrorToResponse(response, http.StatusForbidden, util.Wrap(err, "Failed to authorize the request"))
		return
	}

	artifactFileExists, err := s.artifactFileExists(r.Context(), runID, nodeID, artifactName)
	if err != nil {
		s.writeErrorToResponse(response, http.StatusInternalServerError, err)
		return
	} else if !artifactFileExists {
		s.writeErrorToResponse(response, http.StatusNotFound, fmt.Errorf("artifact not found: %v", err))
		return
	}

	artifactPath, err := s.resourceManager.ResolveArtifactPath(runID, nodeID, artifactName)
	if err != nil {
		s.writeErrorToResponse(response, http.StatusInternalServerError, err)
		return
	}

	reader, err := s.resourceManager.ObjectStore().GetFileReader(r.Context(), artifactPath)
	if err != nil {
		s.writeErrorToResponse(response, http.StatusInternalServerError, fmt.Errorf("failed to get file reader: %v", err))
		return
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			glog.Warningf("Failed to close artifact reader: %v", closeErr)
		}
	}()

	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-cache, private")

	// Write the opening JSON structure
	response.WriteHeader(http.StatusOK)
	if _, err := response.Write([]byte(`{"data":"`)); err != nil {
		glog.Errorf("Failed to write JSON opening: %v", err)
		return
	}

	// Create a base64 encoder that writes directly to the response
	encoder := base64.NewEncoder(base64.StdEncoding, response)

	// Stream the content through the base64 encoder
	if _, err := io.Copy(encoder, reader); err != nil {
		glog.Errorf("Failed to stream and encode artifact: %v", err)
		// We've already started writing the response, so we can't change the status code
		return
	}

	if err := encoder.Close(); err != nil {
		glog.Errorf("Failed to close base64 encoder: %v", err)
		return
	}

	if _, err := response.Write([]byte(`"}`)); err != nil {
		glog.Errorf("Failed to write JSON closing: %v", err)
		return
	}
}

func (s *RunArtifactServer) artifactFileExists(ctx context.Context, runID string, nodeID string, artifactName string) (bool, error) {
	artifactPath, err := s.resourceManager.ResolveArtifactPath(runID, nodeID, artifactName)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	reader, err := s.resourceManager.ObjectStore().GetFileReader(ctx, artifactPath)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			glog.Warningf("Failed to close artifact reader: %v", closeErr)
		}
	}()
	return true, nil
}

// ReadArtifactV1 handles v1 artifact reading (delegates to v2 implementation)
func (s *RunArtifactServer) ReadArtifactV1(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Read artifact v1 called")
	s.ReadArtifact(w, r)
}

func (s *RunArtifactServer) writeErrorToResponse(response http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to read artifact. Error: %+v", err)
	response.WriteHeader(code)
	response.Header().Set("Content-Type", "application/json")
	errorResponse := &api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		if _, writeErr := response.Write([]byte(`{"error_message": "Error streaming artifact"}`)); writeErr != nil {
			glog.Errorf("Failed to write fallback error response: %v", writeErr)
		}
		return
	}
	if _, writeErr := response.Write(errBytes); writeErr != nil {
		glog.Errorf("Failed to write error response: %v", writeErr)
	}
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "Not found") ||
		strings.Contains(errMsg, "NotFound") || strings.Contains(errMsg, "ResourceNotFoundError")
}

func NewRunArtifactServer(resourceManager *resource.ResourceManager) *RunArtifactServer {
	return &RunArtifactServer{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager,
			options:         &RunServerOptions{CollectMetrics: false},
		},
	}
}
