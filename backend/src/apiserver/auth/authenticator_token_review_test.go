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
	"sync/atomic"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		fmt.Sprintf("Failed to find any of '%v' in audience: %v", audience, []string{common.GetTokenReviewAudience()}),
	)
}

func TestTokenReviewAuthenticator_RunScopedTokenUsesSingleReview(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = WithRequestedRunID(ctx, "run-123")
	runAudience := common.TokenAudienceForRun("run-123")
	baseAudience := common.GetTokenReviewAudience()

	fakeClient := &audienceAwareFakeTokenReviewClient{
		tokenAudiences: []string{runAudience},
	}
	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{baseAudience},
		fakeClient,
	)

	userIdentity, err := authenticator.GetUserIdentity(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test", userIdentity)
	assert.Equal(t, int64(1), fakeClient.calls.Load())
	require.Equal(t, []string{baseAudience, runAudience}, fakeClient.lastRequested)

	principal, ok := AuthenticatedPrincipalFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, TokenScopeRun, principal.Scope)
	assert.Equal(t, "run-123", principal.RunID)
	assert.Equal(t, AuthMethodTokenReview, principal.AuthMethod)
}

func TestTokenReviewAuthenticator_BroadTokenWinsWhenBothAudiencesMatch(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = WithRequestedRunID(ctx, "run-123")
	runAudience := common.TokenAudienceForRun("run-123")
	baseAudience := common.GetTokenReviewAudience()

	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{baseAudience},
		&audienceAwareFakeTokenReviewClient{tokenAudiences: []string{baseAudience, runAudience}},
	)

	_, err := authenticator.GetUserIdentity(ctx)
	require.NoError(t, err)

	principal, ok := AuthenticatedPrincipalFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, TokenScopeBroad, principal.Scope)
	assert.Empty(t, principal.RunID)
}

func TestTokenReviewAuthenticator_BroadTokenWorksForRunTargetedRequest(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = WithRequestedRunID(ctx, "run-123")
	baseAudience := common.GetTokenReviewAudience()

	fakeClient := &audienceAwareFakeTokenReviewClient{tokenAudiences: []string{baseAudience}}
	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{baseAudience},
		fakeClient,
	)

	_, err := authenticator.GetUserIdentity(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), fakeClient.calls.Load())

	principal, ok := AuthenticatedPrincipalFromContext(ctx)
	require.True(t, ok)
	assert.Equal(t, TokenScopeBroad, principal.Scope)
}

func TestTokenReviewAuthenticator_RejectsMismatchedRunAudience(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = WithRequestedRunID(ctx, "run-a")

	fakeClient := &audienceAwareFakeTokenReviewClient{
		tokenAudiences: []string{common.TokenAudienceForRun("run-b")},
	}
	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		fakeClient,
	)

	_, err := authenticator.GetUserIdentity(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Failed to authenticate token review")
	assert.Equal(t, int64(1), fakeClient.calls.Load())
}

func TestTokenReviewAuthenticator_TransportErrorDoesNotRetry(t *testing.T) {
	md := metadata.New(map[string]string{common.AuthorizationBearerTokenHeader: common.AuthorizationBearerTokenPrefix + "token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ctx = WithRequestedRunID(ctx, "run-123")

	fakeClient := &audienceAwareFakeTokenReviewClient{failCreate: true}
	authenticator := NewTokenReviewAuthenticator(
		common.AuthorizationBearerTokenHeader,
		common.AuthorizationBearerTokenPrefix,
		[]string{common.GetTokenReviewAudience()},
		fakeClient,
	)

	_, err := authenticator.GetUserIdentity(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to review the token provided")
	assert.Equal(t, int64(1), fakeClient.calls.Load())
}

func TestEnforceAuthenticatedRunScope(t *testing.T) {
	t.Run("allows broad principal", func(t *testing.T) {
		ctx := WithRequestedRunID(context.Background(), "run-a")
		storeAuthenticatedPrincipal(ctx, AuthenticatedPrincipal{
			Username:   "sa",
			AuthMethod: AuthMethodTokenReview,
			Scope:      TokenScopeBroad,
		})
		assert.NoError(t, EnforceAuthenticatedRunScope(ctx, "run-a"))
	})

	t.Run("allows matching run principal", func(t *testing.T) {
		ctx := WithRequestedRunID(context.Background(), "run-a")
		storeAuthenticatedPrincipal(ctx, AuthenticatedPrincipal{
			Username:   "sa",
			AuthMethod: AuthMethodTokenReview,
			Scope:      TokenScopeRun,
			RunID:      "run-a",
		})
		assert.NoError(t, EnforceAuthenticatedRunScope(ctx, "run-a"))
	})

	t.Run("rejects mismatched run principal", func(t *testing.T) {
		ctx := WithRequestedRunID(context.Background(), "run-a")
		storeAuthenticatedPrincipal(ctx, AuthenticatedPrincipal{
			Username:   "sa",
			AuthMethod: AuthMethodTokenReview,
			Scope:      TokenScopeRun,
			RunID:      "run-b",
		})
		err := EnforceAuthenticatedRunScope(ctx, "run-a")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bound to a different pipeline run")
	})
}

type audienceAwareFakeTokenReviewClient struct {
	tokenAudiences []string
	failCreate     bool
	calls          atomic.Int64
	lastRequested  []string
}

func (f *audienceAwareFakeTokenReviewClient) Create(_ context.Context, review *authv1.TokenReview, _ v1.CreateOptions) (*authv1.TokenReview, error) {
	f.calls.Add(1)
	f.lastRequested = append([]string(nil), review.Spec.Audiences...)
	if f.failCreate {
		return nil, fmt.Errorf("tokenreview unavailable")
	}
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
		[]string{common.GetTokenReviewAudience(), "pipelines.kubeflow.org/runs/run-1"},
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
