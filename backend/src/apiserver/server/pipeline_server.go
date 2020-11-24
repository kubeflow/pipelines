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
	"context"
	"net/http"
	"net/url"
	"path"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"

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
	CollectMetrics bool
}

type PipelineServer struct {
	resourceManager *resource.ResourceManager
	httpClient      *http.Client
	options         *PipelineServerOptions
}

func (s *PipelineServer) CreatePipeline(ctx context.Context, request *api.CreatePipelineRequest) (*api.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
	}

	if err := ValidateCreatePipelineRequest(request); err != nil {
		return nil, err
	}

	pipelineUrl := request.Pipeline.Url.PipelineUrl
	resp, err := s.httpClient.Get(pipelineUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, util.NewInternalServerError(err, "Failed to download the pipeline from %v "+
			"Please double check the URL is valid and can be accessed by the pipeline system.", pipelineUrl)
	}
	pipelineFileName := path.Base(pipelineUrl)
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "The URL is valid but pipeline system failed to read the file.")
	}

	pipelineName, err := GetPipelineName(request.Pipeline.Name, pipelineFileName)
	if err != nil {
		return nil, util.Wrap(err, "Invalid pipeline name.")
	}

	resourceReferences := request.Pipeline.GetResourceReferences()
	namespace := common.GetNamespaceFromAPIResourceReferences(resourceReferences)
	err = isAuthorized(s.resourceManager, ctx, namespace)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize with API resource references")
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

func (s *PipelineServer) UpdatePipelineDefaultVersion(ctx context.Context, request *api.UpdatePipelineDefaultVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		updatePipelineDefaultVersionRequests.Inc()
	}
	err := s.CanAccessPipeline(ctx, request.PipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	err = s.resourceManager.UpdatePipelineDefaultVersion(request.PipelineId, request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Update Pipeline Default Version failed.")
	}
	return &empty.Empty{}, nil
}

func (s *PipelineServer) GetPipeline(ctx context.Context, request *api.GetPipelineRequest) (*api.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}

	pipeline, err := s.resourceManager.GetPipeline(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline failed.")
	}
	err = s.CanAccessPipeline(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	return ToApiPipeline(pipeline), nil
}

func (s *PipelineServer) ListPipelines(ctx context.Context, request *api.ListPipelinesRequest) (*api.ListPipelinesResponse, error) {
	if s.options.CollectMetrics {
		listPipelineRequests.Inc()
	}

	filterContext, err := ValidateFilter(request.ResourceReferenceKey)
	if err != nil {
		return nil, util.Wrap(err, "Validating filter failed.")
	}

	refKey := filterContext.ReferenceKey
	if common.IsMultiUserMode() {
		if refKey == nil || refKey.Type != common.Namespace {
			return nil, util.NewInvalidInputError("Invalid resource references for pipelines. ListPipelineVersions requires filtering by namespace.")
		}
		namespace := refKey.ID
		if len(namespace) == 0 {
			return nil, util.NewInvalidInputError("Invalid resource references for pipelines. Namespace is empty.")
		}
		err = isAuthorized(s.resourceManager, ctx, namespace)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API resource references")
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
	return &api.ListPipelinesResponse{Pipelines: apiPipelines, TotalSize: int32(total_size), NextPageToken: nextPageToken}, nil
}

func (s *PipelineServer) DeletePipeline(ctx context.Context, request *api.DeletePipelineRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineRequests.Inc()
	}
	err := s.CanAccessPipeline(ctx, request.Id)
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

func (s *PipelineServer) GetTemplate(ctx context.Context, request *api.GetTemplateRequest) (*api.GetTemplateResponse, error) {
	err := s.CanAccessPipeline(ctx, request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	template, err := s.resourceManager.GetPipelineTemplate(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed.")
	}

	return &api.GetTemplateResponse{Template: string(template)}, nil
}

func ValidateCreatePipelineRequest(request *api.CreatePipelineRequest) error {
	if request.Pipeline.Url == nil || request.Pipeline.Url.PipelineUrl == "" {
		return util.NewInvalidInputError("Pipeline URL is empty. Please specify a valid URL.")
	}

	if _, err := url.ParseRequestURI(request.Pipeline.Url.PipelineUrl); err != nil {
		return util.NewInvalidInputError(
			"Invalid Pipeline URL %v. Please specify a valid URL", request.Pipeline.Url.PipelineUrl)
	}
	return nil
}

func NewPipelineServer(resourceManager *resource.ResourceManager, options *PipelineServerOptions) *PipelineServer {
	return &PipelineServer{resourceManager: resourceManager, httpClient: http.DefaultClient, options: options}
}

func (s *PipelineServer) CreatePipelineVersion(ctx context.Context, request *api.CreatePipelineVersionRequest) (*api.PipelineVersion, error) {
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
		if resourceReference.Key.Type == api.ResourceType_PIPELINE && resourceReference.Relationship == api.Relationship_OWNER {
			pipelineId = resourceReference.Key.Id
		}
	}
	if len(pipelineId) == 0 {
		return nil, util.Wrap(err, "Create pipeline version failed due to missing pipeline id")
	}
	err = s.CanAccessPipeline(ctx, pipelineId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}

	version, err := s.resourceManager.CreatePipelineVersion(request.Version, pipelineFile, common.IsPipelineVersionUpdatedByDefault())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a version.")
	}
	return ToApiPipelineVersion(version)
}

