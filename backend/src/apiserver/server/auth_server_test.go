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
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func TestAuthorizeRequest_SingleUserMode(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}
	clients.SubjectAccessReviewClientFake = client.NewFakeSubjectAccessReviewClientUnauthorized()

	md := metadata.New(map[string]string{})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "ns1",
		Resources: api.AuthorizeRequest_VIEWERS,
		Verb:      api.AuthorizeRequest_GET,
	}

	_, err := authServer.AuthorizeV1(ctx, request)
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

	_, err := authServer.AuthorizeV1(ctx, request)
	assert.Error(t, err)
	assert.EqualError(t, err, "Authorize request is not valid: Invalid input error: Namespace is empty. Please specify a valid namespace")
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

	_, err := authServer.AuthorizeV1(ctx, request)
	assert.Nil(t, err)
}

func TestAuthorizeRequest_Unauthorized(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, _ := initWithExperiment_SubjectAccessReview_Unauthorized(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}

	userIdentity := "user@google.com"
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + userIdentity})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "ns1",
		Resources: api.AuthorizeRequest_VIEWERS,
		Verb:      api.AuthorizeRequest_GET,
	}

	_, err := authServer.AuthorizeV1(ctx, request)
	assert.Error(t, err)

	resourceAttributes := &authorizationv1.ResourceAttributes{
		Namespace: "ns1",
		Verb:      common.RbacResourceVerbGet,
		Group:     common.RbacKubeflowGroup,
		Version:   common.RbacPipelinesVersion,
		Resource:  common.RbacResourceTypeViewers,
	}
	assert.EqualError(t, err, wrapFailedAuthzRequestError(getPermissionDeniedError(userIdentity, resourceAttributes)).Error())
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

	_, err := authServer.AuthorizeV1(ctx, request)
	assert.Nil(t, err)
}

func TestAuthorizeRequest_Unauthenticated(t *testing.T) {
	viper.Set(common.MultiUserMode, "true")
	defer viper.Set(common.MultiUserMode, "false")

	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	authServer := AuthServer{resourceManager: manager}

	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	request := &api.AuthorizeRequest{
		Namespace: "ns1",
		Resources: api.AuthorizeRequest_VIEWERS,
		Verb:      api.AuthorizeRequest_GET,
	}

	_, err := authServer.AuthorizeV1(ctx, request)
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		"there is no user identity header",
	)
}
