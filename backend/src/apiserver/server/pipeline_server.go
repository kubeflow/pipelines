// Copyright 2018-2022 The Kubeflow Authors
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
	"google.golang.org/grpc/codes"
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
	CollectMetrics   bool   `json:"collect_metrics,omitempty"`
	ApiVersion       string `default:"v2beta1" json:"api_version,omitempty"`
	DefaultNamespace string `default:"default" json:"default_namespace,omitempty"`
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
func getPipelineName(queryString string, fileName string) (string, error) {
	pipelineName, err := url.QueryUnescape(queryString)
	if err != nil {
		return "", util.NewInvalidInputErrorWithDetails(err, "Failed to extract pipeline's name as it has invalid format.")
	}
	if pipelineName == "" {
		pipelineName = fileName
	}
	if len(pipelineName) > common.MaxFileNameLength {
		return "", util.NewInvalidInputError("Failed to extract pipeline's name as it is too long. Maximum length is %v.", common.MaxFileNameLength)
	}
	return pipelineName, nil
}

// Checks if a user can access a pipeline version.
func (s *PipelineServer) canAccessPipelineVersion(ctx context.Context, versionId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if versionId != "" {
		namespace, err := s.resourceManager.GetNamespaceFromPipelineVersion(versionId)
		// Manually allow users to access pipelines in the default namespace.
		if (namespace == "") || (namespace == s.options.DefaultNamespace) {
			return nil
		}
		if err != nil {
			return util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to authorize with the pipeline version id %s.", s.options.ApiVersion, versionId))
		}
		resourceAttributes.Namespace = namespace
	}
	if resourceAttributes.Namespace == "" {
		return nil
	}
	return s.haveAccess(ctx, resourceAttributes)
}

// Checks if a user can access a pipeline.
func (s *PipelineServer) canAccessPipeline(ctx context.Context, pipelineId string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authorization if not multi-user mode.
		return nil
	}
	if pipelineId != "" {
		namespace, err := s.resourceManager.GetNamespaceFromPipelineID(pipelineId)
		// Manually allow users to access pipelines in the default namespace.
		if (namespace == "") || (namespace == s.options.DefaultNamespace) {
			return nil
		}
		if err != nil {
			return util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to authorize with the pipeline id %s.", s.options.ApiVersion, pipelineId))
		}
		resourceAttributes.Namespace = namespace
	}
	if resourceAttributes.Namespace == "" {
		return nil
	}
	return s.haveAccess(ctx, resourceAttributes)
}

// Checks if a user has access to a resource.
func (s *PipelineServer) haveAccess(ctx context.Context, resourceAttributes *authorizationv1.ResourceAttributes) error {
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypePipelines
	err := isAuthorized(s.resourceManager, ctx, resourceAttributes)
	if err != nil {
		return util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to authorize with API.", s.options.ApiVersion))
	}
	return nil
}

// Validates a pipeline before creating a record in the DB.
// Requires Name and Namespace to be non-empty.
func (s *PipelineServer) validatePipelineBeforeCreating(p *model.Pipeline) error {
	if p.Name == "" {
		return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s]: Failed to validate a pipeline due to empty name. Please specify a valid name.", s.options.ApiVersion))
	}
	if p.Namespace == "" {
		if err := p.SetProtoFieldValue("namespace", s.options.DefaultNamespace); err != nil {
			return util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to validate a pipeline. Cannot set to a default namespace %s. Try specifying a custom namespace.", s.options.ApiVersion, s.options.DefaultNamespace))
		}
		if p.Namespace == "" {
			return util.NewInternalServerError(fmt.Errorf("Empty default namespace."), "[PipelineServer %s]: Failed to validate a pipeline. Default namespace is empty. Specify it and restart the API server.", s.options.ApiVersion)
		}
	}
	return nil
}

