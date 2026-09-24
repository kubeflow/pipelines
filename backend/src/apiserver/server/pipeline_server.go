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
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/golang/glog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
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

	updatePipelineRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_update_requests",
		Help: "The total number of UpdatePipeline requests",
	})

	updatePipelineVersionRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pipeline_server_update_version_requests",
		Help: "The total number of UpdatePipelineVersion requests",
	})
)

type PipelineServerOptions struct {
	CollectMetrics bool `json:"collect_metrics,omitempty"`
}

type BasePipelineServer struct {
	resourceManager *resource.ResourceManager
	httpClient      *http.Client
	options         *PipelineServerOptions
}

type PipelineServer struct {
	*BasePipelineServer
	apiv2beta1.UnimplementedPipelineServiceServer
}

// Creates a pipeline. Not exported.
func (s *BasePipelineServer) createPipeline(ctx context.Context, pipeline *model.Pipeline) (*model.Pipeline, error) {
	pipeline.Namespace = s.resourceManager.ReplaceNamespace(pipeline.Namespace)
	err := validation.ValidateNamespaceRequired(pipeline.Namespace)
	if err != nil {
		return nil, err
	}

	if pipeline.Name == "" {
		return nil, util.NewInvalidInputError("name is required")
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: pipeline.Namespace,
		Name:      pipeline.Name,
		Verb:      common.RbacResourceVerbCreate,
	}
	err = s.canAccessPipeline(ctx, "", resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a pipeline due to authorization error. Check if you have write permissions to namespace %s", pipeline.Namespace)
	}
	return s.resourceManager.CreatePipeline(pipeline)
}

// Creates a pipeline and a pipeline version in a single transaction.
func (s *BasePipelineServer) createPipelineAndPipelineVersion(ctx context.Context, pipeline *model.Pipeline, pipelineURLStr string, versionTags map[string]string) (*model.Pipeline, *model.PipelineVersion, error) {
	// Resolve name and namespace
	pipelineFileName := path.Base(pipelineURLStr)

	pipeline.Name = buildPipelineName(pipeline.Name, pipeline.DisplayName, pipelineFileName)
	if pipeline.DisplayName == "" {
		pipeline.DisplayName = pipeline.Name
	}

	pipeline.Namespace = s.resourceManager.ReplaceNamespace(pipeline.Namespace)
	err := validation.ValidateNamespaceRequired(pipeline.Namespace)
	if err != nil {
		return nil, nil, err
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: pipeline.Namespace,
		Name:      pipeline.Name,
		Verb:      common.RbacResourceVerbCreate,
	}
	err = s.canAccessPipeline(ctx, "", resourceAttributes)
	if err != nil {
		return nil, nil, err
	}

	// Create a pipeline version with the same name and description
	pipelineVersion := &model.PipelineVersion{
		Name:            pipeline.Name,
		DisplayName:     pipeline.DisplayName,
		PipelineSpecURI: model.LargeText(pipelineURLStr),
		Description:     pipeline.Description,
		Status:          model.PipelineVersionCreating,
		Tags:            versionTags,
	}

	// Download and parse pipeline spec
	pipelineURL, err := url.ParseRequestURI(pipelineURLStr)
	if err != nil {
		return nil, nil, util.NewInvalidInputError("invalid pipeline spec URL: %v", pipelineURLStr)
	}

	if err := validation.ValidatePipelineURL(pipelineURL.String()); err != nil {
		glog.Warningf("Pipeline URL validation failed: %v", err)
		return nil, nil, util.NewInvalidInputError("Pipeline URL validation failed")
	}

	resp, err := s.httpClient.Get(pipelineURL.String())
	if err != nil {
		// Unwrap redirect validation errors so they return 4xx, not 500
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if userErr, ok := urlErr.Err.(*util.UserError); ok {
				return nil, nil, userErr
			}
		}
		return nil, nil, util.NewInternalServerError(err, "error downloading the pipeline spec from %v", pipelineURL.String())
	} else if resp.StatusCode != http.StatusOK {
		return nil, nil, util.NewInvalidInputError("error fetching pipeline spec from %v - request returned %v", pipelineURL.String(), resp.Status)
	}
	defer resp.Body.Close()
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
	if err != nil {
		return nil, nil, err
	}
	pipelineVersion.PipelineSpec = model.LargeText(pipelineFile)

	// Validate the pipeline version
	if err := s.validatePipelineVersionBeforeCreating(pipelineVersion); err != nil {
		return nil, nil, err
	}

	// Create both pipeline and pipeline version is a single transaction
	return s.resourceManager.CreatePipelineAndPipelineVersion(pipeline, pipelineVersion)
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

