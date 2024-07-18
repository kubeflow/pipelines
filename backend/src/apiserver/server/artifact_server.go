// Copyright 2024 The Kubeflow Authors
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
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/objectstore"
	"github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/protobuf/types/known/timestamppb"
	authorizationv1 "k8s.io/api/authorization/v1"
	"strconv"
	"strings"
	"time"
)

var (
	getArtifactRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "artifact_server_get_requests",
		Help: "The total number of GetArtifact requests",
	})
	listArtifactRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "artifact_server_list_requests",
		Help: "The total number of ListArtifacts requests",
	})
)

type ArtifactServerOptions struct {
	CollectMetrics bool `json:"collect_metrics,omitempty"`
}

type ArtifactServer struct {
	resourceManager *resource.ResourceManager
	options         *ArtifactServerOptions
}

// Value constraints by MLMD:
// https://github.com/kubeflow/pipelines/blob/38ef986eaa00e8e8e634a17a7837111b6380685a/third_party/ml-metadata/ml_metadata/proto/metadata_store.proto#L873
const (
	DefaultListArtifactSize = 20
	ArtifactSizeMaximum     = 100
	ContextSizeMaximum      = 100
	ArtifactSizeMinimum     = 1
)

// ListArtifacts lists artifacts. Namespace filtering is assessed based on querying mlmd
// for all contexts within the provided namespace then fetching all artifacts
// associated with each context.
// TODO: Add namespace custom property to artifacts to skip context fetching step.
func (s *ArtifactServer) ListArtifacts(ctx context.Context, r *apiv2beta1.ListArtifactRequest) (*apiv2beta1.ListArtifactResponse, error) {
	if s.options.CollectMetrics {
		listArtifactRequests.Inc()
	}

	if r.Namespace == "" {
		return nil, fmt.Errorf("Missing required namespace parameter.")
	}

	err := s.canAccessArtifact(ctx, r.Namespace, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbList})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	orderByField := r.OrderByField.String()

	orderByAsc := true
	if r.OrderBy == "desc" {
		orderByAsc = false
	}

	maxResultSize := r.MaxResultSize
	if maxResultSize < ArtifactSizeMinimum {
		maxResultSize = DefaultListArtifactSize
	} else if maxResultSize > ArtifactSizeMaximum {
		maxResultSize = ArtifactSizeMaximum
	}

	contextFilterQuery := fmt.Sprintf("custom_properties.namespace.string_value = \"%s\"", r.Namespace)

	nextPageToken := new(string)
	var contextIds []string
	for nextPageToken != nil {
		var err1 error
		var contexts []*ml_metadata.Context
		contexts, nextPageToken, err1 = s.resourceManager.GetContexts(
			ctx,
			ContextSizeMaximum,
			false,
			"",
			contextFilterQuery,
			*nextPageToken)
		if err1 != nil {
			return nil, util.Wrap(err1, "Failed to list artifacts.")
		}
		for _, c := range contexts {
			contextIds = append(contextIds, strconv.FormatInt(*c.Id, 10))
		}
	}

	if len(contextIds) == 0 {
		return nil, fmt.Errorf("Failed to find any artifacts within the specified namespace.")
	}

	artifactFilterQuery := fmt.Sprintf("contexts_a.id in (%s)", strings.Join(contextIds, ","))

	artifacts, nextPageToken, err := s.resourceManager.GetArtifacts(
		ctx,
		maxResultSize,
		orderByAsc,
		orderByField,
		artifactFilterQuery,
		r.NextPageToken)

	if err != nil {
		return nil, util.Wrap(err, "Failed to list artifacts.")
	}

	var artifactsResp []*apiv2beta1.Artifact
	for _, artifact := range artifacts {
		bucketConfig, namespace, err1 := s.resourceManager.GetArtifactSessionInfo(ctx, artifact)
		artifactId := strconv.FormatInt(*artifact.Id, 10)
		if err1 != nil || bucketConfig == nil {
			return nil, util.NewInternalServerError(fmt.Errorf("failed to retrieve session info error: %v", err1), artifactId)
		}
		artifactResp, err1 := s.generateResponseArtifact(ctx, artifact, bucketConfig, namespace, false)
		if err1 != nil {
			return nil, util.NewInternalServerError(fmt.Errorf("encountered error parsing artifact: %v", err), artifactId)
		}
		artifactsResp = append(artifactsResp, artifactResp)
	}

	resp := &apiv2beta1.ListArtifactResponse{
		Artifacts: artifactsResp,
	}

	if nextPageToken != nil {
		resp.NextPageToken = *nextPageToken
	}
	return resp, nil
}

