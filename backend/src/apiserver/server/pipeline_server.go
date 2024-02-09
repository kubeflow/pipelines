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
	"net/http"
	"net/url"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	authorizationv1 "k8s.io/api/authorization/v1"
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

	pipelineVersionCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pipeline_server_pipeline_version_count",
		Help: "The current number of pipeline versions in Kubeflow Pipelines instance",
	})

	updatePipelineDefaultVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_update_default_version_requests",
		Help: "The total number of UpdatePipelineDefaultVersion requests",
	})
)

type PipelineServerOptions struct {
	CollectMetrics bool `json:"collect_metrics,omitempty"`
}

type PipelineServer struct {
	resourceManager *resource.ResourceManager
	httpClient      *http.Client
	options         *PipelineServerOptions
}

// Creates a pipeline. Not exported.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) createPipeline(ctx context.Context, pipeline *model.Pipeline) (*model.Pipeline, error) {
	pipeline.Namespace = s.resourceManager.ReplaceNamespace(pipeline.Namespace)
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: pipeline.Namespace,
		Name:      pipeline.Name,
		Verb:      common.RbacResourceVerbCreate,
	}
	err := s.canAccessPipeline(ctx, "", resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a pipeline due to authorization error. Check if you have write permissions to namespace %s", pipeline.Namespace)
	}
	return s.resourceManager.CreatePipeline(pipeline)
}

// Creates a pipeline and a pipeline version in a single transaction.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) createPipelineAndPipelineVersion(ctx context.Context, pipeline *model.Pipeline, pipelineUrlStr string) (*model.Pipeline, *model.PipelineVersion, error) {
	// Resolve name and namespace
	pipelineFileName := path.Base(pipelineUrlStr)
	pipeline.Name = buildPipelineName(pipeline.Name, pipelineFileName)
	pipeline.Namespace = s.resourceManager.ReplaceNamespace(pipeline.Namespace)

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: pipeline.Namespace,
		Name:      pipeline.Name,
		Verb:      common.RbacResourceVerbCreate,
	}
	err := s.canAccessPipeline(ctx, "", resourceAttributes)
	if err != nil {
		return nil, nil, err
	}

	// Create a pipeline version with the same name and description
	pipelineVersion := &model.PipelineVersion{
		Name:            pipeline.Name,
		PipelineSpecURI: pipelineUrlStr,
		Description:     pipeline.Description,
		Status:          model.PipelineVersionCreating,
	}

	// Download and parse pipeline spec
	pipelineUrl, err := url.ParseRequestURI(pipelineUrlStr)
	if err != nil {
		return nil, nil, util.NewInvalidInputError("invalid pipeline spec URL: %v", pipelineUrlStr)
	}
	resp, err := s.httpClient.Get(pipelineUrl.String())
	if err != nil {
		return nil, nil, util.NewInternalServerError(err, "error downloading the pipeline spec from %v", pipelineUrl.String())
	} else if resp.StatusCode != http.StatusOK {
		return nil, nil, util.NewInvalidInputError("error fetching pipeline spec from %v - request returned %v", pipelineUrl.String(), resp.Status)
	}
	defer resp.Body.Close()
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
	if err != nil {
		return nil, nil, err
	}
	pipelineVersion.PipelineSpec = string(pipelineFile)

	// Validate the pipeline version
	if err := s.validatePipelineVersionBeforeCreating(pipelineVersion); err != nil {
		return nil, nil, err
	}

	// Create both pipeline and pipeline version is a single transaction
	return s.resourceManager.CreatePipelineAndPipelineVersion(pipeline, pipelineVersion)
}

// Creates a pipeline and a pipeline version in a single transaction.
// Supports v1beta1 behavior.
func (s *PipelineServer) CreatePipelineV1(ctx context.Context, request *apiv1beta1.CreatePipelineRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
		createPipelineVersionRequests.Inc()
	}

	// Convert the input request
	pipeline, err := toModelPipeline(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline (v1beta1) due to pipeline conversion error")
	}

	// Create both pipeline and pipeline version is a single transaction
	newPipeline, newPipelineVersion, err := s.createPipelineAndPipelineVersion(ctx, pipeline, request.GetPipeline().GetUrl().GetPipelineUrl())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline (v1beta1)")
	}

	if s.options.CollectMetrics {
		pipelineCount.Inc()
		pipelineVersionCount.Inc()
	}
	return toApiPipelineV1(newPipeline, newPipelineVersion), nil
}