// Fetches a pipeline.
func (s *BasePipelineServer) getPipeline(ctx context.Context, pipelineId string) (*model.Pipeline, error) {
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

// Fetches a pipeline for a given name and namespace.
func (s *BasePipelineServer) getPipelineByName(ctx context.Context, name string, namespace string) (*model.Pipeline, error) {
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	if err := validation.ValidateNamespaceRequired(namespace); err != nil {
		return nil, err
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Name:      name,
		Verb:      common.RbacResourceVerbGet,
	}
	if err := s.canAccessPipeline(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to fetch a pipeline due to authorization error. Check if you have read permission to namespace %v", namespace)
	}
	return s.resourceManager.GetPipelineByNameAndNamespace(name, namespace)
}

// Returns a pipeline given name and namespace.
// Supports v2beta behavior.
func (s *PipelineServer) GetPipelineByName(ctx context.Context, request *apiv2beta1.GetPipelineByNameRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		getPipelineRequests.Inc()
	}

	namespace := request.GetNamespace()
	name := request.GetName()

	pipeline, err := s.getPipelineByName(ctx, name, namespace)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline with name %s and namespace %s. Check error stack.", name, namespace)
	}
	return toApiPipeline(pipeline), nil
}

// Fetches pipelines for the given search query parameters.
func (s *BasePipelineServer) listPipelines(ctx context.Context, namespace string, opts *list.Options, tagFilters map[string]string) ([]*model.Pipeline, int, string, error) {
	// Fill in the default namespace
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	if err := validation.ValidateNamespaceRequired(namespace); err != nil {
		return nil, 0, "", err
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	if err := s.canAccessPipeline(ctx, "", resourceAttributes); err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list pipelines due to authorization error. Check if you have read permission to namespace %v", namespace)
	}
	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace},
	}

	return s.resourceManager.ListPipelines(filterContext, opts, tagFilters)
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
	filterSpec := request.GetFilter()

	// Extract tag filter predicates (keys prefixed with "tags.") from the filter spec.
	// Tag predicates are handled separately via subqueries and must not be passed to
	// the standard filter/list options which map keys to DB columns.
	cleanedFilterSpec, tagFilters, err := extractTagFiltersFromFilterSpec(filterSpec)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines due to invalid tag filter in filter spec")
	}

	// Validate list options with the cleaned filter (tag predicates removed)
	opts, err := validatedListOptions(&model.Pipeline{}, pageToken, int(pageSize), sortBy, cleanedFilterSpec, "v2beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, cleanedFilterSpec)
	}

	pipelines, totalSize, nextPageToken, err := s.listPipelines(ctx, namespace, opts, tagFilters)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines in namespace %s. Check error stack", namespace)
	}
	return &apiv2beta1.ListPipelinesResponse{Pipelines: toApiPipelines(pipelines), TotalSize: int32(totalSize), NextPageToken: nextPageToken}, nil
}

