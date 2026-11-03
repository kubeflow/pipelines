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
	"slices"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const taskWriterIdentity = "system:serviceaccount:ns1:custom-runner"

type taskTokenReview struct{ invalid, wrongAudience bool }

func (c taskTokenReview) Create(_ context.Context, review *authnv1.TokenReview, _ metav1.CreateOptions) (*authnv1.TokenReview, error) {
	audiences := review.Spec.Audiences
	if c.wrongAudience {
		audiences = []string{"other"}
	}
	return &authnv1.TokenReview{Status: authnv1.TokenReviewStatus{Authenticated: !c.invalid, Audiences: audiences, User: authnv1.UserInfo{Username: taskWriterIdentity}}}, nil
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

func seedRunningTask(t *testing.T, manager *resource.ResourceManager, runID, name string) string {
	t.Helper()
	created, err := createRunServer(manager).CreateTask(ctxWithUser(), &apiv2.CreateTaskRequest{RunId: runID, Task: &apiv2.PipelineTask{Name: name, State: apiv2.PipelineTask_RUNNING, Type: apiv2.PipelineTask_RUNTIME}})
	require.NoError(t, err)
	return created.GetTaskId()
}

func runTaskNames(t *testing.T, manager *resource.ResourceManager, runID string) []string {
	t.Helper()
	tasks, _, _, err := manager.ListTasks(runID, "", "", list.EmptyOptions())
	require.NoError(t, err)
	names := make([]string, 0, len(tasks))
	for _, task := range tasks {
		names = append(names, task.Name)
	}
	return names
}

func taskState(t *testing.T, manager *resource.ResourceManager, taskID string) model.TaskStatus {
	t.Helper()
	task, err := manager.GetTask(taskID)
	require.NoError(t, err)
	return task.State
}

// writeTask performs one task write and reports whether its effect was persisted.
func writeTask(t *testing.T, ctx context.Context, server *RunServer, manager *resource.ResourceManager, operation, runID, existingTaskID string) (bool, error) {
	t.Helper()
	switch operation {
	case "create":
		_, err := server.CreateTask(ctx, &apiv2.CreateTaskRequest{RunId: runID, Task: &apiv2.PipelineTask{Name: "written", State: apiv2.PipelineTask_RUNNING, Type: apiv2.PipelineTask_RUNTIME}})
		return slices.Contains(runTaskNames(t, manager, runID), "written"), err
	case "update":
		_, err := server.UpdateTask(ctx, &apiv2.UpdateTaskRequest{RunId: runID, TaskId: existingTaskID, Task: &apiv2.PipelineTask{State: apiv2.PipelineTask_SUCCEEDED}})
		return taskState(t, manager, existingTaskID) == model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), err
	case "bulk-update":
		_, err := server.UpdateTasksBulk(ctx, &apiv2.UpdateTasksBulkRequest{RunId: runID, Tasks: map[string]*apiv2.PipelineTask{existingTaskID: {State: apiv2.PipelineTask_SUCCEEDED}}})
		return taskState(t, manager, existingTaskID) == model.TaskStatus(apiv2.PipelineTask_SUCCEEDED), err
	}
	t.Fatalf("unknown task write %q", operation)
	return false, nil
}

var taskWriteOperations = []string{"create", "update", "bulk-update"}

func TestTaskWriteAuthorization(t *testing.T) {
	expectedCode := map[string]codes.Code{
		"anonymous":        codes.Unauthenticated,
		"invalid":          codes.Unauthenticated,
		"wrong-audience":   codes.Unauthenticated,
		"denied":           codes.PermissionDenied,
		"evaluation-error": codes.Internal,
		"SAR-unavailable":  codes.Internal,
	}
	for _, operation := range taskWriteOperations {
		for _, name := range []string{"anonymous", "invalid", "wrong-audience", "denied", "evaluation-error", "SAR-unavailable", "authorized", "single-user"} {
			t.Run(operation+"/"+name, func(t *testing.T) {
				viper.Set(common.MultiUserMode, name != "single-user")
				t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
				clients, manager, run := initWithOneTimeRunV2(t)
				defer clients.Close()
				existingTaskID := seedRunningTask(t, manager, run.UUID, "existing")

				reviews := &taskAccessReview{deny: name == "denied", evaluationError: name == "evaluation-error", transportError: name == "SAR-unavailable"}
				clients.SubjectAccessReviewClientFake = reviews
				clients.AuthenticatorsFake = []auth.Authenticator{auth.NewTokenReviewAuthenticator(common.AuthorizationBearerTokenHeader, common.AuthorizationBearerTokenPrefix, []string{common.GetTokenReviewAudience()}, taskTokenReview{invalid: name == "invalid", wrongAudience: name == "wrong-audience"})}
				manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
				ctx := context.Background()
				if name != "anonymous" {
					ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("authorization", "Bearer synthetic"))
				}

				persisted, err := writeTask(t, ctx, createRunServer(manager), manager, operation, run.UUID, existingTaskID)
				if name == "authorized" || name == "single-user" {
					require.NoError(t, err)
					require.True(t, persisted)
				} else {
					require.Error(t, err)
					require.Equal(t, expectedCode[name], status.Code(err), err.Error())
					require.False(t, persisted)
				}
				if name == "authorized" {
					require.Len(t, reviews.reviews, 1)
					review := reviews.reviews[0].Spec
					require.Equal(t, taskWriterIdentity, review.User)
					require.Empty(t, review.Groups)
					require.Equal(t, &authzv1.ResourceAttributes{Namespace: run.Namespace, Verb: common.RbacResourceVerbUpdate, Group: common.RbacPipelinesGroup, Version: common.RbacPipelinesVersion, Resource: common.RbacResourceTypeRuns, Name: run.K8SName}, review.ResourceAttributes)
				}
				if name == "single-user" {
					require.Empty(t, reviews.reviews)
				}
			})
		}
	}
}