// Creates a pipeline, but does not create a pipeline version.
// Supports v2beta1 behavior.
func (s *PipelineServer) CreatePipeline(ctx context.Context, request *apiv2beta1.CreatePipelineRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
	}

	// Convert the input request. Fail fast if pipeline is corrupted.
	pipeline, err := toModelPipeline(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline as pipeline conversion failed")
	}

	// Create pipeline
	createdPipeline, err := s.createPipeline(ctx, pipeline)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline due to server error")
	}

	if s.options.CollectMetrics {
		pipelineCount.Inc()
	}
	return toApiPipeline(createdPipeline), nil
}

// TODO(gkcalat): consider removing as default version is deprecated. This requires changes to v1beta1 proto.
// Updates default pipeline version for a given pipeline.
// Supports v1beta1 behavior.
func (s *PipelineServer) UpdatePipelineDefaultVersionV1(ctx context.Context, request *apiv1beta1.UpdatePipelineDefaultVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		updatePipelineDefaultVersionRequests.Inc()
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbUpdate,
	}
	err := s.canAccessPipeline(ctx, request.GetPipelineId(), resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to update (v1beta1) default pipeline version to %s for pipeline %s due to authorization error", request.GetVersionId(), request.GetPipelineId())
	}
	err = s.resourceManager.UpdatePipelineDefaultVersion(request.GetPipelineId(), request.GetVersionId())
	if err != nil {
		return nil, util.Wrapf(err, "Failed to update (v1beta1) default pipeline version to %s for pipeline %s. Check error stack", request.GetVersionId(), request.GetPipelineId())
	}
	return &empty.Empty{}, nil
}

// Fetches a pipeline.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) getPipeline(ctx context.Context, pipelineId string) (*model.Pipeline, error) {
	if pipelineId == "" {
		return nil, util.NewInvalidInputError("Failed to get a pipeline. Pipeline id cannot be empty")
	}
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbGet,
	}
	if err := s.canAccessPipeline(ctx, pipelineId, resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline due authorization error for pipeline id %v", pipelineId)
	}

	return s.resourceManager.GetPipeline(pipelineId)
}

// Returns a pipeline.
// Note, the default pipeline version will be set to be the latest pipeline version.
// Supports v1beta behavior.
func (s *PipelineServer) GetPipelineV1(ctx context.Context, request *apiv1beta1.GetPipelineRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}
	pipelineId := request.GetId()
	pipeline, err := s.getPipeline(ctx, pipelineId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline (v1beta1) %s. Check error stack", pipelineId)
	}

	pipelineVersion, err := s.getLatestPipelineVersion(ctx, pipelineId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline (v1beta1) %s due to error fetching the latest pipeline version", pipelineId)
	}

	return toApiPipelineV1(pipeline, pipelineVersion), nil
}

// Returns a pipeline.
// Supports v2beta behavior.
func (s *PipelineServer) GetPipeline(ctx context.Context, request *apiv2beta1.GetPipelineRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}
	pipelineId := request.GetPipelineId()
	pipeline, err := s.getPipeline(ctx, pipelineId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline %s. Check error stack", pipelineId)
	}
	return toApiPipeline(pipeline), nil
}

// Fetches pipeline and (optionally) pipeline version for a given name and namespace.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) getPipelineByName(ctx context.Context, name string, namespace string, apiRequestVersion string) (*model.Pipeline, *model.PipelineVersion, error) {
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Name:      name,
		Verb:      common.RbacResourceVerbGet,
	}
	if err := s.canAccessPipeline(ctx, "", resourceAttributes); err != nil {
		return nil, nil, util.Wrapf(err, "Failed to fetch a pipeline due to authorization error. Check if you have read permission to namespace %v", namespace)
	}
	switch apiRequestVersion {
	case "v1beta1":
		return s.resourceManager.GetPipelineByNameAndNamespaceV1(name, namespace)
	case "v2beta1":
		p, err := s.resourceManager.GetPipelineByNameAndNamespace(name, namespace)
		return p, nil, err
	default:
		return nil, nil, util.NewInternalServerError(
			util.NewInvalidInputError("Invalid api version detected"),
			"Failed to get a pipeline by name and namespace. API request version %v",
			apiRequestVersion)
	}
}