// extractTagFiltersFromFilterSpec parses a filter spec string, extracts any predicates
// with keys prefixed by "tags." (e.g., "tags.team"), and returns:
//   - cleanedFilterSpec: the filter spec with tag predicates removed (re-serialized)
//   - tagFilters: a map of tag key -> tag value extracted from EQUALS predicates
//   - err: any parsing error
//
// Only EQUALS predicates are supported for tag filtering. Non-tag predicates are
// preserved in the cleaned filter spec for standard filter/list processing.
func extractTagFiltersFromFilterSpec(filterSpec string) (string, map[string]string, error) {
	if filterSpec == "" {
		return "", nil, nil
	}

	decoded, err := url.QueryUnescape(filterSpec)
	if err != nil {
		return filterSpec, nil, util.NewInvalidInputError("failed to decode filter spec: %v", err)
	}

	f := &apiv2beta1.Filter{}
	if err := protojson.Unmarshal([]byte(decoded), f); err != nil {
		return filterSpec, nil, util.NewInvalidInputError("failed to parse filter spec: %v", err)
	}

	var remainingPredicates []*apiv2beta1.Predicate
	tagFilters := make(map[string]string)

	for _, p := range f.GetPredicates() {
		key := p.GetKey()
		if strings.HasPrefix(key, "tags.") {
			tagKey := strings.TrimPrefix(key, "tags.")
			if p.GetOperation() != apiv2beta1.Predicate_EQUALS {
				return "", nil, util.NewInvalidInputError("only EQUALS operation is supported for tag filtering, got %v for key %q", p.GetOperation(), key)
			}
			sv, ok := p.GetValue().(*apiv2beta1.Predicate_StringValue)
			if !ok {
				return "", nil, util.NewInvalidInputError("tag filter value must be a string for key %q", key)
			}
			tagFilters[tagKey] = sv.StringValue
		} else {
			remainingPredicates = append(remainingPredicates, p)
		}
	}

	if len(tagFilters) == 0 {
		return filterSpec, nil, nil
	}

	// Reconstruct the filter spec without tag predicates
	if len(remainingPredicates) == 0 {
		return "", tagFilters, nil
	}
	remainingFilter := &apiv2beta1.Filter{Predicates: remainingPredicates}
	marshaler := &protojson.MarshalOptions{UseProtoNames: true}
	data, err := marshaler.Marshal(remainingFilter)
	if err != nil {
		return "", nil, util.NewInternalServerError(err, "failed to re-serialize filter after extracting tag predicates")
	}
	return url.QueryEscape(string(data)), tagFilters, nil
}

// Removes a pipeline.
func (s *BasePipelineServer) deletePipeline(ctx context.Context, pipelineId string, cascade bool) error {
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

	return s.resourceManager.DeletePipeline(pipelineId, cascade)
}

// Deletes a pipeline.
// Supports v2beta1 behavior.
func (s *PipelineServer) DeletePipeline(ctx context.Context, request *apiv2beta1.DeletePipelineRequest) (*emptypb.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineRequests.Inc()
	}

	if err := s.deletePipeline(ctx, request.GetPipelineId(), request.GetCascade()); err != nil {
		return nil, util.Wrapf(err, "Failed to delete pipeline %s. Check error stack", request.GetPipelineId())
	}

	if s.options.CollectMetrics {
		pipelineCount.Dec()
	}

	return &emptypb.Empty{}, nil
}

