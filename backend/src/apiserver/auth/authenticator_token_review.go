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

	// Prefer request-scoped audiences (for example run-scoped projected tokens)
	// when the caller attached them. Fall back to the configured base audience
	// so non-runtime SA tokens continue to work under RBAC.
	if expectedAudiences, ok := ExpectedTokenAudiences(ctx); ok {
		userInfo, reviewErr := tra.doTokenReview(ctx, token, expectedAudiences)
		if reviewErr == nil {
			return userInfo.Username, nil
		}
		if !audiencesEqual(expectedAudiences, tra.audiences) {
			userInfo, fallbackErr := tra.doTokenReview(ctx, token, tra.audiences)
			if fallbackErr == nil {
				return userInfo.Username, nil
			}
		}
		return "", util.Wrap(reviewErr, "Authentication failure")
	}

	userInfo, err := tra.doTokenReview(ctx, token, tra.audiences)
	if err != nil {
		return "", util.Wrap(err, "Authentication failure")
	}
	return userInfo.Username, nil
}

// ensureAudience makes sure all audience of the authenticator is found in the provided audience list.
func (tra *TokenReviewAuthenticator) ensureAudience(expected []string, audience []string) bool {
	// Create a set (map) to check fast whether something is part of the list
	audienceSet := make(map[string]struct{}, len(audience))
	for _, a := range audience {
		audienceSet[a] = struct{}{}
	}

	// Iterate through the expected audiences and check if they are part of the provided list
	for _, a := range expected {
		if _, ok := audienceSet[a]; !ok {
			return false
		}
	}
	return true
}

func (tra *TokenReviewAuthenticator) doTokenReview(ctx context.Context, userIdentity string, audiences []string) (*authv1.UserInfo, error) {
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
		return nil, util.NewUnauthenticatedError(err, "Request header error: Failed to review the token provided")
	}

	if !review.Status.Authenticated {
		return nil, util.NewUnauthenticatedError(
			errors.New("Failed to authenticate token review"),
			"Review.Status.Authenticated is false. Error %s",
			review.Status.Error,
		)
	}
	if !tra.ensureAudience(audiences, review.Status.Audiences) {
		return nil, util.NewUnauthenticatedError(
			errors.New("Failed to authenticate token review"),
			"Failed to find all of '%v' in audience: %v",
			audiences,
			review.Status.Audiences,
		)
	}

	return &review.Status.User, nil
}

func audiencesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
