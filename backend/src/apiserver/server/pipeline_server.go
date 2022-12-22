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
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metric variables. Please prefix the metric names with pipeline_server_.
var (
	// Used to calculate the request rate.
	createPipelineRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_create_requests",
		Help: "The total number of CreatePipeline requests",
	})

	getPipelineRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_get_requests",
		Help: "The total number of GetPipeline requests",
	})

	listPipelineRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_list_requests",
		Help: "The total number of ListPipelines requests",
	})

	deletePipelineRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_delete_requests",
		Help: "The total number of DeletePipeline requests",
	})

	createPipelineVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_create_version_requests",
		Help: "The total number of CreatePipelineVersion requests",
	})

	getPipelineVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_get_version_requests",
		Help: "The total number of GetPipelineVersion requests",
	})

	listPipelineVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_list_version_requests",
		Help: "The total number of ListPipelineVersions requests",
	})

	deletePipelineVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_delete_version_requests",
		Help: "The total number of DeletePipelineVersion requests",
	})

	// TODO(jingzhang36): error count and success count.

	pipelineCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pipeline_server_pipeline_count",
		Help: "The current number of pipelines in Kubeflow Pipelines instance",
	})

	updatePipelineDefaultVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_update_default_version_requests",
		Help: "The total number of UpdatePipelineDefaultVersion requests",
	})
)

type PipelineServerOptions struct {
	CollectMetrics   bool   `json:"collect_metrics,omitempty"`
	ApiVersion       string `default:"v2beta1" json:"api_version,omitempty"`
	DefaultNamespace string `default:"default" json:"default_namespace,omitempty"`
}

type PipelineServer struct {
	resourceManager *resource.ResourceManager
	httpClient      *http.Client
	options         *PipelineServerOptions
}

// Create a Pipeline (if doesn't exist) and a PipelineVersion
func (s *PipelineServer) CreatePipelineV1(ctx context.Context, request *apiv1beta1.CreatePipelineRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
	}

	// Make sure to convert API to Model in the beginning and never use the API messages.
	pipeline, err := model.ToModelPipeline(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): Failed to convert to the Pipeline", s.options.ApiVersion))
	}

	// Validate the model object and fill the missing data
	if err := ValidatePipelineForCreation(pipeline, s.options); err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): Failed to validate the Pipeline", s.options.ApiVersion))
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: pipeline.GetFieldValue("Namespace"),
		Verb:      common.RbacResourceVerbCreate,
		Resource:  common.RbacResourceTypePipelines,
	}
	err = isAuthorized(s.resourceManager, ctx, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): Failed to authorize with API", s.options.ApiVersion))
	}

	pipelineName, err := GetPipelineName(pipeline.GetFieldValue("Name"), pipelineFileName)
	createdPipeline, err := s.resourceManager.CreatePipelineV1(pipelineName, pipeline.GetFieldValue("Description"), pipeline.GetFieldValue("Namespace"), pipelineFile)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): Create pipeline failed.", s.options.ApiVersion))
	}
	if s.options.CollectMetrics {
		pipelineCount.Inc()
	}

	// Create PipelineVersion to support v1beta1 behavior
	if rPipelineVersion := request.GetPipeline().GetDefaultVersion(); rPipelineVersion != nil {
		if s.options.CollectMetrics {
			createPipelineVersionRequests.Inc()
		}
		pipelineVersion, err := model.ToModelPipelineVersion(rPipelineVersion)
		if err != nil {
			return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): Failed to convert to the default PipelineVersion", s.options.ApiVersion))
		}

	} else {
		pipelineUrl := pipeline.GetFieldValue("Url")
		resp, err := s.httpClient.Get(pipelineUrl)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, util.NewInternalServerError(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): Failed to download the pipeline from %v "+
				"Please double check the URL is valid and can be accessed by the pipeline system.", s.options.ApiVersion), pipelineUrl)
		}
		pipelineFileName := path.Base(pipelineUrl)
		pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, MaxFileLength)
		if err != nil {
			return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): The URL is valid but pipeline system failed to read the file.", s.options.ApiVersion))
		}

		if err != nil {
			return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v1beta1): Invalid pipeline name.", s.options.ApiVersion))
		}

	}

	return ToApiPipeline(createdPipeline), nil
}