func TestCreateTaskHTTPAuthorization(t *testing.T) {
	for _, authorized := range []bool{false, true} {
		t.Run(fmt.Sprint(authorized), func(t *testing.T) {
			viper.Set(common.MultiUserMode, true)
			t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
			clients, manager, run := initWithOneTimeRunV2(t)
			defer clients.Close()
			mux := runtime.NewServeMux()
			require.NoError(t, apiv2.RegisterRunServiceHandlerServer(context.Background(), mux, createRunServer(manager)))
			request := httptest.NewRequest(http.MethodPost, "/apis/v2/runs/"+run.UUID+"/tasks", strings.NewReader(`{"name":"written","state":"RUNNING","type":"RUNTIME"}`))
			request.Header.Set("Content-Type", "application/json")
			if authorized {
				request.Header.Set("Authorization", "Bearer synthetic")
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if authorized {
				require.Equal(t, http.StatusOK, response.Code, response.Body.String())
				require.Equal(t, []string{"written"}, runTaskNames(t, manager, run.UUID))
			} else {
				require.Equal(t, http.StatusUnauthorized, response.Code, response.Body.String())
				require.Empty(t, runTaskNames(t, manager, run.UUID))
			}
		})
	}
}

// runAudienceReview models Kubernetes audience checks for synthetic tokens:
// "run:<id>" carries only that run's audience; any other token the base audience.
type runAudienceReview struct{}

func (runAudienceReview) Create(_ context.Context, review *authnv1.TokenReview, _ metav1.CreateOptions) (*authnv1.TokenReview, error) {
	issued := common.GetTokenReviewAudience()
	if runID, ok := strings.CutPrefix(review.Spec.Token, "run:"); ok {
		issued = common.TokenAudienceForRun(runID)
	}
	if !slices.Contains(review.Spec.Audiences, issued) {
		return &authnv1.TokenReview{Status: authnv1.TokenReviewStatus{Error: "token audiences are invalid for the target audiences"}}, nil
	}
	return &authnv1.TokenReview{Status: authnv1.TokenReviewStatus{Authenticated: true, Audiences: []string{issued}, User: authnv1.UserInfo{Username: taskWriterIdentity}}}, nil
}

func TestRunScopedTokenCannotWriteAnotherRunsTasks(t *testing.T) {
	viper.Set(common.MultiUserMode, true)
	t.Cleanup(func() { viper.Set(common.MultiUserMode, false) })
	for _, operation := range taskWriteOperations {
		for _, token := range []string{"run-scoped", "general"} {
			t.Run(operation+"/"+token, func(t *testing.T) {
				clients, manager, ownRun := initWithOneTimeRunV2(t)
				defer clients.Close()
				// Same namespace and service account: only the token's run binding differs.
				otherRun, err := manager.CreateRun(ctxWithUser(), &model.Run{DisplayName: "other-run", ExperimentId: ownRun.ExperimentId, PipelineSpec: model.PipelineSpec{PipelineSpecManifest: model.LargeText(testIRPipeline)}})
				require.NoError(t, err)
				require.Equal(t, ownRun.Namespace, otherRun.Namespace)
				ownTaskID := seedRunningTask(t, manager, ownRun.UUID, "existing")
				otherTaskID := seedRunningTask(t, manager, otherRun.UUID, "existing")

				clients.AuthenticatorsFake = auth.GetAuthenticators(runAudienceReview{})
				manager = resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
				server := createRunServer(manager)
				bearer := "general"
				if token == "run-scoped" {
					bearer = "run:" + ownRun.UUID
				}
				ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+bearer))

				persisted, err := writeTask(t, ctx, server, manager, operation, ownRun.UUID, ownTaskID)
				require.NoError(t, err)
				require.True(t, persisted)

				persisted, err = writeTask(t, ctx, server, manager, operation, otherRun.UUID, otherTaskID)
				if token == "run-scoped" {
					require.Error(t, err)
					require.Equal(t, codes.Unauthenticated, status.Code(err), err.Error())
					require.False(t, persisted)
				} else {
					require.NoError(t, err)
					require.True(t, persisted)
				}

				// A runtime token must not authorize namespace-wide run access either.
				err = server.canAccessRun(ctx, "", &authzv1.ResourceAttributes{Namespace: ownRun.Namespace, Verb: common.RbacResourceVerbList})
				if token == "run-scoped" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}