// Fetches the latest pipeline version for a given pipeline id.
func (s *BasePipelineServer) getLatestPipelineVersion(ctx context.Context, pipelineId string) (*model.PipelineVersion, error) {
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
func (s *BasePipelineServer) validatePipelineVersionBeforeCreating(p *model.PipelineVersion) error {
	if p.Name == "" {
		return util.NewInvalidInputError("name is required")
	}

	if p.PipelineSpec != "" {
		return nil
	}
	if p.PipelineSpecURI != "" {
		if _, err := url.ParseRequestURI(string(p.PipelineSpecURI)); err == nil {
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
	return &PipelineServer{
		BasePipelineServer: &BasePipelineServer{
			resourceManager: resourceManager,
			httpClient:      validation.SafePipelineHTTPClient(),
			options:         options,
		},
	}
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
	newPipeline, _, err := s.createPipelineAndPipelineVersion(ctx, pipeline, request.GetPipelineVersion().GetPackageUrl().GetPipelineUrl(), request.GetPipelineVersion().GetTags())
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
func (s *BasePipelineServer) createPipelineVersion(ctx context.Context, pv *model.PipelineVersion) (*model.PipelineVersion, error) {
	// Fail if pipeline URL is missing
	if pv.PipelineSpecURI == "" {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due to missing pipeline URL")
	}

	// Fail if parent pipeline id is missing
	if pv.PipelineId == "" {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due empty parent pipeline id")
	}

	if pv.Name == "" {
		return nil, util.NewInvalidInputError("name is required")
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
	// nolint:staticcheck // [ST1003] Field name matches upstream legacy naming
	pipelineUrl, err := url.ParseRequestURI(string(pv.PipelineSpecURI))
	if err != nil {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due to invalid pipeline spec URI. PipelineSpecURI: %v. Please specify a valid URL", pv.PipelineSpecURI)
	}

	if err := validation.ValidatePipelineURL(pipelineUrl.String()); err != nil {
		glog.Warningf("Pipeline URL validation failed: %v", err)
		return nil, util.NewInvalidInputError("Pipeline URL validation failed")
	}
	resp, err := s.httpClient.Get(pipelineUrl.String())
	if err != nil {
		// Unwrap redirect validation errors so they return 4xx, not 500
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if userErr, ok := urlErr.Err.(*util.UserError); ok {
				return nil, userErr
			}
		}
		return nil, util.NewInternalServerError(err, "Failed to create a pipeline version due error downloading the pipeline spec from %v", pipelineUrl.String())
	} else if resp.StatusCode != http.StatusOK {
		return nil, util.NewInvalidInputError("Failed to fetch pipeline spec with url: %v. Request returned %v", pipelineUrl.String(), resp.Status)
	}
	defer resp.Body.Close()
	pipelineFileName := path.Base(pipelineUrl.String())
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version due error reading the pipeline spec")
	}
	pv.PipelineSpec = model.LargeText(pipelineFile)
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
func (s *BasePipelineServer) getPipelineVersion(ctx context.Context, pipelineVersionId string) (*model.PipelineVersion, error) {
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
func (s *BasePipelineServer) listPipelineVersions(ctx context.Context, pipelineID string, opts *list.Options, tagFilters map[string]string) ([]*model.PipelineVersion, int, string, error) {
	// Fail fast if pipeline id is missing
	if pipelineID == "" {
		return nil, 0, "", util.NewInvalidInputError("Failed to list pipeline versions. Pipeline id cannot be empty")
	}

	// Check authorization
	namespace, err := s.resourceManager.FetchNamespaceFromPipelineId(pipelineID)
	if err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list pipeline versions due to error fetching the namespace for pipeline %v", pipelineID)
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: namespace,
		Verb:      common.RbacResourceVerbList,
	}
	if err := s.canAccessPipelineVersion(ctx, "", resourceAttributes); err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list pipeline versions due to authorization error. Check if you have read permission to namespace %v", namespace)
	}

	// Get pipeline versions
	return s.resourceManager.ListPipelineVersions(pipelineID, opts, tagFilters)
}

// Returns an array of pipeline versions for a given query.
// Supports v2beta1 behavior.
func (s *PipelineServer) ListPipelineVersions(ctx context.Context, request *apiv2beta1.ListPipelineVersionsRequest) (*apiv2beta1.ListPipelineVersionsResponse, error) {
	if s.options.CollectMetrics {
		listPipelineVersionRequests.Inc()
	}

	pipelineId := request.GetPipelineId()
	pageToken := request.GetPageToken()
	pageSize := request.GetPageSize()
	sortBy := request.GetSortBy()
	filterSpec := request.GetFilter()

	// Extract tag filter predicates (keys prefixed with "tags.") from the filter spec.
	// Tag predicates are handled separately via subqueries and must not be passed to
	// the standard filter/list options which map keys to DB columns.
	cleanedFilterSpec, tagFilters, err := extractTagFiltersFromFilterSpec(filterSpec)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions due to invalid tag filter in filter spec")
	}

	// Validate query parameters with the cleaned filter (tag predicates removed)
	opts, err := validatedListOptions(&model.PipelineVersion{}, pageToken, int(pageSize), sortBy, cleanedFilterSpec, "v2beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, cleanedFilterSpec)
	}

	pipelineVersions, totalSize, nextPageToken, err := s.listPipelineVersions(ctx, pipelineId, opts, tagFilters)
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
func (s *BasePipelineServer) deletePipelineVersion(ctx context.Context, pipelineId string, pipelineVersionId string) error {
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
// Supports v2beta1 behavior.
func (s *PipelineServer) DeletePipelineVersion(ctx context.Context, request *apiv2beta1.DeletePipelineVersionRequest) (*emptypb.Empty, error) {
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
	return &emptypb.Empty{}, nil
}

// recoverClearTagsIntent checks whether the client intended to clear all tags.
// Protobuf binary encoding collapses an empty map to nil during the HTTP→gRPC
// proxy roundtrip. The clearTagsMiddleware detects an empty tags map in the
// JSON body and sets the x-clear-tags gRPC metadata header. When tags is nil
// and that header is present, this function returns a non-nil empty map so the
// store layer deletes all existing tags.
func recoverClearTagsIntent(ctx context.Context, tags map[string]string) map[string]string {
	if tags != nil {
		return tags
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(common.ClearTagsMetadataKey); len(vals) > 0 && vals[0] == "true" {
			return map[string]string{}
		}
	}
	return nil
}

// UpdatePipeline updates a pipeline's mutable fields (display_name, tags).
// Supports v2beta1 behavior.
func (s *PipelineServer) UpdatePipeline(ctx context.Context, request *apiv2beta1.UpdatePipelineRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		updatePipelineRequests.Inc()
	}

	pipeline := request.GetPipeline()
	pipelineID := pipeline.GetPipelineId()
	if pipelineID == "" {
		return nil, util.NewInvalidInputError("Failed to update a pipeline. Pipeline id cannot be empty")
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbUpdate,
	}
	if err := s.canAccessPipeline(ctx, pipelineID, resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to update pipeline %v due to authorization error", pipelineID)
	}

	tags := recoverClearTagsIntent(ctx, pipeline.GetTags())
	updatedPipeline, err := s.resourceManager.UpdatePipeline(pipelineID, pipeline.GetDisplayName(), tags)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to update pipeline %v. Check error stack", pipelineID)
	}
	return toApiPipeline(updatedPipeline), nil
}

// UpdatePipelineVersion updates a pipeline version's mutable fields (display_name, tags).
// Supports v2beta1 behavior.
func (s *PipelineServer) UpdatePipelineVersion(ctx context.Context, request *apiv2beta1.UpdatePipelineVersionRequest) (*apiv2beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		updatePipelineVersionRequests.Inc()
	}

	pipelineVersion := request.GetPipelineVersion()
	pipelineVersionID := pipelineVersion.GetPipelineVersionId()
	if pipelineVersionID == "" {
		return nil, util.NewInvalidInputError("Failed to update a pipeline version. Pipeline version id cannot be empty")
	}

	pipelineID := pipelineVersion.GetPipelineId()
	if pipelineID == "" {
		return nil, util.NewInvalidInputError("Failed to update pipeline version %v. Pipeline id cannot be empty", pipelineVersionID)
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbUpdate,
	}
	if err := s.canAccessPipelineVersion(ctx, pipelineVersionID, resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to update pipeline version %v due to authorization error", pipelineVersionID)
	}

	tags := recoverClearTagsIntent(ctx, pipelineVersion.GetTags())
	updatedVersion, err := s.resourceManager.UpdatePipelineVersion(pipelineVersionID, pipelineVersion.GetDisplayName(), tags)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to update pipeline version %v. Check error stack", pipelineVersionID)
	}
	return toApiPipelineVersion(updatedVersion), nil
}

