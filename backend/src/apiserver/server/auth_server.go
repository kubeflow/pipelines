package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type AuthServer struct {
	resourceManager *resource.ResourceManager
}

func (s *AuthServer) Authorize(ctx context.Context, request *api.AuthorizeRequest) (
	*empty.Empty, error) {
	err := ValidateAuthorizeRequest(request)
	if err != nil {
		return nil, util.Wrap(err, "Authorize request is not valid")
	}

	// TODO: when KFP changes authorization implementation to have more
	// granularity, we need to start using resources and verb info in the
	// request.
	err = CanAccessNamespace(s.resourceManager, ctx, request.Namespace)
	if err != nil {
		return nil, util.Wrap(err, "Failed to authorize the request")
	}

	return &empty.Empty{}, nil
}

func ValidateAuthorizeRequest(request *api.AuthorizeRequest) error {
	if request == nil {
		return util.NewInvalidInputError("request object is empty.")
	}
	if len(request.Namespace) == 0 {
		return util.NewInvalidInputError("Namespace is empty. Please specify a valid namespace.")
	}
	if request.Resources == api.AuthorizeRequest_UNASSIGNED_RESOURCES {
		return util.NewInvalidInputError("Resources not specified. Please specify a valid resources.")
	}
	if request.Verb == api.AuthorizeRequest_UNASSIGNED_VERB {
		return util.NewInvalidInputError("Verb not specified. Please specify a valid verb.")
	}
	return nil
}

func NewAuthServer(resourceManager *resource.ResourceManager) *AuthServer {
	return &AuthServer{resourceManager: resourceManager}
}