// Creates a Pipeline, but does not create a PipelineVersion. Need to call create a PipelineVersion after creating of a Pipeline.
func (s *PipelineServer) CreatePipeline(ctx context.Context, request *apiv2beta1.CreatePipelineRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
	}

	// Validates the display_name and namespace
	if err := ValidatePipelineForCreation(request, s.options); err != nil {
		return nil, err
	}
	pipelineName := request.Pipeline.GetDisplayName()
	namespace := request.Pipeline.GetNamespace()

	// pipelineUrl := request.Pipeline.Url.PipelineUrl
	// resp, err := s.httpClient.Get(pipelineUrl)
	// if err != nil || resp.StatusCode != http.StatusOK {
	// 	return nil, util.NewInternalServerError(err, "Failed to download the pipeline from %v "+
	// 		"Please double check the URL is valid and can be accessed by the pipeline system.", pipelineUrl)
	// }
	// pipelineFileName := path.Base(pipelineUrl)
	// pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, MaxFileLength)
	// if err != nil {
	// 	return nil, util.Wrap(err, "The URL is valid but pipeline system failed to read the file.")
	// }
	// pipelineName, err := GetPipelineName(request.Pipeline.Name, pipelineFileName)
	// if err != nil {
	// 	return nil, util.Wrap(err, "Invalid pipeline name.")
	// }
	// resourceReferences := request.Pipeline.GetResourceReferences()
	// namespace := common.GetNamespaceFromAPIResourceReferences(resourceReferences)

	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbCreate,
		Resource:  common.RbacResourceTypePipelines,
	}
	if err := isAuthorized(s.resourceManager, ctx, resourceAttributes); err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Creating Pipeline(*v2beta1): Failed to authorize with API", s.options.ApiVersion))
	}

	pipeline, err := s.resourceManager.CreatePipeline(pipelineName, request.Pipeline.Description, namespace, pipelineFile)
	if err != nil {
		return nil, util.Wrap(err, "Create pipeline failed.")
	}
	if s.options.CollectMetrics {
		pipelineCount.Inc()
	}

	return ToApiPipeline(pipeline), nil
}

func (s *PipelineServer) UpdatePipelineDefaultVersionV1(ctx context.Context, request *apiv1beta1.UpdatePipelineDefaultVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		updatePipelineDefaultVersionRequests.Inc()
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbUpdate,
	}
	err := s.CanAccessPipeline(ctx, request.PipelineId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	err = s.resourceManager.UpdatePipelineDefaultVersion(request.PipelineId, request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Update Pipeline Default Version failed.")
	}
	return &empty.Empty{}, nil
}

func (s *PipelineServer) GetPipelineV1(ctx context.Context, request *apiv1beta1.GetPipelineRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}

	pipeline, err := s.resourceManager.GetPipeline(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline failed.")
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbGet,
	}
	err = s.CanAccessPipeline(ctx, request.Id, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	return ToApiPipeline(pipeline), nil
}

func (s *PipelineServer) GetPipelineByNameV1(ctx context.Context, request *apiv1beta1.GetPipelineByNameRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}

	//If namespace is "-" transform it to an emtpy string ""
	if request.Namespace == model.NoNamespace {
		request.Namespace = ""
	}
	if request.Namespace != "" && common.IsMultiUserMode() {
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: request.GetNamespace(),
			Verb:      common.RbacResourceVerbGet,
		}
		if err := s.haveAccess(ctx, resourceAttributes); err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API")
		}
	}

	pipeline, err := s.resourceManager.GetPipelineByNameAndNamespace(request.Name, request.GetNamespace())
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline by name failed.")
	}
	return ToApiPipeline(pipeline), nil
}