// Validates a pipeline version before creating a record in the DB.
// Requires Name and PipelineId to be non-empty and presence of PipelineSpec or a valid URI to the pipeline spec.
func (s *PipelineServer) validatePipelineVersionBeforeCreating(p *model.PipelineVersion) error {
	if p.Name == "" {
		return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s]: Failed to validate a pipeline version due to empty name.", s.options.ApiVersion))
	}
	if p.PipelineId == "" {
		return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s]: Failed to validate a pipeline version due to empty parent pipeline id.", s.options.ApiVersion))
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
	return util.NewInvalidInputError(fmt.Sprintf("[PipelineServer %s]: Failed to validate a pipeline version due missing pipeline spec and invalid source code's URL. PipelineSpec: %s. PipelineSpecURI: %s. CodeSourceUrl: %s. At least one of them must have a valid pipeline spec.", opts.ApiVersion, p.PipelineSpec, p.PipelineSpecURI, p.CodeSourceUrl))
}

// v2beta1: creates a pipeline.
// v1beta1: creates a pipeline and a pipeline version.
func (s *PipelineServer) CreatePipeline(ctx context.Context, r interface{}) (interface{}, error) {
	switch r.(type) {
	case *apiv1beta1.CreatePipelineRequest:
		return s.createPipelineV1(ctx, r.(*apiv1beta1.CreatePipelineRequest))
	case *apiv2beta1.CreatePipelineRequest:
		return s.createPipelineV2(ctx, r.(*apiv2beta1.CreatePipelineRequest))
	default:
		return nil, util.NewUnknownApiVersionError("CreatePipeline", fmt.Sprintf("%t", r))
	}
}