// Returns a pipeline with the default (latest) pipeline version given a name and a namespace.
// Supports v1beta behavior.
func (s *PipelineServer) GetPipelineByNameV1(ctx context.Context, request *apiv1beta1.GetPipelineByNameRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}

	namespace := request.GetNamespace()
	name := request.GetName()

	pipeline, pipelineVersion, err := s.getPipelineByName(ctx, name, namespace, "v1beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline (v1beta1) with name %s and namespace %s. Check error stack.", name, namespace)
	}
	return toApiPipelineV1(pipeline, pipelineVersion), nil
}

// Returns a pipeline given name and namespace.
// Supports v2beta behavior.
func (s *PipelineServer) GetPipelineByName(ctx context.Context, request *apiv2beta1.GetPipelineByNameRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}

	namespace := request.GetNamespace()
	name := request.GetName()

	pipeline, _, err := s.getPipelineByName(ctx, name, namespace, "v2beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline with name %s and namespace %s. Check error stack.", name, namespace)
	}
	return toApiPipeline(pipeline), nil
}

// Fetches an array of pipelines and an array of pipeline versions for given search query parameters.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) listPipelines(ctx context.Context, namespace string, pageToken string, pageSize int32, sortBy string, opts *list.Options, apiRequestVersion string) ([]*model.Pipeline, []*model.PipelineVersion, int, string, error) {
	// Fill in the default namespace
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	if err := s.canAccessPipeline(ctx, "", resourceAttributes); err != nil {
		return nil, nil, 0, "", util.Wrapf(err, "Failed to list pipelines due to authorization error. Check if you have read permission to namespace %v", namespace)
	}
	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace},
	}

	// List pipelines
	switch apiRequestVersion {
	case "v1beta1":
		return s.resourceManager.ListPipelinesV1(filterContext, opts)
	case "v2beta1":
		pipelines, size, token, err := s.resourceManager.ListPipelines(filterContext, opts)
		return pipelines, nil, size, token, err
	default:
		return nil, nil, 0, "", util.NewInternalServerError(
			util.NewInvalidInputError("Invalid api version detected"),
			"Failed to list pipelines due to unsupported API request. API request version %v",
			apiRequestVersion)
	}
}

// Returns pipelines with default pipeline versions for a given query.
// Supports v1beta behavior.
func (s *PipelineServer) ListPipelinesV1(ctx context.Context, request *apiv1beta1.ListPipelinesRequest) (*apiv1beta1.ListPipelinesResponse, error) {
	if s.options.CollectMetrics {
		listPipelineRequests.Inc()
	}
	/*
		Override namespace to support v1beta behavior
		Assume 3 scenarios and ensure backwards compatibility:
		1. User does not provide resource reference
		2. User provides resource reference to public namespace ""
		3. User provides resource reference to namespace
	*/
	namespace := ""
	filterContext, err := validateFilterV1(request.GetResourceReferenceKey())
	if err != nil {
		return nil, util.Wrap(err, "Failed to list pipelines (v1beta1) due to filter validation error")
	}
	refKey := filterContext.ReferenceKey
	if refKey != nil && refKey.Type != model.NamespaceResourceType {
		return nil, util.NewInvalidInputError("Failed to list pipelines (v1beta1) due to invalid resource references for pipelines: %v", refKey)
	}
	if refKey != nil && refKey.Type == model.NamespaceResourceType {
		namespace = refKey.ID
	}

	pageToken := request.GetPageToken()
	pageSize := request.GetPageSize()
	sortBy := request.GetSortBy()
	filter := request.GetFilter()

	// Validate list options
	opts, err := validatedListOptions(&model.Pipeline{}, pageToken, int(pageSize), sortBy, filter, "v1beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, filter)
	}

	pipelines, pipelineVersions, totalSize, nextPageToken, err := s.listPipelines(ctx, namespace, pageToken, pageSize, sortBy, opts, "v1beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines (v1beta1) in namespace %s. Check error stack", namespace)
	}

	apiPipelines := toApiPipelinesV1(pipelines, pipelineVersions)
	return &apiv1beta1.ListPipelinesResponse{Pipelines: apiPipelines, TotalSize: int32(totalSize), NextPageToken: nextPageToken}, nil
}