func (s *PipelineServer) ListPipelinesV1(ctx context.Context, request *apiv1beta1.ListPipelinesRequest) (*apiv1beta1.ListPipelinesResponse, error) {
	if s.options.CollectMetrics {
		listPipelineRequests.Inc()
	}

	filterContext, err := ValidateFilterV1(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed.")
	}

	/*
		Assume 3 scenarios and ensure backwards compatibility:
		1. User does not provide resource reference
		2. User provides resource reference to public namesapce ""
		3. User provides resource reference to namespace x
	*/
	refKey := filterContext.ReferenceKey
	if refKey == nil {
		filterContext = &common.FilterContext{
			ReferenceKey: &common.ReferenceKey{Type: common.Namespace, ID: ""},
		}
	}
	if refKey != nil && refKey.Type != common.Namespace {
		return nil, util.NewInvalidInputError("Invalid resource references for pipelines. ListPipelines requires filtering by namespace.")
	}
	if refKey != nil && refKey.Type == common.Namespace {
		namespace := refKey.ID
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      common.RbacResourceVerbList,
		}
		if err = s.CanAccessPipeline(ctx, "", resourceAttributes); err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API")
		}
	}

	opts, err := validatedListOptions(&model.Pipeline{}, request.PageToken, int(request.PageSize), request.SortBy, request.Filter)

	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	pipelines, total_size, nextPageToken, err := s.resourceManager.ListPipelines(filterContext, opts)
	if err != nil {
		return nil, util.Wrap(err, "List pipelines failed.")
	}
	apiPipelines := ToApiPipelines(pipelines)
	return &apiv1beta1.ListPipelinesResponse{Pipelines: apiPipelines, TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *PipelineServer) DeletePipelineV1(ctx context.Context, request *apiv1beta1.DeletePipelineRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineRequests.Inc()
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbDelete,
	}
	err := s.CanAccessPipeline(ctx, request.Id, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	err = s.resourceManager.DeletePipeline(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Delete pipelines failed.")
	}
	if s.options.CollectMetrics {
		pipelineCount.Dec()
	}

	return &empty.Empty{}, nil
}

func (s *PipelineServer) GetTemplate(ctx context.Context, request *apiv1beta1.GetTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err := s.CanAccessPipeline(ctx, request.Id, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	template, err := s.resourceManager.GetPipelineTemplate(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed.")
	}

	return &apiv1beta1.GetTemplateResponse{Template: string(template)}, nil
}

func NewPipelineServer(resourceManager *resource.ResourceManager, options *PipelineServerOptions) *PipelineServer {
	return &PipelineServer{resourceManager: resourceManager, httpClient: http.DefaultClient, options: options}
}

func (s *PipelineServer) CreatePipelineVersionV1(ctx context.Context, request *apiv1beta1.CreatePipelineVersionRequest) (*apiv1beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		createPipelineVersionRequests.Inc()
	}

	// Read pipeline file.
	if (len(pv.PipelineSpec) == 0) && (len(pv.CodeSourceUrl) == 0) {
		return nil, util.NewInvalidInputError("ResourceManager CreatePipelineVersion: Failed due to missing pipeline_spec and package_url")
	}
	if len(pv.PipelineSpec) == 0 {
		pipelineUrl := pv.CodeSourceUrl
		if _, err := url.ParseRequestURI(pipelineUrl); err != nil {
			return nil, util.NewInvalidInputError("ResourceManager CreatePipelineVersion: Invalid Pipeline URL %v. Please specify a valid URL", pipelineUrl)
		}
		resp, err := s.httpClient.Get(pipelineUrl)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, util.NewInternalServerError(err, "Failed to download the pipeline from %v. Please double check the URL is valid and can be accessed by the pipeline system.", pipelineUrl)
		}
	}
	pipelineFileName := path.Base(pipelineUrl)
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "The URL is valid but pipeline system failed to read the file.")
	}

	if request.Version == nil || request.Version.PackageUrl == nil ||
		len(request.Version.PackageUrl.PipelineUrl) == 0 {
		return nil, util.NewInvalidInputError("Pipeline URL is empty. Please specify a valid URL.")
	}
	pipelineUrl := request.Version.PackageUrl.PipelineUrl
	if _, err := url.ParseRequestURI(request.Version.PackageUrl.PipelineUrl); err != nil {
		return nil, util.NewInvalidInputError("Invalid Pipeline URL %v. Please specify a valid URL", request.Version.PackageUrl.PipelineUrl)
	}
	resp, err := s.httpClient.Get(pipelineUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, util.NewInternalServerError(err, "Failed to download the pipeline from %v. Please double check the URL is valid and can be accessed by the pipeline system.", pipelineUrl)
	}
	pipelineFileName := path.Base(pipelineUrl)
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "The URL is valid but pipeline system failed to read the file.")
	}

	// Extract pipeline id and authorize
	var pipelineId = ""
	for _, resourceReference := range request.Version.ResourceReferences {
		if resourceReference.Key.Type == apiv1beta1.ResourceType_PIPELINE && resourceReference.Relationship == apiv1beta1.Relationship_OWNER {
			pipelineId = resourceReference.Key.Id
		}
	}
	if len(pipelineId) == 0 {
		return nil, util.Wrap(err, "Create pipeline version failed due to missing pipeline id")
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	if err = s.CanAccessPipeline(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize with API")
	}

	version, err := s.resourceManager.CreatePipelineVersion(request.Version, pipelineFile, common.IsPipelineVersionUpdatedByDefault())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a version.")
	}
	return ToApiPipelineVersion(version)
}

