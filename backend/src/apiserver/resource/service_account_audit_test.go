// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type auditModeReview struct {
	status  authzv1.SubjectAccessReviewStatus
	err     error
	reviews []*authzv1.SubjectAccessReview
}

func (c *auditModeReview) Create(_ context.Context, review *authzv1.SubjectAccessReview, _ metav1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	c.reviews = append(c.reviews, review.DeepCopy())
	if c.err != nil {
		return nil, c.err
	}
	return &authzv1.SubjectAccessReview{Status: c.status}, nil
}

func configureServiceAccountAuditTest(t *testing.T, mode, allowed string) {
	t.Helper()
	for key, value := range map[string]string{
		common.ServiceAccountAuthorizationMode: mode,
		common.MultiUserMode:                   "true",
		common.AllowedServiceAccountsFlag:      allowed,
		v1AllowedNamespaces:                    "ns1",
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
}

func TestServiceAccountAuthorizationModeSubmissionPaths(t *testing.T) {
	for _, path := range []string{"run", "schedule", "scheduled replay"} {
		t.Run(path, func(t *testing.T) {
			for _, mode := range []string{"", "enforce", "audit", "invalid"} {
				t.Run("mode="+mode, func(t *testing.T) {
					for _, allowed := range []string{"", "custom-sa"} {
						t.Run("allowlist="+allowed, func(t *testing.T) {
							configureServiceAccountAuditTest(t, mode, allowed)
							store, _, experiment := initWithExperiment(t)
							defer store.Close()
							review := &auditModeReview{} // SAR denies the requested account.
							store.SubjectAccessReviewClientFake = review
							manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
							spec := model.PipelineSpec{WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore())}
							var err error
							switch path {
							case "run":
								_, err = manager.CreateRun(multiUserContext(), &model.Run{
									DisplayName: "audit-run", Namespace: "ns1", ExperimentId: experiment.UUID,
									ServiceAccount: "custom-sa", PipelineSpec: spec,
								})
							case "schedule":
								_, err = manager.CreateJob(multiUserContext(), &model.Job{
									DisplayName: "audit-schedule", Namespace: "ns1", ExperimentId: experiment.UUID,
									ServiceAccount: "custom-sa", PipelineSpec: spec, Enabled: true,
								})
							case "scheduled replay":
								err = manager.authorizeStoredRunServiceAccount(multiUserContext(), &model.Run{
									UUID: "stored-run", Namespace: "ns1", ServiceAccount: "custom-sa",
								})
							}
							if mode == "audit" {
								require.NoError(t, err)
								require.NotEmpty(t, review.reviews, "audit must evaluate SAR even when the allowlist denies")
							} else {
								require.Error(t, err, "omitting the mode must retain enforcement")
							}
							for _, request := range review.reviews {
								require.Equal(t, "user@google.com", request.Spec.User)
								require.Equal(t, authzv1.ResourceAttributes{
									Verb: "use", Namespace: "ns1", Resource: "serviceaccounts", Name: "custom-sa",
								}, *request.Spec.ResourceAttributes)
							}
						})
					}
				})
			}
		})
	}
}

func TestServiceAccountAuditFailsClosedOnAuthenticationAndReviewErrors(t *testing.T) {
	for _, test := range []struct {
		name      string
		ctx       context.Context
		status    authzv1.SubjectAccessReviewStatus
		reviewErr error
		code      codes.Code
	}{
		{name: "missing identity", ctx: context.Background(), code: codes.Unauthenticated},
		{name: "review transport failure", ctx: multiUserContext(), reviewErr: errors.New("review unavailable"), code: codes.Internal},
		{name: "review evaluation failure", ctx: multiUserContext(), status: authzv1.SubjectAccessReviewStatus{EvaluationError: "authorizer unavailable"}, code: codes.Internal},
	} {
		t.Run(test.name, func(t *testing.T) {
			// An allowlist denial must not prevent discovering authentication/review failures.
			configureServiceAccountAuditTest(t, "audit", "")
			store, _, _ := initWithExperiment(t)
			defer store.Close()
			store.SubjectAccessReviewClientFake = &auditModeReview{status: test.status, err: test.reviewErr}
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
			err := manager.authorizeServiceAccount(test.ctx, "custom-sa", "ns1")
			require.Error(t, err)
			require.True(t, util.IsUserErrorCodeMatch(err, test.code), "unexpected error: %v", err)
		})
	}
}

func TestServiceAccountAuditDoesNotBypassUnrelatedAuthorization(t *testing.T) {
	configureServiceAccountAuditTest(t, "audit", "custom-sa")
	store, _, _ := initWithExperiment(t)
	defer store.Close()
	store.SubjectAccessReviewClientFake = &auditModeReview{}
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	err := manager.IsAuthorized(multiUserContext(), &authzv1.ResourceAttributes{
		Verb: "create", Namespace: "other-namespace", Group: common.RbacPipelinesGroup, Resource: common.RbacResourceTypeRuns,
	})
	require.Error(t, err)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied))
}

func TestServiceAccountAuditScheduledTickMigrationAndRevocation(t *testing.T) {
	initEnvVars()
	configureServiceAccountAuditTest(t, "audit", "")
	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	review := &recurringRunAccountReview{}
	store.SubjectAccessReviewClientFake = review
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	manager.time = fixedRecurringTime{epoch: 200}
	ctx := multiUserContext()
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "audit-migration", Namespace: "ns1"})
	require.NoError(t, err)
	job, err := manager.CreateJob(ctx, &model.Job{
		DisplayName: "audit-migration", Namespace: "ns1", ExperimentId: experiment.UUID,
		ServiceAccount: "custom-sa", Enabled: true, MaxConcurrency: 1,
		Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
			PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
		}},
		PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore())},
	})
	require.NoError(t, err)
	submit := func(key string) (*model.Run, error) {
		run := &model.Run{DisplayName: key, RecurringRunId: job.UUID}
		if err := manager.PrepareRecurringRun(ctx, run); err != nil {
			return nil, err
		}
		return manager.CreateRun(ctx, run)
	}
	first, err := submit("audit-tick")
	require.NoError(t, err)
	require.Equal(t, "custom-sa", first.ServiceAccount)
	require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())

	// Completing migration changes policy without recreating the existing schedule.
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	review.allowedAccount = "custom-sa"
	viper.Set(common.ServiceAccountAuthorizationMode, "enforce")
	replay, err := submit("audit-tick")
	require.NoError(t, err)
	require.Equal(t, first.UUID, replay.UUID)
	first.State = model.RuntimeStateSucceeded
	first.FinishedAtInSec = 201
	require.NoError(t, store.RunStore().UpdateRun(first))
	manager.time = fixedRecurringTime{epoch: 210}
	second, err := submit("enforced-tick")
	require.NoError(t, err)
	require.NotEqual(t, first.UUID, second.UUID)
	require.Equal(t, "custom-sa", second.ServiceAccount)
	require.Equal(t, 2, store.ExecClientFake.GetWorkflowCount())

	// Revoking use permission must reject acknowledgement of a retained execution.
	review.allowedAccount = ""
	_, err = submit("enforced-tick")
	require.Error(t, err)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied))
	require.Equal(t, 2, store.ExecClientFake.GetWorkflowCount())
}
