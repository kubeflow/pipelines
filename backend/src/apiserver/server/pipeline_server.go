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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
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
	// ApiVersion       string `default:"v2beta1" json:"api_version,omitempty"`
	// DefaultNamespace string `default:"" json:"default_namespace,omitempty"`
}

type PipelineServer struct {
	resourceManager *resource.ResourceManager
	httpClient      *http.Client
	options         *PipelineServerOptions
}

func NewPipelineServer(resourceManager *resource.ResourceManager, options *PipelineServerOptions) *PipelineServer {
	return &PipelineServer{resourceManager: resourceManager, httpClient: http.DefaultClient, options: options}
}

// This method extract the common logic of naming the pipeline.
// API caller can either explicitly name the pipeline through query string ?name=foobar.
// or API server can use the file name by default.
func buildPipelineName(queryString string, fileName string) (string, error) {
	pipelineName, err := url.QueryUnescape(queryString)
	if err != nil {
		return "", util.NewInvalidInputErrorWithDetails(err, fmt.Sprintf("Failed to extract pipeline's name fro the query as it has invalid format: %s", queryString))
	}
	if pipelineName == "" {
		pipelineName = fileName
	}
	if len(pipelineName) > common.MaxFileNameLength {
		return "", util.NewInvalidInputError("Failed to extract pipeline's name as it is too long. Maximum length is %v", common.MaxFileNameLength)
	}
	return pipelineName, nil
}

// Checks at the resource manager whether a user has access to a resource.
func (s *PipelineServer) checkAccess(ctx context.Context, resourceAttributes *authorizationv1.ResourceAttributes) error {
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypePipelines
	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, "Failed to authorize with API")
	}
	return nil
}

// Checks if a user can access a pipeline version.
func (s *PipelineServer) canAccessPipelineVersion(ctx context.Context, versionId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if versionId != "" {
		namespace, err := s.resourceManager.FetchNamespaceFromPipelineVersionId(versionId)
		if err != nil {
			return util.Wrapf(err, "Failed to access pipeline version %s. Check if it exists and have a namespace assigned", versionId)
		}
		// Allow users to access pipelines in the default namespace.
		if s.resourceManager.IsDefaultNamespace(namespace) {
			return nil
		}
		resourceAttributes.Namespace = namespace
	}
	err := s.checkAccess(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to to access pipeline version %s. Check if you have access to namespace %s", versionId, resourceAttributes.Namespace)
	}
	return nil
}

// Checks if a user can access a pipeline.
func (s *PipelineServer) canAccessPipeline(ctx context.Context, pipelineId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if pipelineId != "" {
		namespace, err := s.resourceManager.FetchNamespaceFromPipelineId(pipelineId)
		if err != nil {
			return util.Wrapf(err, "Failed to access pipeline %s. Check if it exists and have a namespace assigned", pipelineId)
		}
		// Allow users to access pipelines in the default namespace.
		if s.resourceManager.IsDefaultNamespace(namespace) {
			return nil
		}
		resourceAttributes.Namespace = namespace
	}
	err := s.checkAccess(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to to access pipeline %s. Check if you have access to namespace %s", pipelineId, resourceAttributes.Namespace)
	}
	return nil
}

// Validates a pipeline version before creating a record in the DB.
// Requires Name and PipelineId to be non-empty and presence of PipelineSpec or a valid URI to the pipeline spec.
func (s *PipelineServer) validatePipelineVersionBeforeCreating(p *model.PipelineVersion) error {
	if p.Name == "" {
		return util.NewInvalidInputError("Pipeline version's name cannot be empty")
	}
	if p.PipelineId == "" {
		return util.NewInvalidInputError("Pipeline version cannot have empty parent pipeline id")
	}
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

// Create a pipeline from a model.Pipeline. Not exported.
func (s *PipelineServer) createPipeline(ctx context.Context, pipeline *model.Pipeline) (*model.Pipeline, error) {
	// Validate the pipeline. Fail fast if this is corrupted.
	if pipeline.Name == "" {
		return nil, util.NewInvalidInputError("Failed create to a pipeline due to empty name. Please specify a valid name")
	}
	pipeline.Namespace = s.resourceManager.ReplaceNamespace(pipeline.Namespace)
	// Check authorization
	if common.IsMultiUserMode() {
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: pipeline.Namespace,
			Verb:      common.RbacResourceVerbCreate,
			Resource:  common.RbacResourceTypePipelines,
		}
		err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
		if err != nil {
			return nil, util.Wrapf(err, "Failed to create a pipeline due to authorization error. Check if you have write permissions to namespace %s", pipeline.Namespace)
		}
	}
	return s.resourceManager.CreatePipeline(pipeline)
}

