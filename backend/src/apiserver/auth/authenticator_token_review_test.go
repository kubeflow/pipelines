// Copyright 2021 Arrikto Inc.
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

package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestTokenReviewAuthenticatorAuthenticated(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		client.NewFakeTokenReviewClient(),
	)

	userIdentity, err := authenticator.GetUserIdentity(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "test", userIdentity)
}

func TestTokenReviewAuthenticatorAuthenticatedWrongAudience(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	audience := []string{"expected-audience"}

	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		audience,
		client.NewFakeTokenReviewClient(),
	)

	_, err := authenticator.GetUserIdentity(ctx)
	assert.NotNil(t, err)
	assert.Contains(
		t,
		err.Error(),
		fmt.Sprintf("Failed to find all of '%v' in audience: %v", authenticator.audiences, []string{common.GetTokenReviewAudience()}),
	)
}

func TestTokenReviewAuthenticatorUnauthenticated(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		client.NewFakeTokenReviewClientUnauthenticated(),
	)

	_, err := authenticator.GetUserIdentity(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to authenticate token review")
}

func TestTokenReviewAuthenticatorError(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		client.NewFakeTokenReviewClientError(),
	)

	_, err := authenticator.GetUserIdentity(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Request header error: Failed to review the token provided")
}

func TestTokenReviewAuthenticator_ensureAudience(t *testing.T) {
	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		client.NewFakeTokenReviewClient(),
	)

	audienceEnsured := authenticator.ensureAudience([]string{common.GetTokenReviewAudience()})
	assert.True(t, audienceEnsured)
}

func TestTokenReviewAuthenticator_ensureAudienceFail(t *testing.T) {
	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		client.NewFakeTokenReviewClient(),
	)

	audienceEnsured := authenticator.ensureAudience([]string{"request-audience"})
	assert.False(t, audienceEnsured)
}
