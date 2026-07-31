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
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		fmt.Sprintf("Failed to find all of '%v' in audience: %v", audience, []string{common.GetTokenReviewAudience()}),
	)
}

func TestTokenReviewAuthenticator_UsesExpectedAudienceFromContext(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	runAudience := common.TokenAudienceForRun("run-123")
	ctx = WithExpectedTokenAudiences(ctx, []string{runAudience})

	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		&audienceAwareFakeTokenReviewClient{tokenAudiences: []string{runAudience}},
	)

	userIdentity, err := authenticator.GetUserIdentity(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "test", userIdentity)
}

func TestTokenReviewAuthenticator_RejectsMismatchedRunAudience(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = WithExpectedTokenAudiences(ctx, []string{common.TokenAudienceForRun("run-a")})

	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		&audienceAwareFakeTokenReviewClient{tokenAudiences: []string{common.TokenAudienceForRun("run-b")}},
	)

	_, err := authenticator.GetUserIdentity(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to authenticate token review")
}

type audienceAwareFakeTokenReviewClient struct {
	tokenAudiences []string
}

func (f *audienceAwareFakeTokenReviewClient) Create(_ context.Context, review *authv1.TokenReview, _ v1.CreateOptions) (*authv1.TokenReview, error) {
	requested := review.Spec.Audiences
	matched := make([]string, 0, len(requested))
	tokenSet := make(map[string]struct{}, len(f.tokenAudiences))
	for _, audience := range f.tokenAudiences {
		tokenSet[audience] = struct{}{}
	}
	for _, audience := range requested {
		if _, ok := tokenSet[audience]; ok {
			matched = append(matched, audience)
		}
	}
	if len(matched) == 0 {
		return &authv1.TokenReview{Status: authv1.TokenReviewStatus{
			Authenticated: false,
			Error:         "audience mismatch",
		}}, nil
	}
	return &authv1.TokenReview{Status: authv1.TokenReviewStatus{
		Authenticated: true,
		User:          authv1.UserInfo{Username: "test"},
		Audiences:     matched,
	}}, nil
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

	audienceEnsured := authenticator.ensureAudience(
		[]string{common.GetTokenReviewAudience()},
		[]string{common.GetTokenReviewAudience()},
	)
	assert.True(t, audienceEnsured)
}

func TestTokenReviewAuthenticator_ensureAudienceFail(t *testing.T) {
	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		client.NewFakeTokenReviewClient(),
	)

	audienceEnsured := authenticator.ensureAudience(
		[]string{common.GetTokenReviewAudience()},
		[]string{"request-audience"},
	)
	assert.False(t, audienceEnsured)
}