// Create a pipeline version from a model.PipelineVersion. Not exported.
func (s *PipelineServer) createPipelineVersion(ctx context.Context, pv *model.PipelineVersion) (*model.PipelineVersion, error) {
	// Fail if pipeline spec is missing
	if (pv.PipelineSpec == "") && (pv.CodeSourceUrl == "") && (pv.PipelineSpecURI == "") {
		return nil, util.NewInvalidInputError("Failed to create a pipeline version due to missing pipeline spec")
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	if err := s.canAccessPipeline(ctx, pv.PipelineId, resourceAttributes); err != nil {
		return nil, util.Wrapf(err, "Failed to create a pipeline version due authorization error for pipeline id %v", pv.PipelineId)
	}

	// Read pipeline file
	if pv.PipelineSpec == "" {
		pipelineUrl, err := url.ParseRequestURI(pv.CodeSourceUrl)
		if err != nil {
			pipelineUrl, err = url.ParseRequestURI(pv.PipelineSpecURI)
			if err != nil {
				return nil, util.NewInvalidInputError("Failed to create a pipeline version due to invalid pipeline spec URI. PipelineSpecURI: %v. CodeSourceUrl: %v. Please specify a valid URL", pv.PipelineSpecURI, pv.CodeSourceUrl)
			}
		}
		resp, err := s.httpClient.Get(pipelineUrl.String())
		if err != nil || resp.StatusCode != http.StatusOK {
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
	}

	// Validate the pipeline version
	if err := s.validatePipelineVersionBeforeCreating(pv); err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline version due to data validation error. Check the error stack")
	}

	return s.resourceManager.CreatePipelineVersion(pv)
}

// Fetches a model.Pipeline.
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

// Fetches a model.PipelineVersion for given pipeline id.
func (s *PipelineServer) getPipelineVersion(ctx context.Context, pipelineVersionId string) (*model.PipelineVersion, error) {
	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err := s.canAccessPipelineVersion(ctx, pipelineVersionId, resourceAttributes)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to get a pipeline version due to authorization error for pipeline version id %v", pipelineVersionId)
	}
	return s.resourceManager.GetPipelineVersion(pipelineVersionId)
}

// Fetches a model.PipelineVersion.
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

// Fetches pipeline and (optionally) pipeline version for a given name and namespace.
func (s *PipelineServer) getPipelineByName(ctx context.Context, name string, namespace string, apiRequestVersion string) (*model.Pipeline, *model.PipelineVersion, error) {
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	if common.IsMultiUserMode() {
		if !s.resourceManager.IsDefaultNamespace(namespace) {
			resourceAttributes := &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      common.RbacResourceVerbGet,
			}
			if err := s.checkAccess(ctx, resourceAttributes); err != nil {
				return nil, nil, util.Wrapf(err, "Failed to fetch a pipeline due to authorization error. Check if you have read permission to namespace %v", namespace)
			}
		}
	}
	switch apiRequestVersion {
	case "v1beta1":
		return s.resourceManager.GetPipelineByNameAndNamespaceV1(name, namespace)
	case "V2beta1":
		p, err := s.resourceManager.GetPipelineByNameAndNamespace(name, namespace)
		return p, nil, err
	default:
		return nil, nil, util.NewInternalServerError(
			util.NewInvalidInputError("Invalid api version detected"),
			"Failed to get a pipeline by name and namespace. API request version %v. Please, file a bug on github: https://github.com/kubeflow/pipelines/issues",
			apiRequestVersion)
	}
}

// Fetches an array of []model.Pipeline and an array of []model.PipelineVersion for given search query parameters.
func (s *PipelineServer) listPipelines(ctx context.Context, namespace string, pageToken string, pageSize int32, sortBy string, filter string, apiRequestVersion string) ([]*model.Pipeline, []*model.PipelineVersion, int, string, error) {
	// Fill in the default namespace
	namespace = s.resourceManager.ReplaceNamespace(namespace)
	if common.IsMultiUserMode() {
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      common.RbacResourceVerbGet,
		}
		if err := s.checkAccess(ctx, resourceAttributes); err != nil {
			return nil, nil, 0, "", util.Wrapf(err, "Failed to list pipelines due to authorization error. Check if you have read permission to namespace %v", namespace)
		}
	}
	filterContext := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: namespace},
	}

	// Validate list options
	opts, err := validatedListOptions(&model.Pipeline{}, pageToken, int(pageSize), sortBy, filter)
	if err != nil {
		return nil, nil, 0, "", util.Wrapf(err, "Failed to list pipelines due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, filter)
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
			"Failed to list pipelines due to unsupported API request. API request version %v. Please, file a bug on github: https://github.com/kubeflow/pipelines/issues",
			apiRequestVersion)
	}
}

