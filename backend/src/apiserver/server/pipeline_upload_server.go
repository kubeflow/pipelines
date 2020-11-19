// Copyright 2018 Google LLC
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
	"net/url"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// These are valid conditions of a ScheduledWorkflow.
const (
	FormFileKey               = "uploadfile"
	NameQueryStringKey        = "name"
	DescriptionQueryStringKey = "description"
	// Pipeline Id in the query string specifies a pipeline when creating versions.
	PipelineKey = "pipelineid"
)

// Metric variables. Please prefix the metric names with pipeline_upload_ or pipeline_version_upload_.
var (
	uploadPipelineRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_upload_requests",
		Help: "The number of pipeline upload requests",
	})

	uploadPipelineVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_version_upload_requests",
		Help: "The number of pipeline version upload requests",
	})

	// TODO(jingzhang36): error count and success count.
)

type PipelineUploadServerOptions struct {
	CollectMetrics bool
}

type PipelineUploadServer struct {
	resourceManager *resource.ResourceManager
	options         *PipelineUploadServerOptions
}

// HTTP multipart endpoint for uploading pipeline file.
// https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html
// This endpoint is not exposed through grpc endpoint, since grpc-gateway can't convert the gRPC
// endpoint to the HTTP endpoint.
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/500
// Thus we create the HTTP endpoint directly and using swagger to auto generate the HTTP client.
func (s *PipelineUploadServer) UploadPipeline(w http.ResponseWriter, r *http.Request) {
	if s.options.CollectMetrics {
		uploadPipelineRequests.Inc()
	}

	glog.Infof("Upload pipeline called")
	file, header, err := r.FormFile(FormFileKey)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to read pipeline from file"))
		return
	}
	defer file.Close()

	pipelineFile, err := ReadPipelineFile(header.Filename, file, MaxFileLength)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Error read pipeline file."))
		return
	}

	fileNameQueryString := r.URL.Query().Get(NameQueryStringKey)
	pipelineName, err := GetPipelineName(fileNameQueryString, header.Filename)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Invalid pipeline name."))
		return
	}
	// We don't set a max length for pipeline description here, since in our DB the description type is longtext.
	pipelineDescription, err := url.QueryUnescape(r.URL.Query().Get(DescriptionQueryStringKey))
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Error read pipeline description."))
		return
	}
	newPipeline, err := s.resourceManager.CreatePipeline(pipelineName, pipelineDescription, pipelineFile)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating pipeline"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{EnumsAsInts: false, OrigName: true}
	err = marshaler.Marshal(w, ToApiPipeline(newPipeline))
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating pipeline"))
		return
	}
}

// HTTP multipart endpoint for uploading pipeline version file.
// https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html
// This endpoint is not exposed through grpc endpoint, since grpc-gateway can't convert the gRPC
// endpoint to the HTTP endpoint.
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/500
// Thus we create the HTTP endpoint directly and using swagger to auto generate the HTTP client.
func (s *PipelineUploadServer) UploadPipelineVersion(w http.ResponseWriter, r *http.Request) {
	if s.options.CollectMetrics {
		uploadPipelineVersionRequests.Inc()
	}

	glog.Infof("Upload pipeline version called")
	file, header, err := r.FormFile(FormFileKey)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to read pipeline version from file"))
		return
	}
	defer file.Close()

	pipelineFile, err := ReadPipelineFile(header.Filename, file, MaxFileLength)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Error read pipeline version file."))
		return
	}

	versionNameQueryString := r.URL.Query().Get(NameQueryStringKey)
	// If new version's name is not included in query string, use file name.
	pipelineVersionName, err := GetPipelineName(versionNameQueryString, header.Filename)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Invalid pipeline version name."))
		return
	}

	pipelineId := r.URL.Query().Get(PipelineKey)
	if len(pipelineId) == 0 {
		s.writeErrorToResponse(w, http.StatusBadRequest, errors.New("Please specify a pipeline id when creating versions."))
		return
	}

	newPipelineVersion, err := s.resourceManager.CreatePipelineVersion(
		&api.PipelineVersion{
			Name: pipelineVersionName,
			ResourceReferences: []*api.ResourceReference{
				&api.ResourceReference{
					Key: &api.ResourceKey{
						Id:   pipelineId,
						Type: api.ResourceType_PIPELINE,
					},
					Relationship: api.Relationship_OWNER,
				},
			},
		}, pipelineFile, common.IsPipelineVersionUpdatedByDefault())
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating pipeline version"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{EnumsAsInts: false, OrigName: true}
	createdPipelineVersion, err := ToApiPipelineVersion(newPipelineVersion)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating pipeline version"))
		return
	}
	err = marshaler.Marshal(w, createdPipelineVersion)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Error creating pipeline version"))
		return
	}

	if s.options.CollectMetrics {
		pipelineCount.Inc()
	}
}

func (s *PipelineUploadServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to upload pipelines. Error: %+v", err)
	w.WriteHeader(code)
	errorResponse := api.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte("Error uploading pipeline"))
	}
	w.Write(errBytes)
}

func NewPipelineUploadServer(resourceManager *resource.ResourceManager, options *PipelineUploadServerOptions) *PipelineUploadServer {
	return &PipelineUploadServer{resourceManager: resourceManager, options: options}
}
