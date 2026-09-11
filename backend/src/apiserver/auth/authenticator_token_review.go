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

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TokenReviewAuthenticator struct {
	// tokenHeader in which the authenticator expects to find the ServiceAccountToken
	tokenHeader string
	// tokenPrefix is the prefix encountered before the token
	tokenPrefix string
	// audiences the authenticator identifies as
	audiences []string
	// client to use to do TokenReviews
	client client.TokenReviewInterface
}

func NewTokenReviewAuthenticator(tokenHeader, tokenPrefix string, audiences []string, tokenReviewClient client.TokenReviewInterface) *TokenReviewAuthenticator {
	return &TokenReviewAuthenticator{
		tokenHeader: tokenHeader,
		tokenPrefix: tokenPrefix,
		audiences:   audiences,
		client:      tokenReviewClient,
	}
}

func (tra *TokenReviewAuthenticator) GetUserIdentity(ctx context.Context) (string, error) {
	token, err := singlePrefixedHeaderFromMetadata(ctx, tra.tokenHeader, tra.tokenPrefix)
	if err != nil {
		return "", err
	}

	requestedAudiences := append([]string(nil), tra.audiences...)
	requestedRunID := ""
	if runID, ok := RequestedRunIDFromContext(ctx); ok {
		requestedRunID = runID
		requestedAudiences = uniqueAudiences(append(requestedAudiences, common.TokenAudienceForRun(runID)))
	}

	userInfo, matchedAudiences, err := tra.doTokenReview(ctx, token, requestedAudiences)
	if err != nil {
		return "", util.Wrap(err, "Authentication failure")
	}

	principal := classifyTokenReviewPrincipal(userInfo.Username, tra.audiences, matchedAudiences, requestedRunID)
	storeAuthenticatedPrincipal(ctx, principal)
	return principal.Username, nil
}

// ensureAudience reports whether any requested audience appears in the TokenReview
// status audiences (Kubernetes any-match semantics).
func (tra *TokenReviewAuthenticator) ensureAudience(requested []string, statusAudiences []string) bool {
	return len(intersectAudiences(requested, statusAudiences)) > 0
}

func (tra *TokenReviewAuthenticator) doTokenReview(ctx context.Context, userIdentity string, audiences []string) (*authv1.UserInfo, []string, error) {
	review, err := tra.client.Create(
		ctx,
		&authv1.TokenReview{
			Spec: authv1.TokenReviewSpec{
				Token:     userIdentity,
				Audiences: audiences,
			},
		},
		v1.CreateOptions{},
	)
	if err != nil {
		return nil, nil, util.NewUnauthenticatedError(err, "Request header error: Failed to review the token provided")
	}

	if !review.Status.Authenticated {
		return nil, nil, util.NewUnauthenticatedError(
			errors.New("Failed to authenticate token review"),
			"Review.Status.Authenticated is false. Error %s",
			review.Status.Error,
		)
	}
	matchedAudiences := intersectAudiences(audiences, review.Status.Audiences)
	if len(matchedAudiences) == 0 {
		return nil, nil, util.NewUnauthenticatedError(
			errors.New("Failed to authenticate token review"),
			"Failed to find any of '%v' in audience: %v",
			audiences,
			review.Status.Audiences,
		)
	}

	return &review.Status.User, matchedAudiences, nil
}

func classifyTokenReviewPrincipal(
	username string,
	baseAudiences []string,
	matchedAudiences []string,
	requestedRunID string,
) AuthenticatedPrincipal {
	principal := AuthenticatedPrincipal{
		Username:   username,
		AuthMethod: AuthMethodTokenReview,
		Scope:      TokenScopeBroad,
	}
	if len(intersectAudiences(baseAudiences, matchedAudiences)) > 0 {
		return principal
	}
	if requestedRunID == "" {
		return principal
	}
	runAudience := common.TokenAudienceForRun(requestedRunID)
	for _, audience := range matchedAudiences {
		if audience == runAudience {
			principal.Scope = TokenScopeRun
			principal.RunID = requestedRunID
			return principal
		}
	}
	return principal
}

func uniqueAudiences(audiences []string) []string {
	if len(audiences) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(audiences))
	result := make([]string, 0, len(audiences))
	for _, audience := range audiences {
		if audience == "" {
			continue
		}
		if _, ok := seen[audience]; ok {
			continue
		}
		seen[audience] = struct{}{}
		result = append(result, audience)
	}
	return result
}

func intersectAudiences(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	rightSet := make(map[string]struct{}, len(right))
	for _, audience := range right {
		rightSet[audience] = struct{}{}
	}
	result := make([]string, 0, len(left))
	seen := make(map[string]struct{}, len(left))
	for _, audience := range left {
		if _, ok := rightSet[audience]; !ok {
			continue
		}
		if _, ok := seen[audience]; ok {
			continue
		}
		seen[audience] = struct{}{}
		result = append(result, audience)
	}
	return result
}
