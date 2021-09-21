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

	authenticator := NewHTTPHeaderAuthenticator(common.GetKubeflowUserIDHeader(), common.GetKubeflowUserIDPrefix())

	userIdentity, err := authenticator.GetUserIdentity(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "user@google.com", userIdentity)
}

func TestHTTPHeaderAuthenticatorError(t *testing.T) {
	md := metadata.New(map[string]string{"no-identity-header": "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewHTTPHeaderAuthenticator(common.GetKubeflowUserIDHeader(), common.GetKubeflowUserIDPrefix())

	_, err := authenticator.GetUserIdentity(ctx)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Request header error: there is no user identity header.")
}

func TestHTTPHeaderAuthenticatorFromHeaderNonGoogle(t *testing.T) {
	header := "Kubeflow-UserID"
	prefix := ""

	md := metadata.New(map[string]string{header: prefix + "user"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	authenticator := NewHTTPHeaderAuthenticator(header, prefix)

	userIdentity, err := authenticator.GetUserIdentity(ctx)
	assert.Nil(t, err)
	assert.Equal(t, "user", userIdentity)
}