func (s *PipelineServer) GetPipelineVersion(ctx context.Context, request *api.GetPipelineVersionRequest) (*api.PipelineVersion, error) {
	if s.options.CollectMetrics {
		getPipelineVersionRequests.Inc()
	}
	err := s.CanAccessPipelineVersion(ctx, request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	version, err := s.resourceManager.GetPipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline version failed.")
	}
	return ToApiPipelineVersion(version)
}

func (s *PipelineServer) ListPipelineVersions(ctx context.Context, request *api.ListPipelineVersionsRequest) (*api.ListPipelineVersionsResponse, error) {
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
		err = isAuthorized(s.resourceManager, ctx, namespace)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API resource references")
		}
	}

	pipelineVersions, totalSize, nextPageToken, err :=
		s.resourceManager.ListPipelineVersions(request.ResourceKey.Id, opts)
	if err != nil {
		return nil, util.Wrap(err, "List pipeline versions failed.")
	}
	apiPipelineVersions, _ := ToApiPipelineVersions(pipelineVersions)

	return &api.ListPipelineVersionsResponse{
		Versions:      apiPipelineVersions,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize)}, nil
}

func (s *PipelineServer) DeletePipelineVersion(ctx context.Context, request *api.DeletePipelineVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineVersionRequests.Inc()
	}
	namespace, err := s.resourceManager.GetNamespacePipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to get namespace from Pipeline VersionId")
	}
	err = isAuthorized(s.resourceManager, ctx, namespace)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize with API resource references")
	}
	err = s.resourceManager.DeletePipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Delete pipeline versions failed.")
	}

	return &empty.Empty{}, nil
}

func (s *PipelineServer) GetPipelineVersionTemplate(ctx context.Context, request *api.GetPipelineVersionTemplateRequest) (*api.GetTemplateResponse, error) {
	err := s.CanAccessPipelineVersion(ctx, request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	template, err := s.resourceManager.GetPipelineVersionTemplate(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed.")
	}

	return &api.GetTemplateResponse{Template: string(template)}, nil
}

func (s *PipelineServer) CanAccessPipelineVersion(ctx context.Context, versionId string) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	namespace, err := s.resourceManager.GetNamespacePipelineVersion(versionId)
	if namespace == "" {
		return nil
	}
	if err != nil {
		return util.Wrap(err, "Failed to get namespace from Pipeline VersionId")
	}
	err = isAuthorized(s.resourceManager, ctx, namespace)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API resource references")
	}
	return nil
}

func (s *PipelineServer) CanAccessPipeline(ctx context.Context, pipelineId string) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	namespace, err := s.resourceManager.GetNamespaceFromPipelineID(pipelineId)
	if namespace == "" {
		return nil
	}
	if err != nil {
		return util.Wrap(err, "Failed to authorize with the Pipeline ID.")
	}
	err = isAuthorized(s.resourceManager, ctx, namespace)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API resource references")
	}
	return nil
}