func (s *PipelineServer) CreatePipelineVersion(ctx context.Context, request *apiv2beta1.CreatePipelineVersionRequest) (*apiv2beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		createPipelineVersionRequests.Inc()
	}

	// Read pipeline file.
	if request.Version == nil || request.Version.PackageUrl == nil ||
		len(request.Version.PackageUrl.PipelineUrl) == 0 {
		return nil, util.NewInvalidInputError("Pipeline URL is empty. Please specify a valid URL.")
	}
	pipelineUrl := request.Version.PackageUrl.PipelineUrl
	if _, err := url.ParseRequestURI(request.Version.PackageUrl.PipelineUrl); err != nil {
		return nil, util.NewInvalidInputError("Invalid Pipeline URL %v. Please specify a valid URL", request.Version.PackageUrl.PipelineUrl)
	}
	resp, err := s.httpClient.Get(pipelineUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, util.NewInternalServerError(err, "Failed to download the pipeline from %v. Please double check the URL is valid and can be accessed by the pipeline system.", pipelineUrl)
	}
	pipelineFileName := path.Base(pipelineUrl)
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "The URL is valid but pipeline system failed to read the file.")
	}

	// Extract pipeline id and authorize
	var pipelineId = ""
	for _, resourceReference := range request.Version.ResourceReferences {
		if resourceReference.Key.Type == apiv1beta1.ResourceType_PIPELINE && resourceReference.Relationship == apiv1beta1.Relationship_OWNER {
			pipelineId = resourceReference.Key.Id
		}
	}
	if len(pipelineId) == 0 {
		return nil, util.Wrap(err, "Create pipeline version failed due to missing pipeline id")
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	if err = s.CanAccessPipeline(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize with API")
	}

	version, err := s.resourceManager.CreatePipelineVersion(request.Version, pipelineFile, common.IsPipelineVersionUpdatedByDefault())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a version.")
	}
	return ToApiPipelineVersion(version)
}

func (s *PipelineServer) GetPipelineVersionV1(ctx context.Context, request *apiv1beta1.GetPipelineVersionRequest) (*apiv1beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		getPipelineVersionRequests.Inc()
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err := s.CanAccessPipelineVersion(ctx, request.VersionId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	version, err := s.resourceManager.GetPipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline version failed.")
	}
	return ToApiPipelineVersion(version)
}

func (s *PipelineServer) ListPipelineVersionsV1(ctx context.Context, request *apiv1beta1.ListPipelineVersionsRequest) (*apiv1beta1.ListPipelineVersionsResponse, error) {
	if s.options.CollectMetrics {
		listPipelineVersionRequests.Inc()
	}

	opts, err := validatedListOptions(
		&model.PipelineVersion{},
		request.PageToken,
		int(request.PageSize),
		request.SortBy,
		request.Filter)

	if err != nil {
		return nil, util.Wrap(err, "Failed to create list options")
	}

	//Ensure resourceKey has been set
	if request.ResourceKey == nil {
		return nil, util.NewInvalidInputError("ResourceKey must be set in the input")
	}

	namespace, err := s.resourceManager.GetNamespaceFromPipelineID(request.ResourceKey.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get namespace from pipelineId.")
	}
	// Only check if the namespace isn't empty.
	if common.IsMultiUserMode() && namespace != "" {
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      common.RbacResourceVerbList,
		}
		err = s.CanAccessPipelineVersion(ctx, "", resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API")
		}
	}

	pipelineVersions, totalSize, nextPageToken, err :=
		s.resourceManager.ListPipelineVersions(request.ResourceKey.Id, opts)
	if err != nil {
		return nil, util.Wrap(err, "List pipeline versions failed.")
	}
	apiPipelineVersions, _ := ToApiPipelineVersions(pipelineVersions)

	return &apiv1beta1.ListPipelineVersionsResponse{
		Versions:      apiPipelineVersions,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize)}, nil
}

func (s *PipelineServer) DeletePipelineVersionV1(ctx context.Context, request *apiv1beta1.DeletePipelineVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineVersionRequests.Inc()
	}
	pipelineVersion, err := s.resourceManager.GetPipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get Pipeline Version.")
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err = s.CanAccessPipeline(ctx, pipelineVersion.PipelineId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize with API")
	}
	err = s.resourceManager.DeletePipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Delete pipeline versions failed.")
	}

	return &empty.Empty{}, nil
}