// Returns pipelines for a given query.
// Supports v2beta1 behavior.
func (s *PipelineServer) ListPipelines(ctx context.Context, request *apiv2beta1.ListPipelinesRequest) (*apiv2beta1.ListPipelinesResponse, error) {
	if s.options.CollectMetrics {
		listPipelineRequests.Inc()
	}

	namespace := request.GetNamespace()
	pageToken := request.GetPageToken()
	pageSize := request.GetPageSize()
	sortBy := request.GetSortBy()
	filter := request.GetFilter()

	// Validate list options
	opts, err := validatedListOptions(&model.Pipeline{}, pageToken, int(pageSize), sortBy, filter, "v2beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, filter)
	}

	pipelines, _, totalSize, nextPageToken, err := s.listPipelines(ctx, namespace, pageToken, pageSize, sortBy, opts, "v2beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines in namespace %s. Check error stack", namespace)
	}
	return &apiv2beta1.ListPipelinesResponse{Pipelines: toApiPipelines(pipelines), TotalSize: int32(totalSize), NextPageToken: nextPageToken}, nil
}

// Removes a pipeline.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) deletePipeline(ctx context.Context, pipelineId string) error {
	// Fail fast
	if pipelineId == "" {
		return util.NewInvalidInputError("Failed to delete a pipeline due missing pipeline id")
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbDelete,
	}
	err := s.canAccessPipeline(ctx, pipelineId, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to delete a pipeline due authorization error for pipeline id %v", pipelineId)
	}

	return s.resourceManager.DeletePipeline(pipelineId)
}

// Deletes a pipeline.
// Supports v1beta1 behavior.
func (s *PipelineServer) DeletePipelineV1(ctx context.Context, request *apiv1beta1.DeletePipelineRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineRequests.Inc()
	}

	if err := s.deletePipeline(ctx, request.GetId()); err != nil {
		return nil, util.Wrapf(err, "Failed to delete pipeline (v1beta1) %s. Check error stack", request.GetId())
	}

	if s.options.CollectMetrics {
		pipelineCount.Dec()
	}

	return &empty.Empty{}, nil
}

// Deletes a pipeline.
// Supports v2beta1 behavior.
func (s *PipelineServer) DeletePipeline(ctx context.Context, request *apiv2beta1.DeletePipelineRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineRequests.Inc()
	}

	if err := s.deletePipeline(ctx, request.GetPipelineId()); err != nil {
		return nil, util.Wrapf(err, "Failed to delete pipeline %s. Check error stack", request.GetPipelineId())
	}

	if s.options.CollectMetrics {
		pipelineCount.Dec()
	}

	return &empty.Empty{}, nil
}

// Returns the default (latest) pipeline template for a given pipeline id.
// Supports v1beta1 behavior.
func (s *PipelineServer) GetTemplate(ctx context.Context, request *apiv1beta1.GetTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	pipelineId := request.GetId()
	if pipelineId == "" {
		return nil, util.NewInvalidInputError("Failed to get the default pipeline template (v1beta1). Pipeline id cannot be empty")
	}
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbGet,
	}
	err := s.canAccessPipeline(ctx, pipelineId, resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get the default template (v1beta1) due to authorization error. Verify that you have access to pipeline id %s", pipelineId)
	}
	template, err := s.resourceManager.GetPipelineLatestTemplate(pipelineId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get the default template (v1beta1) for pipeline %s. Check error stack", pipelineId)
	}
	return &apiv1beta1.GetTemplateResponse{Template: string(template)}, nil
}

// Fetches the latest pipeline version for a given pipeline id.
func (s *PipelineServer) getLatestPipelineVersion(ctx context.Context, pipelineId string) (*model.PipelineVersion, error) {
	if pipelineId == "" {
		return nil, util.NewInvalidInputError("Failed to get the latest pipeline version as pipeline id is empty")
	}
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbGet,
	}
	if err := s.canAccessPipeline(ctx, pipelineId, resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to get the latest pipeline version due authorization error for pipeline id %v", pipelineId)
	}
	return s.resourceManager.GetLatestPipelineVersion(pipelineId)
}

// Validates a pipeline version before creating a record in the DB.
// Requires Name and PipelineId to be non-empty and presence of PipelineSpec or a valid URI to the pipeline spec.
func (s *PipelineServer) validatePipelineVersionBeforeCreating(p *model.PipelineVersion) error {
	if p.PipelineSpec != "" {
		return nil
	}
	if p.PipelineSpecURI != "" {
		if _, err := url.ParseRequestURI(p.PipelineSpecURI); err == nil {
			return nil
		}
	}
	if p.CodeSourceUrl != "" {
		if _, err := url.ParseRequestURI(p.CodeSourceUrl); err == nil {
			return nil
		}
	}
	return util.NewInvalidInputError("Pipeline version must have a pipeline spec or a valid source code's URL. PipelineSpec: %s. PipelineSpecURI: %s. CodeSourceUrl: %s. At least one of them must have a valid pipeline spec", p.PipelineSpec, p.PipelineSpecURI, p.CodeSourceUrl)
}