// Checks if a user can access a pipeline version.
// Adds namespace of the parent pipeline if version id is not empty,
// API group, version, and resource type.
func (s *BasePipelineServer) canAccessPipelineVersion(ctx context.Context, versionId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	return authorizePipelineVersionAccess(ctx, s.resourceManager, versionId, resourceAttributes)
}

func authorizePipelineVersionAccess(ctx context.Context, resourceManager *resource.ResourceManager, versionID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	pipelineID := ""
	if versionID != "" {
		pipelineVersion, err := resourceManager.GetPipelineVersion(versionID)
		if err != nil {
			return util.Wrapf(err, "Failed to access pipeline version %s. Check if it exists", versionID)
		}
		pipelineID = pipelineVersion.PipelineId
	}
	return authorizePipelineAccess(ctx, resourceManager, pipelineID, resourceAttributes)
}

// Checks if a user can access a pipeline.
// Adds parent namespace if pipeline id is not empty,
// API group, version, and resource type.
func (s *BasePipelineServer) canAccessPipeline(ctx context.Context, pipelineId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	return authorizePipelineAccess(ctx, s.resourceManager, pipelineId, resourceAttributes)
}

func authorizePipelineAccess(ctx context.Context, resourceManager *resource.ResourceManager, pipelineID string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	_, err := authorizePipelineAccessAndGet(ctx, resourceManager, pipelineID, resourceAttributes)
	return err
}

