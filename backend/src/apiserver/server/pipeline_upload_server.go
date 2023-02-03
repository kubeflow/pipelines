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
	"net/url"

	"github.com/golang/glog"
	"github.com/golang/protobuf/jsonpb"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

const (
	FormFileKey               = "uploadfile"
	NameQueryStringKey        = "name"
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

// Creates a pipeline and a pipeline version.
// HTTP multipart endpoint for uploading pipeline file.
// https://www.w3.org/Protocols/rfc1341/7_2_Multipart.html
// This endpoint is not exposed through grpc endpoint, since grpc-gateway can't convert the gRPC
// endpoint to the HTTP endpoint.
// See https://github.com/grpc-ecosystem/grpc-gateway/issues/500
// Thus we create the HTTP endpoint directly and using swagger to auto generate the HTTP client.
func (s *PipelineUploadServer) UploadPipeline(w http.ResponseWriter, r *http.Request) {
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

	namespaceQuery := r.URL.Query().Get(NamespaceStringQuery)
	pipelineNamespace, err := GetPipelineNamespace(namespaceQuery)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline due to invalid pipeline namespace"))
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
	pipelineName, err := buildPipelineName(fileNameQueryString, header.Filename)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline due to invalid pipeline name"))
		return
	}
	// We don't set a max length for pipeline description here, since in our DB the description type is longtext.
	pipelineDescription, err := url.QueryUnescape(r.URL.Query().Get(DescriptionQueryStringKey))
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline due error reading pipeline description"))
		return
	}

	pipeline := &model.Pipeline{
		Name:        pipelineName,
		Description: pipelineDescription,
		Namespace:   pipelineNamespace,
	}
	newPipeline, err := s.resourceManager.CreatePipeline(pipeline)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline"))
		return
	}

	pipelineVersion := &model.PipelineVersion{
		Name:         newPipeline.Name,
		Description:  newPipeline.Description,
		PipelineId:   newPipeline.UUID,
		PipelineSpec: string(pipelineFile),
	}
	newPipelineVersion, err := s.resourceManager.CreatePipelineVersion(pipelineVersion)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline. Pipeline version creation failed"))
		return
	} else if s.options.CollectMetrics {
		pipelineVersionCount.Inc()
	}

	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{EnumsAsInts: false, OrigName: true}
	err = marshaler.Marshal(w, toApiPipelineV1(newPipeline, newPipelineVersion))
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline due to error marshalling the pipeline"))
		return
	}
}

// Creates a pipeline version under an existing pipeline.
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
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline version due to error parsing pipeline spec filename"))
		return
	}
	defer file.Close()

	pipelineFile, err := ReadPipelineFile(header.Filename, file, common.MaxFileLength)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline version due to error reading pipeline spec file"))
		return
	}

	versionNameQueryString := r.URL.Query().Get(NameQueryStringKey)
	// If new version's name is not included in query string, use file name.
	pipelineVersionName, err := buildPipelineName(versionNameQueryString, header.Filename)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusBadRequest, util.Wrap(err, "Failed to create a pipeline version due to error reading pipeline version name"))
		return
	}

	versionDescription := r.URL.Query().Get(DescriptionQueryStringKey)

	pipelineId := r.URL.Query().Get(PipelineKey)
	if len(pipelineId) == 0 {
		s.writeErrorToResponse(w, http.StatusBadRequest, errors.New("Failed to create a pipeline version due to error reading pipeline id"))
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

	newPipelineVersion, err := s.resourceManager.CreatePipelineVersion(
		&model.PipelineVersion{
			Name:         pipelineVersionName,
			Description:  versionDescription,
			PipelineId:   pipelineId,
			PipelineSpec: string(pipelineFile),
		},
	)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline version"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	marshaler := &jsonpb.Marshaler{EnumsAsInts: false, OrigName: true}
	createdPipelineVersion := toApiPipelineVersionV1(newPipelineVersion)
	if createdPipelineVersion == nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.NewInternalServerError(errors.New("Failed to convert internal pipeline version representation to its v1beta1 API counterpart"), "Failed to create a pipeline version due to error converting it to API"))
		return
	}
	err = marshaler.Marshal(w, createdPipelineVersion)
	if err != nil {
		s.writeErrorToResponse(w, http.StatusInternalServerError, util.Wrap(err, "Failed to create a pipeline version due to marshalling error"))
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

func GetPipelineNamespace(queryString string) (string, error) {
	pipelineNamespace, err := url.QueryUnescape(queryString)
	if err != nil {
		return "", util.NewInvalidInputErrorWithDetails(err, "Pipeline namespace in the query string has invalid format")
	}
	return pipelineNamespace, nil
}
