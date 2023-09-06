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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestHTTPHeaderAuthenticator(t *testing.T) {
	md := metadata.New(map[string]string{common.GoogleIAPUserIdentityHeader: common.GoogleIAPUserIdentityPrefix + "user@google.com"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewHTTPHeaderAuthenticator(
		common.GetKubeflowUserIDHeader(),
		common.GetKubeflowUserIDPrefix(),
		common.GetKubeflowGroupsHeader(),
		common.GetExperimentalGroupsSupport(),
	)

	userIdentity, err := authenticator.GetUserIdentity(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "user@google.com", userIdentity)
}

func TestHTTPHeaderAuthenticatorError(t *testing.T) {
	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewHTTPHeaderAuthenticator(
		common.GetKubeflowUserIDHeader(),
		common.GetKubeflowUserIDPrefix(),
		common.GetKubeflowGroupsHeader(),
		common.GetExperimentalGroupsSupport(),
	)

	_, err := authenticator.GetUserIdentity(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Request header error: there is no user identity header.")
}

func TestHTTPHeaderAuthenticatorFromHeaderNonGoogle(t *testing.T) {
	header := "Kubeflow-UserID"
	prefix := ""
	groups := ""

	md := metadata.New(map[string]string{header: prefix + "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewHTTPHeaderAuthenticator(header, prefix, groups, false)

	userIdentity, err := authenticator.GetUserIdentity(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "user", userIdentity)
}

func TestHTTPGroupHeaderAuthenticatorFromHeaderNonGoogle(t *testing.T) {
	userIDHeader := "Kubeflow-UserID"
	groupsHeader := "Kubeflow-Groups"
	groups := "one,two"
	md := metadata.New(map[string]string{groupsHeader: groups})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewHTTPHeaderAuthenticator(userIDHeader, "", groupsHeader, true)
	userGroups, err := authenticator.GetUserGroups(ctx)
	assert.Nil(t, err)
	assert.NotEmpty(t, userGroups)
	assert.Len(t, userGroups, 2)
	assert.Equal(t, userGroups[0], "one")
	assert.Equal(t, userGroups[1], "two")
}

func TestHTTPGroupHeaderExperimentalFlagDisableAuthenticatorFromHeaderNonGoogle(t *testing.T) {
	userIDHeader := "Kubeflow-UserID"
	groupsHeader := "Kubeflow-Groups"
	groups := "one,two"
	md := metadata.New(map[string]string{groupsHeader: groups})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewHTTPHeaderAuthenticator(userIDHeader, "", groupsHeader, false)
	userGroups, err := authenticator.GetUserGroups(ctx)
	assert.Empty(t, userGroups)
	assert.Nil(t, err)
}
