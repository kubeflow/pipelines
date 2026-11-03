// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	authv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type runtimeAuthRunStore struct {
	storage.RunStoreInterface
	calls    int
	hydrated bool
}

func (s *runtimeAuthRunStore) GetRun(id string, hydrate bool) (*model.Run, error) {
	s.calls++
	s.hydrated = hydrate
	return &model.Run{UUID: id, Namespace: "ns1", K8SName: "workflow-1"}, nil
}

type runtimeTokenReviewer struct {
	calls         int
	audiences     []string
	username      string
	authenticated bool
	request       *authv1.TokenReview
}

func (r *runtimeTokenReviewer) Create(_ context.Context, request *authv1.TokenReview, _ metav1.CreateOptions) (*authv1.TokenReview, error) {
	r.calls++
	r.request = request.DeepCopy()
	var matched []string
	for _, requested := range request.Spec.Audiences {
		for _, audience := range r.audiences {
			if requested == audience {
				matched = append(matched, audience)
			}
		}
	}
	return &authv1.TokenReview{Status: authv1.TokenReviewStatus{
		Authenticated: r.authenticated && len(matched) > 0,
		Audiences:     matched, User: authv1.UserInfo{Username: r.username},
	}}, nil
}

func TestAuthenticateRuntimeTaskRequiresRunScopedTokenInSingleUser(t *testing.T) {
	previous := viper.Get(common.MultiUserMode)
	viper.Set(common.MultiUserMode, false)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, previous) })
	require.False(t, common.IsMultiUserMode())
	runAudience := common.TokenAudienceForRun("run-1")
	baseAudience := common.GetTokenReviewAudience()
	for _, test := range []struct {
		name                                 string
		audiences                            []string
		username                             string
		headerOnly, unauthenticated, allowed bool
	}{
		{name: "run service account", audiences: []string{runAudience}, username: "system:serviceaccount:ns1:pipeline-runner", allowed: true},
		{name: "broad API token", audiences: []string{baseAudience}, username: "system:serviceaccount:ns1:pipeline-runner"},
		{name: "broad and run audiences", audiences: []string{baseAudience, runAudience}, username: "system:serviceaccount:ns1:pipeline-runner"},
		{name: "other run", audiences: []string{common.TokenAudienceForRun("run-2")}, username: "system:serviceaccount:ns1:pipeline-runner"},
		{name: "other namespace", audiences: []string{runAudience}, username: "system:serviceaccount:ns10:pipeline-runner"},
		{name: "user identity", audiences: []string{runAudience}, username: "user@example.com"},
		{name: "identity header only", headerOnly: true},
		{name: "failed token review", audiences: []string{runAudience}, username: "system:serviceaccount:ns1:pipeline-runner", unauthenticated: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &runtimeAuthRunStore{}
			reviewer := &runtimeTokenReviewer{audiences: test.audiences, username: test.username, authenticated: !test.unauthenticated}
			manager := &ResourceManager{runStore: store, tokenReviewClient: reviewer}
			headers := metadata.Pairs("kubeflow-userid", "trusted-user@example.com")
			if !test.headerOnly {
				headers.Set(common.AuthorizationBearerTokenHeader, common.AuthorizationBearerTokenPrefix+"runtime-token")
			}
			err := manager.AuthenticateRuntimeTask(metadata.NewIncomingContext(context.Background(), headers), "run-1")
			if test.allowed {
				require.NoError(t, err)
				require.Equal(t, 1, store.calls)
				require.False(t, store.hydrated, "runtime authorization needs only run namespace")
			} else {
				require.Error(t, err)
				require.True(t, util.IsUserErrorCodeMatch(err, codes.Unauthenticated), "got %v", err)
			}
			if test.headerOnly {
				require.Nil(t, reviewer.request, "public identity headers must not opt into the runtime protocol")
				require.Zero(t, store.calls)
			} else {
				require.NotNil(t, reviewer.request)
				require.Equal(t, []string{baseAudience, runAudience}, reviewer.request.Spec.Audiences)
			}
		})
	}
}

func TestAuthenticateRuntimeTaskFailsWithoutTokenReview(t *testing.T) {
	manager := &ResourceManager{runStore: &runtimeAuthRunStore{}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(common.AuthorizationBearerTokenHeader, common.AuthorizationBearerTokenPrefix+"runtime-token"))
	err := manager.AuthenticateRuntimeTask(ctx, "run-1")
	require.True(t, util.IsUserErrorCodeMatch(err, codes.Unauthenticated), "got %v", err)
}

// The public header authenticator is deliberately configured in these tests:
// a verified runtime token must still be the sole source of the SAR identity.
type runtimeSubjectAccessReviewer struct {
	status  authorizationv1.SubjectAccessReviewStatus
	request *authorizationv1.SubjectAccessReview
	calls   int
}