func NewPipelineServer(resourceManager *resource.ResourceManager, options *PipelineServerOptions) *PipelineServer {
	return &PipelineServer{resourceManager: resourceManager, httpClient: http.DefaultClient, options: options}
}

// Creates a pipeline and a pipeline version in a single transaction.
// Supports v2beta1 behavior.
func (s *PipelineServer) CreatePipelineAndVersion(ctx context.Context, request *apiv2beta1.CreatePipelineAndVersionRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
		createPipelineVersionRequests.Inc()
	}

	// Convert the input request
	pipeline, err := toModelPipeline(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline due to pipeline conversion error")
	}

	// Create both pipeline and pipeline version in a single transaction
	newPipeline, _, err := s.createPipelineAndPipelineVersion(ctx, pipeline, request.GetPipelineVersion().GetPackageUrl().GetPipelineUrl())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline")
	}

	if s.options.CollectMetrics {
		pipelineCount.Inc()
		pipelineVersionCount.Inc()
	}
	return toApiPipeline(newPipeline), nil
}

// Creates a pipeline version from. Not exported.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) createPipelineVersion(ctx context.Context, pv *model.PipelineVersion) (*model.PipelineVersion, error) {
	// Fail if pipeline URL is missing
	if pv.PipelineSpecURI == "" {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due to missing pipeline URL")
	}

	// Fail if parent pipeline id is missing
	if pv.PipelineId == "" {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due empty parent pipeline id")
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Name: pv.Name,
		Verb: common.RbacResourceVerbCreate,
	}
	if err := s.canAccessPipeline(ctx, pv.PipelineId, resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to create a pipeline version due authorization error for pipeline id %v", pv.PipelineId)
	}

	// Read pipeline file
	pipelineUrl, err := url.ParseRequestURI(pv.PipelineSpecURI)
	if err != nil {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due to invalid pipeline spec URI. PipelineSpecURI: %v. Please specify a valid URL", pv.PipelineSpecURI)
	}

	resp, err := s.httpClient.Get(pipelineUrl.String())
	if err != nil || resp.StatusCode != http.StatusOK {
		if err == nil {
			return nil, util.NewInvalidInputError("Failed to fetch pipeline spec with url: %v. Request returned %v", pipelineUrl.String(), resp.Status)
		}
		return nil, util.NewInternalServerError(err, "Failed to create a pipeline version due error downloading the pipeline spec from %v", pipelineUrl.String())
	}
	defer resp.Body.Close()
	pipelineFileName := path.Base(pipelineUrl.String())
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version due error reading the pipeline spec")
	}
	pv.PipelineSpec = string(pipelineFile)
	if pv.Name == "" {
		pv.Name = pipelineFileName
	}

	// Validate the pipeline version
	if err := s.validatePipelineVersionBeforeCreating(pv); err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version due to data validation error. Check the error stack")
	}

	return s.resourceManager.CreatePipelineVersion(pv)
}

// Creates a pipeline version.
// Supports v1beta behavior.
func (s *PipelineServer) CreatePipelineVersionV1(ctx context.Context, request *apiv1beta1.CreatePipelineVersionRequest) (*apiv1beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		createPipelineVersionRequests.Inc()
	}

	// Fail fast
	if request.Version == nil || request.Version.PackageUrl == nil ||
		len(request.Version.PackageUrl.PipelineUrl) == 0 {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version (v1beta1). Pipeline version is nil or pipeline's URL is empty")
	}

	// Convert to pipeline
	pv, err := toModelPipelineVersion(request.GetVersion())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version (v1beta1) due to conversion error")
	}

	// Extract pipeline id
	pipelineId := ""
	for _, resourceReference := range request.Version.ResourceReferences {
		if resourceReference.Key.Type == apiv1beta1.ResourceType_PIPELINE && resourceReference.Relationship == apiv1beta1.Relationship_OWNER {
			pipelineId = resourceReference.Key.Id
		}
	}
	if len(pipelineId) == 0 {
		return nil, util.Wrap(err, "Failed to create a pipeline version (v1beta1) due to missing pipeline id")
	}
	pv.PipelineId = pipelineId

	// Create a pipeline version
	newpv, err := s.createPipelineVersion(ctx, pv)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version (v1beta1) due to internal server error")
	}
	if s.options.CollectMetrics {
		pipelineVersionCount.Inc()
	}

	// Convert back to API
	// Note, v1beta1 PipelineVersion does not have error message. Errors in converting to API will result in error.
	result := toApiPipelineVersionV1(newpv)
	if result == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Failed to convert internal pipeline version representation to its v1beta1 API counterpart"), "Failed to create a pipeline version (v1beta1) due to error converting back to API")
	}
	return result, nil
}