func (s *ArtifactServer) GetArtifact(ctx context.Context, r *apiv2beta1.GetArtifactRequest) (*apiv2beta1.Artifact, error) {
	if s.options.CollectMetrics {
		getArtifactRequests.Inc()
	}

	artifactId, err := strconv.ParseInt(r.ArtifactId, 10, 64)
	if err != nil {
		return nil, util.Wrap(err, fmt.Sprintf("failed to parse artifact parameter in request: %v", err))
	}
	artifacts, err := s.resourceManager.GetArtifactById(ctx, []int64{artifactId})

	// note artifacts length will never be greater than one
	// since artifact ID uniquely identifies one artifact but we add a check for completeness
	if err != nil || artifacts == nil || len(artifacts) > 1 {
		return nil, util.NewResourceNotFoundError(fmt.Sprintf("failed to find artifact with id %d: %v", artifactId, err), r.ArtifactId)
	}

	artifact := artifacts[0]
	sessionInfo, namespace, err := s.resourceManager.GetArtifactSessionInfo(ctx, artifact)
	if err != nil || sessionInfo == nil {
		return nil, util.NewInternalServerError(fmt.Errorf("failed to retrieve session info error: %v", err), r.ArtifactId)
	}

	err = s.canAccessArtifact(ctx, namespace, &authorizationv1.ResourceAttributes{Verb: common.RbacResourceVerbGet})
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	downloadURL := false
	if r.GetView() == apiv2beta1.GetArtifactRequest_DOWNLOAD {
		downloadURL = true
	}

	artifactResp, err := s.generateResponseArtifact(ctx, artifact, sessionInfo, namespace, downloadURL)
	if err != nil {
		return nil, util.NewInternalServerError(fmt.Errorf("encountered error parsing artifact: %v", err), r.ArtifactId)
	}

	return artifactResp, nil
}

func NewArtifactServer(resourceManager *resource.ResourceManager, options *ArtifactServerOptions) *ArtifactServer {
	return &ArtifactServer{resourceManager: resourceManager, options: options}
}

// generateResponseArtifact will return artifact metadata when given an mlmd artifact
// with the artifact's associated namespace and bucketconfig. When includeShareUrl is true
// the metadata will include a signed URL for the associated artifact.
func (s *ArtifactServer) generateResponseArtifact(
	ctx context.Context,
	artifact *ml_metadata.Artifact,
	bucketConfig *objectstore.Config,
	namespace string,
	includeShareUrl bool,
) (*apiv2beta1.Artifact, error) {
	secret, err := s.resourceManager.GetSecret(ctx, namespace, bucketConfig.Session.SecretName)
	if err != nil {
		return nil, err
	}
	key, err := objectstore.ArtifactKeyFromURI(*artifact.Uri)
	if err != nil {
		return nil, err
	}
	size, err := s.resourceManager.GetObjectSize(bucketConfig, secret, *artifact.Uri)
	if err != nil {
		return nil, err
	}
	artifactResp := &apiv2beta1.Artifact{
		ArtifactId:      strconv.FormatInt(*artifact.Id, 10),
		ArtifactType:    *artifact.Type,
		ArtifactSize:    size,
		StorageProvider: strings.TrimSuffix(bucketConfig.Scheme, "://"),
		StoragePath:     key,
		Uri:             *artifact.Uri,
		Namespace:       namespace,
		CreatedAt:       timestamppb.New(time.UnixMilli(*artifact.CreateTimeSinceEpoch)),
		LastUpdatedAt:   timestamppb.New(time.UnixMilli(*artifact.LastUpdateTimeSinceEpoch)),
	}

	if includeShareUrl {
		expiry := time.Second * time.Duration(common.GetSignedURLExpiryTimeSeconds())
		shareUrl, err := s.resourceManager.GetSignedUrl(bucketConfig, secret, expiry, *artifact.Uri)
		if err != nil {
			return nil, err
		}
		artifactResp.DownloadUrl = shareUrl
	}

	return artifactResp, nil
}

// Checks if a user can access an Artifact.
func (s *ArtifactServer) canAccessArtifact(ctx context.Context, namespace string, resourceAttributes *authorizationv1.ResourceAttributes) error {
	if !common.IsMultiUserMode() {
		// Skip authz if not multi-user mode.
		return nil
	}
	resourceAttributes.Namespace = namespace
	resourceAttributes.Group = common.RbacPipelinesGroup
	resourceAttributes.Version = common.RbacPipelinesVersion
	resourceAttributes.Resource = common.RbacResourceTypeArtifacts
	err := s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return util.Wrapf(err, "Failed to access artifact. Check if you have access to namespace %s", resourceAttributes.Namespace)
	}
	return nil
}
