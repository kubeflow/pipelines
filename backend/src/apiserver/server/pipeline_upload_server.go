// Copyright 2018 The Kubeflow Authors
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

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/golang/glog"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

const (
	FormFileKey               = "uploadfile"
	NameQueryStringKey        = "name"
	DisplayNameQueryStringKey = "display_name"
	DescriptionQueryStringKey = "description"
	NamespaceStringQuery      = "namespace"
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
	CollectMetrics bool `json:"collect_metrics,omitempty"`
	// ApiVersion       string `default:"v2beta1" json:"api_version,omitempty"`
	// DefaultNamespace string `default:"" json:"default_namespace,omitempty"`
}

type PipelineUploadServer struct {
	resourceManager *resource.ResourceManager
	options         *PipelineUploadServerOptions
}

func (s *PipelineUploadServer) UploadPipelineV1(w http.ResponseWriter, r *http.Request) {
	s.uploadPipeline("v1beta1", w, r)
}

func (s *PipelineUploadServer) UploadPipeline(w http.ResponseWriter, r *http.Request) {
	s.uploadPipeline("v2beta1", w, r)
}

// Creates a pipeline and a pipeline version.
// HTTP multipart endpoint for uploading pipeline file.
// https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html
// This endpoint is not exposed through grpc endpoint, since grpc-gateway can't convert the gRPC
// endpoint to the HTTP endpoint.
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/500
// Thus we create the HTTP endpoint directly and using swagger to auto generate the HTTP client.
func (s *PipelineUploadServer) uploadPipeline(api_version string, w http.ResponseWriter, r *http.Request) {
	if s.options.CollectMetrics {
		uploadPipelineRequests.Inc()
		uploadPipelineVersionRequests.Inc()
	}

	glog.Infof("Upload pipeline called")
	file, header, err := r.FormFile(FormFileKey)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to read pipeline from file"))
		return
	}
	defer file.Close()

	pipelineFile, err := ReadPipelineFile(header.Filename, file, common.MaxFileLength)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to read a pipeline spec file"))
		return
	}

	pipelineNamespace := r.URL.Query().Get(NamespaceStringQuery)
	if err := validation.ValidateFieldLength("Pipeline", "Namespace", pipelineNamespace); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pipelineNamespace = s.resourceManager.ReplaceNamespace(pipelineNamespace)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: pipelineNamespace,
		Verb:      common.RbacResourceVerbCreate,
	}
	err = s.canUploadVersionedPipeline(r, "", resourceAttributes)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline due to authorization error"))
		return
	}

	fileNameQueryString := r.URL.Query().Get(NameQueryStringKey)
	displayNameQueryString := r.URL.Query().Get(DisplayNameQueryStringKey)
	pipelineName := buildPipelineName(fileNameQueryString, displayNameQueryString, header.Filename)
	displayName := displayNameQueryString
	if displayName == "" {
		displayName = pipelineName
	}

	pipeline := &model.Pipeline{
		Name:        pipelineName,
		DisplayName: displayName,
		Description: r.URL.Query().Get(DescriptionQueryStringKey),
		Namespace:   pipelineNamespace,
	}

	pipelineVersion := &model.PipelineVersion{
		Name:         pipeline.Name,
		DisplayName:  pipeline.DisplayName,
		Description:  pipeline.Description,
		PipelineSpec: string(pipelineFile),
	}

	if err := validation.ValidateFieldLength("Pipeline", "Name", pipeline.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	newPipeline, newPipelineVersion, err := s.resourceManager.CreatePipelineAndPipelineVersion(pipeline, pipelineVersion)
	if err != nil {
		if util.IsUserErrorCodeMatch(err, codes.AlreadyExists) {
			s.writeErrorToResponse(w, http.StatusConflict, util.Wrap(err, "Failed to create a pipeline and a pipeline version. The pipeline already exists."))
			return
		}
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline and a pipeline version"))
		return
	}

	if s.options.CollectMetrics {
		pipelineVersionCount.Inc()
	}

	var messageToMarshal proto.Message
	if api_version == "v1beta1" {
		messageToMarshal = toApiPipelineV1(newPipeline, newPipelineVersion)
	} else if api_version == "v2beta1" {
		messageToMarshal = toApiPipeline(newPipeline)
	} else {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline. Invalid API version"))
		return
	}

	// Marshal the message to bytes
	marshaler := &protojson.MarshalOptions{
		UseProtoNames: true,
		// Note: Default behavior in protojson is to output enum names (strings).
	}
	data, err := marshaler.Marshal(messageToMarshal)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline. Marshaling error"))
		return
	}
	_, err = w.Write(data)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline. Write error."))
		return
	}
}

func (s *PipelineUploadServer) UploadPipelineVersionV1(w http.ResponseWriter, r *http.Request) {
	s.uploadPipelineVersion("v1beta1", w, r)
}