// authorizePipelineAccessAndGet applies pipeline authorization and returns the
// pipeline loaded to derive its authorization attributes. Callers that also
// need pipeline ownership can reuse the row instead of reading it twice.
func authorizePipelineAccessAndGet(ctx context.Context, resourceManager *resource.ResourceManager, pipelineID string, resourceAttributes *authorizationv1.ResourceAttributes) (*model.Pipeline, error) {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil, nil
	}
	var pipeline *model.Pipeline
	if pipelineID != "" {
		var err error
		pipeline, err = resourceManager.GetPipeline(pipelineID)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to access pipeline %s. Check if it exists and have a namespace assigned", pipelineID)
		}
		resourceAttributes.Namespace = pipeline.Namespace
		if resourceAttributes.Name == "" {
			resourceAttributes.Name = pipeline.Name
		}
	}
	// Skip authorization for read-only operations on shared pipelines in multi-user mode.
	// Write operations (create, update, delete) on shared pipelines must still be authorized
	// against the KFP system namespace, since shared pipelines have no namespace of their own.
	if resourceManager.IsEmptyNamespace(resourceAttributes.Namespace) {
		if resourceAttributes.Verb == common.RbacResourceVerbGet || resourceAttributes.Verb == common.RbacResourceVerbList {
			return pipeline, nil
		}
		resourceAttributes.Namespace = common.GetPodNamespace()
		resourceAttributes.Group = common.RbacPipelinesGroup
		resourceAttributes.Version = common.RbacPipelinesVersion
		resourceAttributes.Resource = common.RbacResourceTypePipelines
		err := resourceManager.IsAuthorized(ctx, resourceAttributes)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to access shared pipeline %s. Check if you have permission to %s shared pipelines", pipelineID, resourceAttributes.Verb)
		}
		return pipeline, nil
	}
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypePipelines
	err := resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to access pipeline %s. Check if you have access to namespace %s", pipelineID, resourceAttributes.Namespace)
	}
	return pipeline, nil
}

// buildPipelineName extracts the common logic of naming the pipeline.
// The API caller can either explicitly name the pipeline through query strings ?name=foobar and/or
// ?display_name=foobar, or the API server can use the file name by default.
func buildPipelineName(pipelineName string, pipelineDisplayName string, fileName string) string {
	if pipelineName != "" {
		return pipelineName
	}

	if pipelineDisplayName != "" {
		return pipelineDisplayName
	}

	return fileName
}