// Fetches an array of []model.PipelineVersion for given search query parameters.
func (s *PipelineServer) listPipelineVersions(ctx context.Context, pipelineId string, pageToken string, pageSize int32, sortBy string, filter string) ([]*model.PipelineVersion, int, string, error) {
	// Fail fast of pipeline id or namespace are missing
	if pipelineId == "" {
		return nil, 0, "", util.NewInvalidInputError("Failed to list pipeline versions. Pipeline id cannot be empty")
	}

	// Validate query parameters
	opts, err := validatedListOptions(
		&model.PipelineVersion{},
		pageToken,
		int(pageSize),
		sortBy,
		filter,
	)
	if err != nil {
		return nil, 0, "", util.Wrapf(err, "Failed to list pipeline versions due invalid list options: pageToken: %v, pageSize: %v, sortBy: %v, filter: %v", pageToken, int(pageSize), sortBy, filter)
	}

	// Check authorization
	if common.IsMultiUserMode() {
		namespace, err := s.resourceManager.FetchNamespaceFromPipelineId(pipelineId)
		if err != nil {
			return nil, 0, "", util.Wrapf(err, "Failed to list pipeline versions due to error fetching the namespace for pipeline %v", pipelineId)
		}
		if !s.resourceManager.IsDefaultNamespace(namespace) {
			resourceAttributes := &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      common.RbacResourceVerbGet,
			}
			if err := s.checkAccess(ctx, resourceAttributes); err != nil {
				return nil, 0, "", util.Wrapf(err, "Failed to list pipeline versions due to authorization error. Check if you have read permission to namespace %v", namespace)
			}
		}
	}

	// Get pipeline versions
	return s.resourceManager.ListPipelineVersions(pipelineId, opts)
}

// Removes a model.Pipeline.
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

// Removes a model.PipelineVersion.
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
		Verb: common.RbacResourceVerbList,
	}
	err := s.canAccessPipelineVersion(ctx, pipelineVersionId, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to delete a pipeline version id %v due to authorization error for pipeline id %v", pipelineVersionId, pipelineId)
	}

	return s.resourceManager.DeletePipelineVersion(pipelineVersionId)
}

// Creates a v1beta1 pipeline and a v1beta1 pipeline version.
// If the pipeline already exists, creates a new pipeline version.
// Note, this allows partial creation (pipeline or pipeline version only,
// which is different from v1beta1 behavior.
func (s *PipelineServer) CreatePipelineV1(ctx context.Context, request *apiv1beta1.CreatePipelineRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
		// Request is intended to create a pipeline and a pipeline version
		if (request.GetPipeline().GetDefaultVersion() != nil) || (request.GetPipeline().GetUrl().GetPipelineUrl() != "") {
			// In v1beta1 we did not increment this counter when CreatePipeline was called
			createPipelineVersionRequests.Inc()
		}
	}

	// Convert the input request. Fail fast if either pipeline or pipeline version is corrupted.
	pipeline, err := toModelPipeline(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline (v1beta1) due to pipeline conversion error")
	}
	pipelineVersion, err := toModelPipelineVersion(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a pipeline (v1beta1) due to pipeline version conversion error")
	}

	// Get pipeline name
	pipelineFileName := path.Base(pipelineVersion.CodeSourceUrl)
	pipelineName, err := buildPipelineName(pipeline.Name, pipelineFileName)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to create a new pipeline (v1beta1) due to invalid name and filename combination (%v, %v). Please, file a bug on github: https://github.com/kubeflow/pipelines/issues", pipeline.Name, pipelineFileName)
	}
	pipeline.Name = pipelineName

	// Create the pipeline
	createdPipeline, perr := s.createPipeline(ctx, pipeline)
	if perr == nil {
		if s.options.CollectMetrics {
			// Increment if a new pipeline has been created
			pipelineCount.Inc()
		}
	} else if perr.(*util.UserError).ExternalStatusCode() == codes.AlreadyExists {
		createdPipeline, err = s.resourceManager.GetPipelineByNameAndNamespace(pipeline.Name, pipeline.Namespace)
		if err != nil {
			// This should never happen
			return nil, util.Wrap(perr, util.Wrap(err, "Failed to create a new pipeline (v1beta1), but an existing pipeline was not found. Please, file a bug on github: https://github.com/kubeflow/pipelines/issues").Error())
		}
	} else {
		return nil, util.Wrap(perr, "Failed to create a new pipeline (v1beta1). Check the error stack")
	}
	pipelineVersion.PipelineId = createdPipeline.UUID

	// Create the pipeline version
	createdPipelineVersion, pverr := s.createPipelineVersion(ctx, pipelineVersion)
	if pverr == nil {
		if s.options.CollectMetrics {
			// Increment if a new pipeline version has been created
			pipelineVersionCount.Inc()
		}
		return toApiPipelineV1(createdPipeline, createdPipelineVersion), nil
	}

	// A new pipeline has been created, but pipeline version creation failed after validation
	if perr == nil {
		return toApiPipelineV1(createdPipeline, &model.PipelineVersion{}), nil
	}

	return nil, util.Wrap(perr, util.Wrap(pverr, "Failed to create a pipeline and a pipeline version (v1beta1)").Error())
}

