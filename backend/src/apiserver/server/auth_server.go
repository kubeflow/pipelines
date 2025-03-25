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
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authorizationv1 "k8s.io/api/authorization/v1"
)

var rbacResourceTypeToGroup = map[string]string{
	common.RbacResourceTypePipelines:   common.RbacPipelinesGroup,
	common.RbacResourceTypeExperiments: common.RbacPipelinesGroup,
	common.RbacResourceTypeRuns:        common.RbacPipelinesGroup,
	common.RbacResourceTypeJobs:        common.RbacPipelinesGroup,
	common.RbacResourceTypeViewers:     common.RbacKubeflowGroup,
}

type AuthServer struct {
	resourceManager *resource.ResourceManager
}

func (s *AuthServer) AuthorizeV1(ctx context.Context, request *api.AuthorizeRequest) (
	*empty.Empty, error,
) {
	err := ValidateAuthorizeRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Authorize request is not valid")
	}

	namespace := strings.ToLower(request.GetNamespace())
	verb := strings.ToLower(request.GetVerb().String())
	resource := strings.ToLower(request.GetResources().String())
	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace:   namespace,
		Verb:        verb,
		Group:       rbacResourceTypeToGroup[resource],
		Version:     common.RbacPipelinesVersion,
		Resource:    resource,
		Subresource: "",
		Name:        "",
	}
	err = s.resourceManager.IsAuthorized(ctx, resourceAttributes)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	return &empty.Empty{}, nil
}

func ValidateAuthorizeRequest(request *api.AuthorizeRequest) error {
	if request == nil {
		return util.NewInvalidInputError("request object is empty")
	}
	if len(request.Namespace) == 0 {
		return util.NewInvalidInputError("Namespace is empty. Please specify a valid namespace")
	}
	if request.Resources == api.AuthorizeRequest_UNASSIGNED_RESOURCES {
		return util.NewInvalidInputError("Resources not specified. Please specify a valid resources")
	}
	if request.Verb == api.AuthorizeRequest_UNASSIGNED_VERB {
		return util.NewInvalidInputError("Verb not specified. Please specify a valid verb")
	}
	return nil
}

func NewAuthServer(resourceManager *resource.ResourceManager) *AuthServer {
	return &AuthServer{resourceManager: resourceManager}
}