// Creates a pipeline version.
// Supports v2beta1 behavior.
func (s *PipelineServer) CreatePipelineVersion(ctx context.Context, request *apiv2beta1.CreatePipelineVersionRequest) (*apiv2beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		createPipelineVersionRequests.Inc()
	}

	// Fail fast
	if request.GetPipelineVersion() == nil {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version. Pipeline version is nil")
	} else if request.GetPipelineVersion().GetPackageUrl() == nil {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version. Package URL is nil")
	} else if request.GetPipelineVersion().GetPackageUrl().GetPipelineUrl() == "" {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version. Package URL is empty")
	} else if request.GetPipelineId() == "" || request.GetPipelineVersion().GetPipelineId() == "" {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version. Parent pipeline id is empty")
	}

	// Convert to pipeline
	pv, err := toModelPipelineVersion(request.GetPipelineVersion())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version due to conversion error")
	}

	// Extract pipeline id
	if request.GetPipelineId() != "" {
		pv.PipelineId = request.GetPipelineId()
	}
	if pv.PipelineId == "" {
		return nil, util.Wrap(err, "Failed to create a pipeline version due to missing pipeline id")
	}

	newPipelineVersion, err := s.createPipelineVersion(ctx, pv)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version. Check error stack")
	}

	if s.options.CollectMetrics {
		pipelineVersionCount.Inc()
	}
	return toApiPipelineVersion(newPipelineVersion), nil
}

// Fetches a pipeline version for given pipeline id.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) getPipelineVersion(ctx context.Context, pipelineVersionId string) (*model.PipelineVersion, error) {
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbGet,
	}
	err := s.canAccessPipelineVersion(ctx, pipelineVersionId, resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline version due to authorization error for pipeline version id %v", pipelineVersionId)
	}
	return s.resourceManager.GetPipelineVersion(pipelineVersionId)
}

// Returns a pipeline version.
// Supports v1beta behavior.
func (s *PipelineServer) GetPipelineVersionV1(ctx context.Context, request *apiv1beta1.GetPipelineVersionRequest) (*apiv1beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		getPipelineVersionRequests.Inc()
	}

	pipelineVersion, err := s.getPipelineVersion(ctx, request.GetVersionId())
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline version (v1beta1) %s", request.GetVersionId())
	}
	apiPipelineVersion := toApiPipelineVersionV1(pipelineVersion)
	if apiPipelineVersion == nil {
		return nil, util.NewInternalServerError(util.NewInvalidInputError("Pipeline version cannot be converted to v1beta1 API"), "Failed to get a pipeline version (v1beta1) due to error converting to API counterpart")
	}
	return apiPipelineVersion, nil
}

// Returns a pipeline version.
// Supports v2beta1 behavior.
func (s *PipelineServer) GetPipelineVersion(ctx context.Context, request *apiv2beta1.GetPipelineVersionRequest) (*apiv2beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		getPipelineVersionRequests.Inc()
	}

	pipelineVersion, err := s.getPipelineVersion(ctx, request.GetPipelineVersionId())
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline version %s", request.GetPipelineVersionId())
	}
	return toApiPipelineVersion(pipelineVersion), nil
}

