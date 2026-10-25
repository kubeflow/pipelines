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

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type taskTokenReview struct{ invalid, wrongAudience bool }

func (c taskTokenReview) Create(_ context.Context, review *authnv1.TokenReview, _ metav1.CreateOptions) (*authnv1.TokenReview, error) {
	audience := review.Spec.Audiences
	if c.wrongAudience {
		audience = []string{"other"}
	}
	return &authnv1.TokenReview{Status: authnv1.TokenReviewStatus{Authenticated: !c.invalid, Audiences: audience, User: authnv1.UserInfo{Username: "system:serviceaccount:ns1:custom-runner"}}}, nil
}

type taskAccessReview struct {
	deny, evaluationError, transportError bool
	reviews                               []*authzv1.SubjectAccessReview
}

func (c *taskAccessReview) Create(_ context.Context, review *authzv1.SubjectAccessReview, _ metav1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	c.reviews = append(c.reviews, review.DeepCopy())
	if c.transportError {
		return nil, errors.New("unavailable")
	}
	evaluation := ""
	if c.evaluationError {
		evaluation = "failed to evaluate"
	}
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{Allowed: !c.deny, EvaluationError: evaluation}}, nil
}

func TestCreateTaskAuthorization(t *testing.T) {
	for _, name := range []string{"anonymous", "invalid", "wrong-audience", "denied", "evaluation-error", "SAR-unavailable", "authorized", "single-user"} {
		t.Run(name, func(t *testing.T) {
			viper.Set(common.MultiUserMode, name != "single-user")
			clients, _, run := initWithOneTimeRun(t)
			defer clients.Close()
			t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
			reviews := &taskAccessReview{deny: name == "denied", evaluationError: name == "evaluation-error", transportError: name == "SAR-unavailable"}
			clients.SubjectAccessReviewClientFake = reviews
			clients.AuthenticatorsFake = []auth.Authenticator{auth.NewTokenReviewAuthenticator("authorization", "Bearer ", []string{common.GetTokenReviewAudience()}, taskTokenReview{invalid: name == "invalid", wrongAudience: name == "wrong-audience"})}
			manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{})
			server := NewTaskServer(manager)
			ctx := context.Background()
			if name != "anonymous" {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer synthetic"))
			}
			result, err := server.CreateTaskV1(ctx, &api.CreateTaskRequest{Task: &api.Task{PipelineName: "pipeline/test", RunId: run.UUID, MlmdExecutionID: "42", Fingerprint: "fingerprint", CreatedAt: timestamppb.New(time.Unix(1, 0))}})
			listed, listErr := server.ListTasksV1(context.Background(), &api.ListTasksRequest{})
			require.NoError(t, listErr)
			if name == "authorized" || name == "single-user" {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, listed.Tasks, 1)
			} else {
				require.Error(t, err)
				require.Nil(t, result)
				require.Empty(t, listed.Tasks)
			}
			if name == "authorized" {
				require.Len(t, reviews.reviews, 1)
				review := reviews.reviews[0]
				require.Equal(t, "system:serviceaccount:ns1:custom-runner", review.Spec.User)
				require.Equal(t, "createTask", review.Spec.ResourceAttributes.Verb)
				require.Equal(t, run.Namespace, review.Spec.ResourceAttributes.Namespace)
				require.Equal(t, run.K8SName, review.Spec.ResourceAttributes.Name)
				require.Equal(t, "runs", review.Spec.ResourceAttributes.Resource)
			}
		})
	}
}

func TestCreateTaskHTTPAuthorization(t *testing.T) {
	for _, authorized := range []bool{false, true} {
		t.Run(fmt.Sprint(authorized), func(t *testing.T) {
			viper.Set(common.MultiUserMode, true)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
			clients, manager, run := initWithOneTimeRun(t)
			defer clients.Close()
			mux := runtime.NewServeMux()
			require.NoError(t, api.RegisterTaskServiceHandlerServer(context.Background(), mux, NewTaskServer(manager)))
			body := fmt.Sprintf(`{"pipelineName":"pipeline/test","runId":%q,"mlmdExecutionID":"42","fingerprint":"fp","created_at":"1970-01-01T00:00:01Z"}`, run.UUID)
			request := httptest.NewRequest(http.MethodPost, "/apis/v1alpha1/tasks", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			if authorized {
				request.Header.Set("Authorization", "Bearer synthetic")
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if authorized {
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			} else {
				require.NotEqual(t, http.StatusOK, response.Code, response.Body.String())
			}
			listed, err := NewTaskServer(manager).ListTasksV1(context.Background(), &api.ListTasksRequest{})
			require.NoError(t, err)
			if authorized {
				require.Len(t, listed.Tasks, 1)
			} else {
				require.Empty(t, listed.Tasks)
			}
		})
	}
}

// audienceReview models Kubernetes checking the token's audience against the
// audiences requested by KFP. The token strings are synthetic test identifiers.
type audienceReview struct{}

func (audienceReview) Create(_ context.Context, r *authnv1.TokenReview, _ metav1.CreateOptions) (*authnv1.TokenReview, error) {
	audience := "pipelines.kubeflow.org"
	if r.Spec.Token == "task-only" {
		audience += "/task-write"
	}
	accepted := false
	for _, requested := range r.Spec.Audiences {
		accepted = accepted || requested == audience
	}
	return &authnv1.TokenReview{Status: authnv1.TokenReviewStatus{Authenticated: accepted, Audiences: []string{audience}, User: authnv1.UserInfo{Username: "system:serviceaccount:ns1:custom-runner"}}}, nil
}

type audienceClientManager struct{ *resource.FakeClientManager }

func (c audienceClientManager) TokenReviewClient() client.TokenReviewInterface {
	return audienceReview{}
}

func TestTaskOnlyTokenCannotAuthorizeOtherRunOperations(t *testing.T) {
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
	for _, token := range []string{"task-only", "general"} {
		t.Run(token, func(t *testing.T) {
			clients, _, run := initWithOneTimeRun(t)
			defer clients.Close()
			clients.AuthenticatorsFake = auth.GetAuthenticators(audienceReview{})
			manager := resource.NewResourceManager(audienceClientManager{clients}, &resource.ResourceManagerOptions{})
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
			result, err := NewTaskServer(manager).CreateTaskV1(ctx, &api.CreateTaskRequest{Task: &api.Task{PipelineName: "pipeline/test", RunId: run.UUID, MlmdExecutionID: "42", Fingerprint: "fp", CreatedAt: timestamppb.New(time.Unix(1, 0))}})
			require.NoError(t, err)
			require.NotNil(t, result)
			runServer := &BaseRunServer{resourceManager: manager}
			for _, verb := range []string{common.RbacResourceVerbGet, common.RbacResourceVerbDelete, common.RbacResourceVerbRetry} {
				err := runServer.canAccessRun(ctx, run.UUID, &authzv1.ResourceAttributes{Verb: verb})
				if token == "task-only" {
					require.Error(t, err, verb)
				} else {
					require.NoError(t, err, verb)
				}
			}
		})
	}
}