func (s *PipelineServer) GetPipelineVersionTemplate(ctx context.Context, request *apiv1beta1.GetPipelineVersionTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err := s.CanAccessPipelineVersion(ctx, request.VersionId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	template, err := s.resourceManager.GetPipelineVersionTemplate(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed.")
	}

	return &apiv1beta1.GetTemplateResponse{Template: string(template)}, nil
}

func (s *PipelineServer) CanAccessPipelineVersion(ctx context.Context, versionId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if versionId != "" {
		namespace, err := s.resourceManager.GetNamespaceFromPipelineVersion(versionId)
		if namespace == "" {
			return nil
		}
		if err != nil {
			return util.Wrap(err, "Failed to get namespace from Pipeline VersionId")
		}
		resourceAttributes.Namespace = namespace
	}
	if resourceAttributes.Namespace == "" {
		return nil
	}
	return s.haveAccess(ctx, resourceAttributes)
}

func (s *PipelineServer) CanAccessPipeline(ctx context.Context, pipelineId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if pipelineId != "" {
		namespace, err := s.resourceManager.GetNamespaceFromPipelineID(pipelineId)
		if namespace == "" {
			return nil
		}
		if err != nil {
			return util.Wrap(err, "Failed to authorize with the Pipeline ID.")
		}
		resourceAttributes.Namespace = namespace
	}
	if resourceAttributes.Namespace == "" {
		return nil
	}

	return s.haveAccess(ctx, resourceAttributes)
}

func (s *PipelineServer) haveAccess(ctx context.Context, resourceAttributes *authorizationv1.ResourceAttributes) error {
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypePipelines
	err := isAuthorized(s.resourceManager, ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API")
	}
	return nil
}

func ValidatePipelineForCreation(p *model.Pipeline, opts *PipelineServerOptions) error {
	switch opts.ApiVersion {
	case "v1beta1":
		if p.Url == "" {
			return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s] Validating Pipeline(*v1beta1): URL is empty. Please specify a valid URL.", opts.ApiVersion))
		}
		if _, err := url.ParseRequestURI(p.Url); err != nil {
			return util.NewInvalidInputError(
				"[PipelineServer %s] Validating Pipeline(*v1beta1): Invalid URL %v. Please specify a valid URL", opts.ApiVersion, p.Url)
		}
		return nil
	case "v2beta1":
		if p.GetProtoFieldValue("display_name") == "" {
			return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s] Validating Pipeline(*v2beta1): Name is empty. Please specify a valid Name.", opts.ApiVersion))
		}
		if p.GetProtoFieldValue("namespace") == "" {
			if err := p.SetProtoFieldValue("namespace", opts.DefaultNamespace); err != nil {
				return util.Wrap(err, fmt.Sprintf("[PipelineServer %s] Validating Pipeline(*v2beta1): Namespace is empty and failed to set to a default.", opts.ApiVersion))
			}
		}
		return nil
	}
	return nil
}

// // Validates Pipeline proto for KFP v1:
// //   - Url must be non-empty and contain a valid URI.
// func ValidatePipelineProtoV1(request *apiv1beta1.CreatePipelineRequest, opts *server.PipelineServerOptions) error {
// 	if request.Pipeline.Url == nil || request.Pipeline.Url.PipelineUrl == "" {
// 		return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s] Validating Pipeline(*v1beta1): URL is empty. Please specify a valid URL.", opts.ApiVersion))
// 	}
// 	if _, err := url.ParseRequestURI(request.Pipeline.Url.PipelineUrl); err != nil {
// 		return util.NewInvalidInputError(
// 			"[PipelineServer %s] Validating Pipeline(*v1beta1): Invalid URL %v. Please specify a valid URL", opts.ApiVersion, request.Pipeline.Url.PipelineUrl)
// 	}
// 	return nil
// }

// // Validates CreatePipelineRequest for KFP v2:
// //   - display_name must be non-empty;
// //   - namespace must be set to a default value if it's empty;
// func ValidateCreatePipelineRequest(request *apiv2beta1.CreatePipelineRequest, opts *PipelineServerOptions) error {
// 	if request.Pipeline.GetDisplayName() == "" {
// 		return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s] Validating Pipeline(*v2beta1): Name is empty. Please specify a valid Name.", opts.ApiVersion))
// 	}
// 	if request.Pipeline.GetNamespace() == "" {
// 		request.Pipeline.Namespace = opts.DefaultNamespace
// 	}
// 	return nil
// }