func (s *PipelineUploadServer) UploadPipelineVersion(w http.ResponseWriter, r *http.Request) {
	s.uploadPipelineVersion("v2beta1", w, r)
}

// Creates a pipeline version under an existing pipeline.
// HTTP multipart endpoint for uploading pipeline version file.
// https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html
// This endpoint is not exposed through grpc endpoint, since grpc-gateway can't convert the gRPC
// endpoint to the HTTP endpoint.
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/500
// Thus we create the HTTP endpoint directly and using swagger to auto generate the HTTP client.
func (s *PipelineUploadServer) uploadPipelineVersion(api_version string, w http.ResponseWriter, r *http.Request) {
	if s.options.CollectMetrics {
		uploadPipelineVersionRequests.Inc()
	}

	glog.Infof("Upload pipeline version called")
	file, header, err := r.FormFile(FormFileKey)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline version due to error parsing pipeline spec filename"))
		return
	}
	defer file.Close()

	pipelineFile, err := ReadPipelineFile(header.Filename, file, common.MaxFileLength)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline version due to error reading pipeline spec file"))
		return
	}
	pipelineId := r.URL.Query().Get(PipelineKey)
	if pipelineId == "" {
		s.writeErrorToResponse(w, http.StatusBadRequest, errors.New("Failed to create a pipeline version due to error reading pipeline id"))
		return
	}

	versionNameQueryString := r.URL.Query().Get(NameQueryStringKey)
	versionDisplayNameQueryString := r.URL.Query().Get(DisplayNameQueryStringKey)
	pipelineVersionName := buildPipelineName(versionNameQueryString, versionDisplayNameQueryString, header.Filename)

	displayName := versionDisplayNameQueryString
	if displayName == "" {
		displayName = pipelineVersionName
	}

	// Validate PipelineVersion name length
	if err := validation.ValidateFieldLength("PipelineVersion", "Name", pipelineVersionName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	namespace, err := s.resourceManager.FetchNamespaceFromPipelineId(pipelineId)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline version due to error reading namespace"))
		return
	}

	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbCreate,
	}
	err = s.canUploadVersionedPipeline(r, pipelineId, resourceAttributes)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline version due to authorization error"))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	newPipelineVersion, err := s.resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:         pipelineVersionName,
			DisplayName:  displayName,
			Description:  r.URL.Query().Get(DescriptionQueryStringKey),
			PipelineId:   pipelineId,
			PipelineSpec: string(pipelineFile),
		},
	)
	if err != nil {
		if util.IsUserErrorCodeMatch(err, codes.AlreadyExists) {
			s.writeErrorToResponse(w, http.StatusConflict, util.Wrap(err, "Failed to create a pipeline version. The pipeline already exists."))
			return
		}
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline version"))
		return
	}

	var messageToMarshal proto.Message
	if api_version == "v1beta1" {
		messageToMarshal = toApiPipelineVersionV1(newPipelineVersion)
	} else if api_version == "v2beta1" {
		messageToMarshal = toApiPipelineVersion(newPipelineVersion)
	} else {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline version. Invalid API version"))
		return
	}
	// Marshal the message to bytes
	marshaler := &protojson.MarshalOptions{
		UseProtoNames: true,
		// Note: Default behavior in protojson is to output enum names (strings).
	}
	data, err := marshaler.Marshal(messageToMarshal)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline version. Marshaling error"))
		return
	}
	_, err = w.Write(data)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline version. Write error."))
		return
	}

	if s.options.CollectMetrics {
		pipelineVersionCount.Inc()
	}
}

func (s *PipelineUploadServer) canUploadVersionedPipeline(r *http.Request, pipelineId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if len(pipelineId) > 0 {
		namespace, err := s.resourceManager.FetchNamespaceFromPipelineId(pipelineId)
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the Pipeline ID")
		}
		if len(resourceAttributes.Namespace) == 0 {
			resourceAttributes.Namespace = namespace
		}
	}
	if resourceAttributes.Namespace == "" {
		return nil
	}

	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypePipelines

	ctx := context.Background()
	md := metadata.MD{}
	for key, values := range r.Header {
		md.Set(key, values...)
	}
	ctx = metadata.NewIncomingContext(ctx, md)

	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Authorization Failure")
	}
	return nil
}

func (s *PipelineUploadServer) writeErrorToResponse(w http.ResponseWriter, code int, err error) {
	glog.Errorf("Failed to upload pipelines. Error: %+v", err)
	w.WriteHeader(code)
	errorResponse := &apiv1beta1.Error{ErrorMessage: err.Error(), ErrorDetails: fmt.Sprintf("%+v", err)}
	errBytes, err := json.Marshal(errorResponse)
	if err != nil {
		w.Write([]byte("Error uploading pipeline"))
	}
	w.Write(errBytes)
}

func NewPipelineUploadServer(resourceManager *resource.ResourceManager, options *PipelineUploadServerOptions) *PipelineUploadServer {
	return &PipelineUploadServer{resourceManager: resourceManager, options: options}
}
