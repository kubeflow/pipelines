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
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

const (
	RunKey  = "run_id"
	NodeKey = "node_id"
)

type RunLogServer struct {
	*BaseRunServer
	httpClient *http.Client
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

	follow := r.URL.Query().Get("follow") == "true" // defaults to false

	if err := s.authorize(r, runId); err != nil {
		s.writeErrorToResponse(w, http.StatusForbidden, util.Wrap(err, "Failed to authorize the request"))
		return
	}

	// Set the success headers without committing a status code: the first
	// streamed log write sends 200 implicitly, while a validation failure
	// inside ReadLog can still produce a non-2xx JSON response.
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache, private")

	dst := &runLogResponseWriter{ResponseWriter: w, follow: follow}
	err := s.resourceManager.ReadLog(r.Context(), runId, nodeId, follow, dst)
	if r.Context().Err() != nil {
		return
	}
	if err != nil {
		if dst.written {
			glog.Errorf("Run log stream ended after response started: %v", err)
			return
		}
		s.writeErrorToResponse(w, http.StatusInternalServerError, err)
	}
}

// Delay committing headers until a log write so validation errors remain JSON.
// Once streaming starts, flush short writes and never append an error response.
type runLogResponseWriter struct {
	http.ResponseWriter
	follow  bool
	written bool
}

func (w *runLogResponseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if n > 0 {
		w.written = true
		if w.follow {
			if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
	return n, err
}

// This route is registered directly on the mux rather than through grpc-gateway,
// so the caller identity arrives in the request headers, not in gRPC metadata.
// Copy the headers into metadata so the authenticators can read them, as
// canUploadVersionedPipeline does.
// The dedicated readLog verb keeps log access behind a SubjectAccessReview even
// in shared read mode, which auto-approves the shared get/list verbs.
func (s *RunLogServer) authorize(r *http.Request, runID string) error {
	md := metadata.MD{}
	for key, values := range r.Header {
		md.Set(key, values...)
	}
	ctx := metadata.NewIncomingContext(r.Context(), md)
	return s.canAccessRun(ctx, runID, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbReadLog})
}

func (s *RunLogServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to read run log. Error: %+v", err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	errorResponse := &apiError{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte("Error reading run log"))
	}
	w.Write(errBytes)
}

func NewRunLogServer(resourceManager *resource.ResourceManager) *RunLogServer {
	return &RunLogServer{
		BaseRunServer: &BaseRunServer{
			resourceManager: resourceManager,
			options:         &RunServerOptions{CollectMetrics: false},
		},
		httpClient: http.DefaultClient,
	}
}
