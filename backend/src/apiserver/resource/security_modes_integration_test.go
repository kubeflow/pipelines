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
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type securityModesReview struct {
	denied         map[string]bool
	failingAccount string
	failure        authzv1.SubjectAccessReviewStatus
	err            error
}

func (r *securityModesReview) Create(_ context.Context, review *authzv1.SubjectAccessReview, _ metav1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	attributes := review.Spec.ResourceAttributes
	status := authzv1.SubjectAccessReviewStatus{Allowed: true}
	if attributes.Resource == "serviceaccounts" {
		if attributes.Name == r.failingAccount {
			return &authzv1.SubjectAccessReview{Status: r.failure}, r.err
		}
		status.Allowed = !r.denied[attributes.Name]
	}
	return &authzv1.SubjectAccessReview{Status: status}, nil
}

func configureSecurityModes(t *testing.T, primary, workflow string) {
	t.Helper()
	for key, value := range map[string]string{
		common.ServiceAccountAuthorizationMode: primary,
		common.WorkflowIdentityMode:            workflow,
		common.MultiUserMode:                   "true",
		common.AllowedServiceAccountsFlag:      "custom-sa,helper-sa",
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
}

func TestSecurityModesIndependentPolicyEnforcement(t *testing.T) {
	for _, primary := range []string{"enforce", "audit"} {
		for _, workflowMode := range []string{"enforce", "audit"} {
			for _, denial := range []string{"main", "helper", "both"} {
				t.Run(primary+"/"+workflowMode+"/"+denial, func(t *testing.T) {
					configureSecurityModes(t, primary, workflowMode)
					store, _, _ := initWithExperiment(t)
					defer store.Close()
					denyMain, denyHelper := denial != "helper", denial != "main"
					store.SubjectAccessReviewClientFake = &securityModesReview{denied: map[string]bool{"custom-sa": denyMain, "helper-sa": denyHelper}}
					manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
					execution := util.NewWorkflow(testWorkflow.DeepCopy())
					execution.Spec.ServiceAccountName = "custom-sa"
					execution.Spec.Templates[0].ServiceAccountName = "helper-sa"
					err := manager.authorizeExecutionServiceAccounts(multiUserContext(), execution, false, "ns1", "integration_test")
					if (denyMain && primary == "enforce") || (denyHelper && workflowMode == "enforce") {
						require.Error(t, err)
						require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "unexpected error: %v", err)
					} else {
						require.NoError(t, err)
					}
				})
			}
		}
	}
}

func TestSecurityModesInfrastructureFailureAlwaysBlocks(t *testing.T) {
	for _, account := range []string{"custom-sa", "helper-sa"} {
		for _, failure := range []struct {
			name   string
			status authzv1.SubjectAccessReviewStatus
			err    error
		}{
			{name: "transport", err: errors.New("authorization unavailable")},
			{name: "evaluation", status: authzv1.SubjectAccessReviewStatus{EvaluationError: "authorization unavailable"}},
			{name: "allowed with evaluation error", status: authzv1.SubjectAccessReviewStatus{Allowed: true, EvaluationError: "authorization unavailable"}},
		} {
			t.Run(account+"/"+failure.name, func(t *testing.T) {
				configureSecurityModes(t, "audit", "audit")
				store, _, _ := initWithExperiment(t)
				defer store.Close()
				store.SubjectAccessReviewClientFake = &securityModesReview{failingAccount: account, failure: failure.status, err: failure.err}
				manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
				execution := util.NewWorkflow(testWorkflow.DeepCopy())
				execution.Spec.ServiceAccountName = "custom-sa"
				execution.Spec.Templates[0].ServiceAccountName = "helper-sa"
				err := manager.authorizeExecutionServiceAccounts(multiUserContext(), execution, false, "ns1", "integration_test")
				require.Error(t, err)
				require.True(t, util.IsUserErrorCodeMatch(err, codes.Internal), "unexpected error: %v", err)
			})
		}
	}
}

type helperIdentityDispatcher struct {
	apiserverPlugins.NoOpDispatcher
}

func (helperIdentityDispatcher) PluginsRegistered() bool { return true }
func (helperIdentityDispatcher) OnBeforeRunCreation(_ context.Context, _ *apiserverPlugins.PendingRun, execution util.ExecutionSpec) error {
	execution.(*util.Workflow).Spec.Templates[0].ServiceAccountName = "helper-sa"
	return nil
}

