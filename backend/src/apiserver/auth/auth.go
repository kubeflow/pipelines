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
)

type Authenticator interface {
	GetUserIdentity(ctx context.Context) (string, error)
}

var IdentityHeaderMissingError = util.NewUnauthenticatedError(
	errors.New("Request header error: there is no user identity header."),
	"Request header error: there is no user identity header.",
)

// Make this public for tests to force its re-instantiation
var Authenticators []Authenticator

func GetAuthenticators(tokenReviewClient client.TokenReviewInterface) []Authenticator {
	if Authenticators == nil {
		Authenticators = []Authenticator{
			NewTokenReviewAuthenticator(
				common.AuthorizationBearerTokenHeader,
				common.AuthorizationBearerTokenPrefix,
				[]string{common.TokenReviewAudience},
				tokenReviewClient,
			),
			NewHTTPHeaderAuthenticator(common.GetKubeflowUserIDHeader(), common.GetKubeflowUserIDPrefix()),
		}
	}
	return Authenticators
}
