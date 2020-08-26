package server

import (
	"context"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestAuthorizeRequest_SingleUserMode(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}
	clients.KfamClientFake = client.NewFakeKFAMClientUnauthorized()

	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "ns1",
		Resources: api.AuthorizeRequest_VIEWERS,
		Verb:      api.AuthorizeRequest_GET,
	}

	_, err := authServer.Authorize(ctx, request)
	// Authz is completely skipped without checking anything.
	assert.Nil(t, err)
}

func TestAuthorizeRequest_InvalidRequest(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}

	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "",
		Resources: api.AuthorizeRequest_UNASSIGNED_RESOURCES,
		Verb:      api.AuthorizeRequest_UNASSIGNED_VERB,
	}

	_, err := authServer.Authorize(ctx, request)
	assert.Error(t, err)
	assert.EqualError(t, err, "Authorize request is not valid: Invalid input error: Namespace is empty. Please specify a valid namespace.")
}

func TestAuthorizeRequest_Authorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: "accounts.google.com:user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "ns1",
		Resources: api.AuthorizeRequest_VIEWERS,
		Verb:      api.AuthorizeRequest_GET,
	}

	_, err := authServer.Authorize(ctx, request)
	assert.Nil(t, err)
}

func TestAuthorizeRequest_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, _ := initWithExperiment_KFAM_Unauthorized(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: "accounts.google.com:user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "ns1",
		Resources: api.AuthorizeRequest_VIEWERS,
		Verb:      api.AuthorizeRequest_GET,
	}

	_, err := authServer.Authorize(ctx, request)
	assert.Error(t, err)
	assert.EqualError(t, err, "Failed to authorize the request: Failed to authorize namespace: BadRequestError: Unauthorized access for user@google.com to namespace ns1: Unauthorized access for user@google.com to namespace ns1")
}

func TestAuthorizeRequest_EmptyUserIdPrefix(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")
	viper.Set(common.KubeflowUserIDPrefix, "")
	defer viper.Set(common.KubeflowUserIDPrefix, common.GoogleIAPUserIdentityPrefix)

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}

	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "ns1",
		Resources: api.AuthorizeRequest_VIEWERS,
		Verb:      api.AuthorizeRequest_GET,
	}

	_, err := authServer.Authorize(ctx, request)
	assert.Nil(t, err)
}