func (r *runtimeSubjectAccessReviewer) Create(_ context.Context, request *authorizationv1.SubjectAccessReview, _ metav1.CreateOptions) (*authorizationv1.SubjectAccessReview, error) {
	r.calls++
	r.request = request.DeepCopy()
	return &authorizationv1.SubjectAccessReview{Status: r.status}, nil
}

func TestAuthorizeRuntimeTaskUsesVerifiedServiceAccount(t *testing.T) {
	previousMode, previousSharedRead := viper.Get(common.MultiUserMode), viper.Get(common.MultiUserModeSharedReadAccess)
	viper.Set(common.MultiUserMode, true)
	viper.Set(common.MultiUserModeSharedReadAccess, false)
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, previousMode)
		viper.Set(common.MultiUserModeSharedReadAccess, previousSharedRead)
	})
	for _, test := range []struct {
		name   string
		status authorizationv1.SubjectAccessReviewStatus
		code   codes.Code
	}{
		{name: "allowed", status: authorizationv1.SubjectAccessReviewStatus{Allowed: true}, code: codes.OK},
		{name: "denied", status: authorizationv1.SubjectAccessReviewStatus{Reason: "runtime SA denied"}, code: codes.PermissionDenied},
		{name: "incomplete evaluation", status: authorizationv1.SubjectAccessReviewStatus{Allowed: true, EvaluationError: "authorizer unavailable"}, code: codes.Internal},
	} {
		t.Run(test.name, func(t *testing.T) {
			const runtimeIdentity = "system:serviceaccount:ns1:pipeline-runner"
			store := &runtimeAuthRunStore{}
			reviewer := &runtimeTokenReviewer{audiences: []string{common.TokenAudienceForRun("run-1")}, username: runtimeIdentity, authenticated: true}
			sar := &runtimeSubjectAccessReviewer{status: test.status}
			manager := &ResourceManager{
				runStore: store, tokenReviewClient: reviewer, subjectAccessReviewClient: sar,
				authenticators: []auth.Authenticator{auth.NewHTTPHeaderAuthenticator("kubeflow-userid", "")},
			}
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				"kubeflow-userid", "cluster-admin@example.com",
				common.AuthorizationBearerTokenHeader, common.AuthorizationBearerTokenPrefix+"runtime-token",
			))
			err := manager.AuthorizeRuntimeTask(ctx, "run-1", common.RbacResourceVerbUpdate)
			if test.code == codes.OK {
				require.NoError(t, err)
			} else {
				require.True(t, util.IsUserErrorCodeMatch(err, test.code), "got %v", err)
			}
			require.Equal(t, 1, reviewer.calls)
			require.Equal(t, 1, store.calls)
			require.False(t, store.hydrated)
			require.Equal(t, 1, sar.calls)
			require.Equal(t, runtimeIdentity, sar.request.Spec.User)
			require.Equal(t, &authorizationv1.ResourceAttributes{
				Namespace: "ns1", Name: "workflow-1", Verb: common.RbacResourceVerbUpdate,
				Group: common.RbacPipelinesGroup, Version: common.RbacPipelinesVersion, Resource: common.RbacResourceTypeRuns,
			}, sar.request.Spec.ResourceAttributes)
		})
	}
}

func TestAuthorizeRuntimeTaskAuthenticatesBeforeAuthorizationBypass(t *testing.T) {
	previousMode, previousSharedRead := viper.Get(common.MultiUserMode), viper.Get(common.MultiUserModeSharedReadAccess)
	t.Cleanup(func() {
		viper.Set(common.MultiUserMode, previousMode)
		viper.Set(common.MultiUserModeSharedReadAccess, previousSharedRead)
	})
	for _, test := range []struct {
		name      string
		multiUser bool
		verb      string
	}{
		{name: "single user write", verb: common.RbacResourceVerbUpdate},
		{name: "shared read", multiUser: true, verb: common.RbacResourceVerbGet},
	} {
		t.Run(test.name, func(t *testing.T) {
			viper.Set(common.MultiUserMode, test.multiUser)
			viper.Set(common.MultiUserModeSharedReadAccess, true)
			reviewer := &runtimeTokenReviewer{audiences: []string{common.TokenAudienceForRun("run-1")}, username: "system:serviceaccount:ns1:pipeline-runner", authenticated: true}
			manager := &ResourceManager{runStore: &runtimeAuthRunStore{}, tokenReviewClient: reviewer}
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
				common.AuthorizationBearerTokenHeader, common.AuthorizationBearerTokenPrefix+"runtime-token",
			))
			require.NoError(t, manager.AuthorizeRuntimeTask(ctx, "run-1", test.verb))
			require.Equal(t, 1, reviewer.calls)
			headerOnly := metadata.NewIncomingContext(context.Background(), metadata.Pairs("kubeflow-userid", "cluster-admin@example.com"))
			err := manager.AuthorizeRuntimeTask(headerOnly, "run-1", test.verb)
			require.True(t, util.IsUserErrorCodeMatch(err, codes.Unauthenticated), "got %v", err)
			require.Equal(t, 1, reviewer.calls, "missing token must fail before TokenReview")
		})
	}
}