func TestSecurityModesRecurringTickReturnToEnforcement(t *testing.T) {
	initEnvVars()
	configureSecurityModes(t, "audit", "audit")
	store := NewFakeClientManagerOrFatalV2()
	defer store.Close()
	review := &securityModesReview{denied: map[string]bool{"helper-sa": true}}
	store.SubjectAccessReviewClientFake = review
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	manager.time = fixedRecurringTime{epoch: 200}
	ctx := multiUserContext()
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "identity-migration", Namespace: "ns1"})
	require.NoError(t, err)
	manager.pluginDispatcher = helperIdentityDispatcher{}
	job, err := manager.CreateJob(ctx, &model.Job{
		DisplayName: "identity-migration", Namespace: "ns1", ExperimentId: experiment.UUID,
		ServiceAccount: "custom-sa", Enabled: true, MaxConcurrency: 1,
		Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
			PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
		}},
		PipelineSpec: model.PipelineSpec{PipelineSpecManifest: model.LargeText(v2SpecHelloWorld), RuntimeConfig: model.RuntimeConfig{Parameters: `{"text":"world"}`, PipelineRoot: "schedule-root"}},
	})
	require.NoError(t, err)
	submit := func(key string) (*model.Run, error) {
		run := &model.Run{DisplayName: key, RecurringRunId: job.UUID}
		if err := manager.PrepareRecurringRun(ctx, run); err != nil {
			return nil, err
		}
		return manager.CreateRun(ctx, run)
	}
	first, err := submit("identity-audit-tick")
	require.NoError(t, err)
	require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
	first.State = model.RuntimeStateSucceeded
	first.FinishedAtInSec = 201
	require.NoError(t, store.RunStore().UpdateRun(first))
	manager.time = fixedRecurringTime{epoch: 210}
	before, err := store.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	// Primary-account audit must not keep relaxing additional identities after migration.
	viper.Set(common.WorkflowIdentityMode, "enforce")
	// Replaying a retained run inspects its original runtime, not the current template.
	_, err = submit("identity-audit-tick")
	require.Error(t, err)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "unexpected replay error: %v", err)
	require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
	_, err = submit("identity-enforced-tick")
	require.Error(t, err)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "unexpected error: %v", err)
	require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
	after, err := store.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.True(t, after.Pending, "post-plugin denial must keep the tick retryable")
	require.Equal(t, before.LastRunIndex+1, after.LastRunIndex)
	// Grant the missing permission and retry the same tick without recreating the schedule.
	review.denied["helper-sa"] = false
	second, err := submit("identity-enforced-tick")
	require.NoError(t, err)
	require.NotEqual(t, first.UUID, second.UUID)
	require.Equal(t, 2, store.ExecClientFake.GetWorkflowCount())
}

func TestSecurityModesStoredReplayIdentity(t *testing.T) {
	for _, mode := range []string{"enforce", "audit"} {
		for _, recovered := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/recovered=%t", mode, recovered), func(t *testing.T) {
				configureSecurityModes(t, "audit", mode)
				store, _, _ := initWithExperiment(t)
				defer store.Close()
				review := &securityModesReview{denied: map[string]bool{"helper-sa": true}}
				store.SubjectAccessReviewClientFake = review
				manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
				execution := util.NewWorkflow(testWorkflow.DeepCopy())
				execution.SetExecutionName("retained-workflow")
				execution.Spec.ServiceAccountName = "custom-sa"
				execution.Spec.Templates[0].ServiceAccountName = "helper-sa"
				run := &model.Run{UUID: "retained-run", Namespace: "ns1", ServiceAccount: "custom-sa", RunDetails: model.RunDetails{WorkflowRuntimeManifest: model.LargeText(execution.ToStringForStore())}}
				if recovered {
					run.ServiceAccount = ""
				}
				err := manager.authorizeStoredRunServiceAccount(multiUserContext(), run)
				if mode == "enforce" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				// Infrastructure failures still block when the additional account is in audit.
				review.failingAccount = "helper-sa"
				review.err = errors.New("review unavailable")
				require.Error(t, manager.authorizeStoredRunServiceAccount(multiUserContext(), run))
			})
		}
	}
}

func TestSecurityModesTemplatedMainAccountStillBlocks(t *testing.T) {
	configureSecurityModes(t, "audit", "audit")
	store, _, _ := initWithExperiment(t)
	defer store.Close()
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	execution := util.NewWorkflow(testWorkflow.DeepCopy())
	execution.Spec.ServiceAccountName = "{{workflow.parameters.account}}"
	err := manager.authorizeExecutionServiceAccounts(multiUserContext(), execution, false, "ns1", "create_run")
	require.Error(t, err)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.InvalidArgument), "unexpected error: %v", err)
}