// Creates a v2beta1 pipeline, but does not create a pipeline version.
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

// Creates a v1beta1 pipeline version.
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

// Creates a v2beta1 pipeline version.
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

// Returns a v1beta1 pipeline.
// Default pipeline version is set to be the latest pipeline version.
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

// Returns a v2beta1 pipeline.
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

// Returns a v1beta1 pipeline with the default (latest) pipeline version given name and namespace.
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

// Returns a v2beta1 pipeline given name and namespace.
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

// Return a v2beta1 pipeline version.
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

// Return a v2beta1 pipeline version.
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

// Return the default (latest) pipeline template for a given pipeline id.
// Supports v1beta behavior.
func (s *PipelineServer) GetTemplate(ctx context.Context, request *apiv1beta1.GetTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	pipelineId := request.GetId()
	if pipelineId == "" {
		return nil, util.NewInvalidInputError("Failed to get the default pipeline template (v1beta1). Pipeline id cannot be empty")
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

// Returns pipeline template.
// Supports v1beta1 behavior.
func (s *PipelineServer) GetPipelineVersionTemplate(ctx context.Context, request *apiv1beta1.GetPipelineVersionTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
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

// Returns v1beta1 pipelines with default v1beta1 pipeline versions for a given query.
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

	pipelines, pipelineVersions, totalSize, nextPageToken, err := s.listPipelines(ctx, namespace, pageToken, pageSize, sortBy, filter, "v1beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines (v1beta1) in namespace %s. Check error stack", namespace)
	}

	apiPipelines := toApiPipelinesV1(pipelines, pipelineVersions)
	return &apiv1beta1.ListPipelinesResponse{Pipelines: apiPipelines, TotalSize: int32(totalSize), NextPageToken: nextPageToken}, nil
}

// Returns v2beta1 pipelines for a given query.
func (s *PipelineServer) ListPipelines(ctx context.Context, request *apiv2beta1.ListPipelinesRequest) (*apiv2beta1.ListPipelinesResponse, error) {
	if s.options.CollectMetrics {
		listPipelineRequests.Inc()
	}

	namespace := request.GetNamespace()
	pageToken := request.GetPageToken()
	pageSize := request.GetPageSize()
	sortBy := request.GetSortBy()
	filter := request.GetFilter()

	pipelines, _, totalSize, nextPageToken, err := s.listPipelines(ctx, namespace, pageToken, pageSize, sortBy, filter, "v2beta1")
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipelines in namespace %s. Check error stack", namespace)
	}
	return &apiv2beta1.ListPipelinesResponse{Pipelines: toApiPipelines(pipelines), TotalSize: int32(totalSize), NextPageToken: nextPageToken}, nil
}

// Returns a list of v1beta1 pipeline versions for a given query.
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

	pipelineVersions, totalSize, nextPageToken, err := s.listPipelineVersions(ctx, pipelineId, pageToken, pageSize, sortBy, filter)
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

// Returns a list of v2beta1 pipeline versions for a given query.
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

	pipelineVersions, totalSize, nextPageToken, err := s.listPipelineVersions(ctx, pipelineId, pageToken, pageSize, sortBy, filter)
	if err != nil {
		return nil, util.Wrapf(err, "Failed to list pipeline versions for pipeline %s", pipelineId)
	}
	return &apiv2beta1.ListPipelineVersionsResponse{
		PipelineVersions: toApiPipelineVersions(pipelineVersions),
		NextPageToken:    nextPageToken,
		TotalSize:        int32(totalSize),
	}, nil
}

// TODO(gkcalat): consider removing before v2beta1 GA as default version is deprecated. This requires changes to v1beta1 proto.
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

// Deletes a v1beta1 pipeline.
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

// Deletes a v2beta1 pipeline.
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

// Deletes a v1beta1 pipeline version.
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

// Deletes a v2beta1 pipeline version.
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