// Fetches an array of pipeline versions for given search query parameters.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) listPipelineVersions(ctx context.Context, pipelineId string, pageToken string, pageSize int32, sortBy string, opts *list.Options) ([]*model.PipelineVersion, int, string, error) {
	// Fail fast of pipeline id or namespace are missing
	if pipelineId == "" {
		return nil, 0, "", util.NewInvalidInputError("Failed to list pipeline versions. Pipeline id cannot be empty")
	}

	// Check authorization
	namespace, err := s.resourceManager.FetchNamespaceFromPipelineId(pipelineId)
	if err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list pipeline versions due to error fetching the namespace for pipeline %v", pipelineId)
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	if err := s.canAccessPipelineVersion(ctx, "", resourceAttributes); err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list pipeline versions due to authorization error. Check if you have read permission to namespace %v", namespace)
	}

	// Get pipeline versions
	return s.resourceManager.ListPipelineVersions(pipelineId, opts)
}

// Returns an array of pipeline versions for a given query.
// Supports v1beta1 behavior.
func (s *PipelineServer) ListPipelineVersionsV1(ctx context.Context, request *apiv1beta1.ListPipelineVersionsRequest) (*apiv1beta1.ListPipelineVersionsResponse, error) {
	if s.options.CollectMetrics {
		listPipelineVersionRequests.Inc()
	}

	// Check if parent pipeline id is present in the request
	if request.ResourceKey == nil {
		return nil, util.NewInvalidInputError("Failed to list pipeline versions (v1beta1) due to missing pipeline id")
	}
	pipelineId := request.GetResourceKey().GetId()
	pageToken := request.GetPageToken()
	pageSize := request.GetPageSize()
	sortBy := request.GetSortBy()
	filter := request.GetFilter()

	// Validate query parameters
	opts, err := validatedListOptions(&model.PipelineVersion{}, pageToken, int(pageSize), sortBy, filter, "v1beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, filter)
	}

	pipelineVersions, totalSize, nextPageToken, err := s.listPipelineVersions(ctx, pipelineId, pageToken, pageSize, sortBy, opts)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions (v1beta1) with pipeline id %s. Check error stack", pipelineId)
	}
	apiPipelineVersions := toApiPipelineVersionsV1(pipelineVersions)
	if apiPipelineVersions == nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions (v1beta1) with pipeline id %s due to conversion error", pipelineId)
	}
	return &apiv1beta1.ListPipelineVersionsResponse{
		Versions:      apiPipelineVersions,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}, nil
}

// Returns an array of pipeline versions for a given query.
// Supports v2beta1 behavior.
func (s *PipelineServer) ListPipelineVersions(ctx context.Context, request *apiv2beta1.ListPipelineVersionsRequest) (*apiv2beta1.ListPipelineVersionsResponse, error) {
	if s.options.CollectMetrics {
		listPipelineVersionRequests.Inc()
	}

	// Check if parent pipeline id is present in the request
	pipelineId := request.GetPipelineId()
	pageToken := request.GetPageToken()
	pageSize := request.GetPageSize()
	sortBy := request.GetSortBy()
	filter := request.GetFilter()

	// Validate query parameters
	opts, err := validatedListOptions(&model.PipelineVersion{}, pageToken, int(pageSize), sortBy, filter, "v2beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, filter)
	}

	pipelineVersions, totalSize, nextPageToken, err := s.listPipelineVersions(ctx, pipelineId, pageToken, pageSize, sortBy, opts)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions for pipeline %s", pipelineId)
	}
	return &apiv2beta1.ListPipelineVersionsResponse{
		PipelineVersions: toApiPipelineVersions(pipelineVersions),
		NextPageToken:    nextPageToken,
		TotalSize:        int32(totalSize),
	}, nil
}

// Removes a pipeline version.
// Applies common logic on v1beta1 and v2beta1 API.
func (s *PipelineServer) deletePipelineVersion(ctx context.Context, pipelineId string, pipelineVersionId string) error {
	// Fail fast
	if pipelineId == "" {
		return util.NewInvalidInputError("Failed to delete a pipeline version id %v due missing pipeline id", pipelineVersionId)
	}
	if pipelineVersionId == "" {
		return util.NewInvalidInputError("Failed to delete a pipeline version due missing pipeline version id")
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbDelete,
	}
	err := s.canAccessPipelineVersion(ctx, pipelineVersionId, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to delete a pipeline version id %v due to authorization error for pipeline id %v", pipelineVersionId, pipelineId)
	}

	return s.resourceManager.DeletePipelineVersion(pipelineVersionId)
}

