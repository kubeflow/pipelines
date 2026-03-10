// Copyright 2026 The Kubeflow Authors
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

package mlflow

import (
	"context"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	authorizationv1 "k8s.io/api/authorization/v1"
)

func newCtxWithUserHeader(user string) context.Context {
	md := metadata.Pairs(strings.ToLower(common.GetKubeflowUserIDHeader()), common.GetKubeflowUserIDPrefix()+user)
	return metadata.NewIncomingContext(context.Background(), md)
}

// fakeIsAuthorizedDenied always returns a permission-denied error.
func fakeIsAuthorizedDenied(_ context.Context, _ *authorizationv1.ResourceAttributes) error {
	return &fakePermDeniedError{}
}

type fakePermDeniedError struct{}

func (e *fakePermDeniedError) Error() string { return "not authorized" }

// fakeIsAuthorizedAllowed always returns nil (authorized).
func fakeIsAuthorizedAllowed(_ context.Context, _ *authorizationv1.ResourceAttributes) error {
	return nil
}

func TestAuthorizeExperimentAction(t *testing.T) {
	t.Run("standalone mode auth disabled skips check", func(t *testing.T) {
		viper.Set(common.MultiUserMode, "false")
		viper.Set(AuthorizationEnabledConfigKey, "false")
		t.Cleanup(func() {
			viper.Set(common.MultiUserMode, "")
			viper.Set(AuthorizationEnabledConfigKey, "")
		})
		require.NoError(t, AuthorizeExperimentAction(context.Background(), "ns1", nil, fakeIsAuthorizedAllowed))
	})

	t.Run("standalone mode auth enabled requires forwarded user header", func(t *testing.T) {
		viper.Set(common.MultiUserMode, "false")
		viper.Set(AuthorizationEnabledConfigKey, "true")
		t.Cleanup(func() {
			viper.Set(common.MultiUserMode, "")
			viper.Set(AuthorizationEnabledConfigKey, "")
		})
		err := AuthorizeExperimentAction(context.Background(), "ns1", nil, fakeIsAuthorizedAllowed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "User identity")
	})

	t.Run("standalone mode auth enabled fails when SAR denies", func(t *testing.T) {
		viper.Set(common.MultiUserMode, "false")
		viper.Set(AuthorizationEnabledConfigKey, "true")
		t.Cleanup(func() {
			viper.Set(common.MultiUserMode, "")
			viper.Set(AuthorizationEnabledConfigKey, "")
		})
		sarClient := client.NewFakeSubjectAccessReviewClientUnauthorized()
		err := AuthorizeExperimentAction(newCtxWithUserHeader("user@example.com"), "ns1", sarClient, fakeIsAuthorizedAllowed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not authorized")
	})

	t.Run("multi-user mode uses IsAuthorized path", func(t *testing.T) {
		viper.Set(common.MultiUserMode, "true")
		viper.Set(AuthorizationEnabledConfigKey, "false")
		t.Cleanup(func() {
			viper.Set(common.MultiUserMode, "")
			viper.Set(AuthorizationEnabledConfigKey, "")
		})
		err := AuthorizeExperimentAction(newCtxWithUserHeader("user@example.com"), "ns1", nil, fakeIsAuthorizedDenied)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not authorized")
	})
}