// Create a pipeline and a pipeline version.
// If the pipeline already exists, creates a new pipeline version.
// Supports v1beta1 behavior, but allows partial creation (pipeline or pipeline version only).
func (s *PipelineServer) createPipelineV1(ctx context.Context, request *apiv1beta1.CreatePipelineRequest) (*apiv1beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
		if s.options.ApiVersion == "v2beta1" {
			// Request is intended to create a pipeline and a pipeline version
			if (request.GetPipeline().GetDefaultVersion() != nil) || (request.GetPipeline().GetUrl().GetPipelineUrl() != "") {
				// In v1beta1 we did not increment this counter when CreatePipeline was called
				createPipelineVersionRequests.Inc()
			}
		}
	}

	// Convert the input request. Fail fast if either pipeline or pipeline version is corrupted.
	pipeline, err := ToModelPipeline(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline (v1beta1) as pipeline conversion failed.", s.options.ApiVersion))
	}
	pipelineVersion, err := ToModelPipelineVersion(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline (v1beta1) as pipeline version conversion failed.", s.options.ApiVersion))
	}

	// Check authorization
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: pipeline.Namespace,
		Verb:      common.RbacResourceVerbCreate,
		Resource:  common.RbacResourceTypePipelines,
	}
	err = isAuthorized(s.resourceManager, ctx, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline (v1beta1) due to authorization error. Check if you have write permissions to namespace %s.", s.options.ApiVersion, pipeline.Namespace))
	}

	// Check pipeline spec
	resp, err := s.httpClient.Get(pipelineVersion.CodeSourceUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, util.NewInternalServerError(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline (v1beta1) as pipeline URL %v cannot be accessed.", s.options.ApiVersion, pipelineVersion.CodeSourceUrl))
	}
	pipelineFileName := path.Base(pipelineVersion.CodeSourceUrl)
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline (v1beta1) as pipeline spec is corrupted.", s.options.ApiVersion))
	}
	pipelineVersion.PipelineSpec = string(pipelineFile)

	// Get pipeline name
	pipelineName, err := getPipelineName(pipeline.Name, pipelineFileName)
	pipeline.Name = pipelineName

	// Validate the pipeline. Fail fast if this is corrupted.
	if err := s.validatePipelineBeforeCreating(&pipeline); err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline (v1beta1) as pipeline validation failed.", s.options.ApiVersion))
	}

	// Create the pipeline
	createdPipeline, perr := s.resourceManager.CreatePipeline(pipeline)
	if perr == nil {
		if s.options.CollectMetrics {
			// Increment if a new pipeline has been created
			pipelineCount.Inc()
		}
	} else if perr.(*util.UserError).ExternalStatusCode() == codes.AlreadyExists {
		createdPipeline, err = s.resourceManager.GetPipelineByNameAndNamespace(pipeline.Name, pipeline.Namespace)
		if err != nil {
			// This should never happen
			return nil, util.Wrap(perr, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a new pipeline (v1beta1), but an existing pipeline was not found. Please, file a bug on github: https://github.com/kubeflow/pipelines/issues.", s.options.ApiVersion)).Error())
		}
	} else {
		return nil, util.Wrap(perr, fmt.Sprintf("[PipelineServer %s]: Failed to create a new pipeline (v1beta1).", s.options.ApiVersion))
	}

	// Validate the pipeline version
	if pverr := s.validatePipelineVersionBeforeCreating(&pipelineVersion); pverr != nil {
		if perr != nil {
			return nil, util.Wrap(perr, util.Wrap(pverr, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline (v1beta1) and failed to validate pipeline version.", s.options.ApiVersion)).Error())
		} else {
			// A new pipeline has been created
			return ToApiPipelineV1(createdPipeline, &model.PipelineVersion{}), nil
		}
	}

	// Create the pipeline version
	createdPipelineVersion, pverr := s.resourceManager.CreatePipelineVersion(pipelineVersion)
	if pverr == nil {
		if s.options.CollectMetrics {
			// Increment if a new pipeline version has been created
			pipelineVersionCount.Inc()
		}
		return ToApiPipelineV1(createdPipeline, createdPipelineVersion), nil
	}

	// A new pipeline has been created
	if perr == nil {
		return ToApiPipelineV1(createdPipeline, &model.PipelineVersion{}), nil
	}

	return nil, util.Wrap(perr, util.Wrap(pverr, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline and a pipeline version (v1beta1).", s.options.ApiVersion)).Error())
}

// Creates a pipeline, but does not create a pipeline version.
func (s *PipelineServer) createPipelineV2(ctx context.Context, request *apiv2beta1.CreatePipelineRequest) (*apiv2beta1.Pipeline, error) {
	if s.options.CollectMetrics {
		createPipelineRequests.Inc()
	}

	// Convert the input request. Fail fast if pipeline is corrupted.
	pipeline, err := ToModelPipeline(request.GetPipeline())
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline as pipeline conversion failed.", s.options.ApiVersion))
	}

	// Validate the pipeline. Fail fast if this is corrupted.
	if err := s.validatePipelineBeforeCreating(&pipeline); err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a pipeline as pipeline validation failed.", s.options.ApiVersion))
	}

	// Create the pipeline
	createdPipeline, err := s.resourceManager.CreatePipeline(pipeline)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to create a new pipeline.", s.options.ApiVersion))
	}

	if s.options.CollectMetrics {
		pipelineCount.Inc()
	}
	return ToApiPipeline(createdPipeline), nil
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
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
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
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
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
	if err = s.canAccessPipeline(ctx, "", resourceAttributes); err != nil {
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
	pipelineFile, err := ReadPipelineFile(pipelineFileName, resp.Body, common.MaxFileLength)
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
	if err = s.canAccessPipeline(ctx, "", resourceAttributes); err != nil {
		return nil, util.Wrap(err, "Failed to authorize with API")
	}

	version, err := s.resourceManager.CreatePipelineVersion(request.Version, pipelineFile, common.IsPipelineVersionUpdatedByDefault())
	if err != nil {
		return nil, util.Wrap(err, "Failed to create a version.")
	}
	return ToApiPipelineVersionV1(version)
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
	err = s.canAccessPipeline(ctx, request.Id, resourceAttributes)
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

func (s *PipelineServer) GetTemplate(ctx context.Context, request *apiv1beta1.GetTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err := s.canAccessPipeline(ctx, request.Id, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	template, err := s.resourceManager.GetPipelineTemplate(request.Id)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline template failed.")
	}

	return &apiv1beta1.GetTemplateResponse{Template: string(template)}, nil
}

func (s *PipelineServer) GetPipelineVersionV1(ctx context.Context, request *apiv1beta1.GetPipelineVersionRequest) (*apiv1beta1.PipelineVersion, error) {
	if s.options.CollectMetrics {
		getPipelineVersionRequests.Inc()
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err := s.canAccessPipelineVersion(ctx, request.VersionId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	version, err := s.resourceManager.GetPipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Get pipeline version failed.")
	}
	return ToApiPipelineVersionV1(version)
}

// Returns pipeline template.
// Supports v1beta1 behavior.
func (s *PipelineServer) GetPipelineVersionTemplate(ctx context.Context, request *apiv1beta1.GetPipelineVersionTemplateRequest) (*apiv1beta1.GetTemplateResponse, error) {
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbList,
	}
	err := s.canAccessPipelineVersion(ctx, request.VersionId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to get a pipeline template due to authorization error.", s.options.ApiVersion))
	}
	template, err := s.resourceManager.GetPipelineVersionTemplate(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("[PipelineServer %s]: Failed to get a pipeline template.", s.options.ApiVersion))
	}

	return &apiv1beta1.GetTemplateResponse{Template: string(template)}, nil
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
		filterContext = &model.FilterContext{
			ReferenceKey: &model.ReferenceKey{Type: model.NamespaceResourceType, ID: ""},
		}
	}
	if refKey != nil && refKey.Type != model.NamespaceResourceType {
		return nil, util.NewInvalidInputError("Invalid resource references for pipelines. ListPipelines requires filtering by namespace.")
	}
	if refKey != nil && refKey.Type == model.NamespaceResourceType {
		namespace := refKey.ID
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      common.RbacResourceVerbList,
		}
		if err = s.canAccessPipeline(ctx, "", resourceAttributes); err != nil {
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
		err = s.canAccessPipelineVersion(ctx, "", resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize with API")
		}
	}

	pipelineVersions, totalSize, nextPageToken, err :=
		s.resourceManager.ListPipelineVersions(request.ResourceKey.Id, opts)
	if err != nil {
		return nil, util.Wrap(err, "List pipeline versions failed.")
	}
	apiPipelineVersions, _ := ToApiPipelineVersionsV1(pipelineVersions)

	return &apiv1beta1.ListPipelineVersionsResponse{
		Versions:      apiPipelineVersions,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize)}, nil
}

func (s *PipelineServer) UpdatePipelineDefaultVersionV1(ctx context.Context, request *apiv1beta1.UpdatePipelineDefaultVersionRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		updatePipelineDefaultVersionRequests.Inc()
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbUpdate,
	}
	err := s.canAccessPipeline(ctx, request.PipelineId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the requests.")
	}
	err = s.resourceManager.UpdatePipelineDefaultVersion(request.PipelineId, request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Update Pipeline Default Version failed.")
	}
	return &empty.Empty{}, nil
}

func (s *PipelineServer) DeletePipelineV1(ctx context.Context, request *apiv1beta1.DeletePipelineRequest) (*empty.Empty, error) {
	if s.options.CollectMetrics {
		deletePipelineRequests.Inc()
	}
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Verb: common.RbacResourceVerbDelete,
	}
	err := s.canAccessPipeline(ctx, request.Id, resourceAttributes)
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
	err = s.canAccessPipeline(ctx, pipelineVersion.PipelineId, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize with API")
	}
	err = s.resourceManager.DeletePipelineVersion(request.VersionId)
	if err != nil {
		return nil, util.Wrap(err, "Delete pipeline versions failed.")
	}

	return &empty.Empty{}, nil
}