// Deletes a pipeline version.
// Supports v1beta1 behavior.
func (s *PipelineServer) DeletePipelineVersionV1(ctx context.Context, request *apiv1beta1.DeletePipelineVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineVersionRequests.Inc()
	}

	pipelineVersionId := request.GetVersionId()
	if pipelineVersionId == "" {
		return nil, util.NewInvalidInputError("Failed to delete a pipeline version (v1beta1) due missing pipeline version id")
	}

	pipelineVersion, err := s.resourceManager.GetPipelineVersion(pipelineVersionId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to delete a pipeline version (v1beta1) %s due error fetching a pipeline id", pipelineVersionId)
	}

	pipelineId := pipelineVersion.PipelineId

	err = s.deletePipelineVersion(ctx, pipelineId, pipelineVersionId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to delete a pipeline version (v1beta1) %v under pipeline %v. Check error stack", pipelineVersionId, pipelineId)
	}

	if s.options.CollectMetrics {
		pipelineVersionCount.Dec()
	}
	return &empty.Empty{}, nil
}

// Deletes a pipeline version.
// Supports v2beta1 behavior.
func (s *PipelineServer) DeletePipelineVersion(ctx context.Context, request *apiv2beta1.DeletePipelineVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineVersionRequests.Inc()
	}

	pipelineVersionId := request.GetPipelineVersionId()
	if pipelineVersionId == "" {
		return nil, util.NewInvalidInputError("Failed to delete a pipeline version due missing pipeline version id")
	}

	pipelineId := request.GetPipelineId()
	if pipelineId == "" {
		return nil, util.NewInvalidInputError("Failed to delete a pipeline version %s due missing pipeline id", pipelineVersionId)
	}

	err := s.deletePipelineVersion(ctx, pipelineId, pipelineVersionId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to delete a pipeline version id %v under pipeline id %v. Check error stack", pipelineVersionId, pipelineId)
	}

	if s.options.CollectMetrics {
		pipelineVersionCount.Dec()
	}
	return &empty.Empty{}, nil
}

// Returns pipeline template.
// Supports v1beta1 behavior.
func (s *PipelineServer) GetPipelineVersionTemplate(ctx context.Context, request *apiv1beta1.GetPipelineVersionTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbGet,
	}
	err := s.canAccessPipelineVersion(ctx, request.GetVersionId(), resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline template due to authorization error. Verify that you have access to pipeline version %s", request.GetVersionId())
	}
	template, err := s.resourceManager.GetPipelineVersionTemplate(request.VersionId)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline template for pipeline version %s", request.GetVersionId())
	}

	return &apiv1beta1.GetTemplateResponse{Template: string(template)}, nil
}

// Checks if a user can access a pipeline version.
// Adds namespace of the parent pipeline if version id is not empty,
// API group, version, and resource type.
func (s *PipelineServer) canAccessPipelineVersion(ctx context.Context, versionId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	pipelineId := ""
	if versionId != "" {
		pipelineVersion, err := s.resourceManager.GetPipelineVersion(versionId)
		if err != nil {
			return util.Wrapf(err, "Failed to access pipeline version %s. Check if it exists", versionId)
		}
		pipelineId = pipelineVersion.PipelineId
	}
	return s.canAccessPipeline(ctx, pipelineId, resourceAttributes)
}

// Checks if a user can access a pipeline.
// Adds parent namespace if pipeline id is not empty,
// API group, version, and resource type.
func (s *PipelineServer) canAccessPipeline(ctx context.Context, pipelineId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if pipelineId != "" {
		pipeline, err := s.resourceManager.GetPipeline(pipelineId)
		if err != nil {
			return util.Wrapf(err, "Failed to access pipeline %s. Check if it exists and have a namespace assigned", pipelineId)
		}
		resourceAttributes.Namespace = pipeline.Namespace
		if resourceAttributes.Name == "" {
			resourceAttributes.Name = pipeline.Name
		}
	}
	// Skip authentication if the namespace is empty to enable shared pipelines in multi-user mode
	if s.resourceManager.IsEmptyNamespace(resourceAttributes.Namespace) {
		return nil
	}
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypePipelines
	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to access pipeline %s. Check if you have access to namespace %s", pipelineId, resourceAttributes.Namespace)
	}
	return nil
}

// This method extract the common logic of naming the pipeline.
// API caller can either explicitly name the pipeline through query string ?name=foobar.
// or API server can use the file name by default.
func buildPipelineName(pipelineName string, fileName string) string {
	if pipelineName == "" {
		pipelineName = fileName
	}
	return pipelineName
}
