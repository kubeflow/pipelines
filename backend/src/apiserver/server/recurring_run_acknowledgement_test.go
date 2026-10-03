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
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type recurringAcknowledgementReview struct {
	deniedResource string
	deniedVerb     string
	reviews        []*authv1.SubjectAccessReview
}

func (s *recurringAcknowledgementReview) Create(_ context.Context, review *authv1.SubjectAccessReview, _ metav1.CreateOptions) (*authv1.SubjectAccessReview, error) {
	s.reviews = append(s.reviews, review.DeepCopy())
	attributes := review.Spec.ResourceAttributes
	allowed := attributes.Resource != s.deniedResource || attributes.Verb != s.deniedVerb
	return &authv1.SubjectAccessReview{Status: authv1.SubjectAccessReviewStatus{Allowed: allowed}}, nil
}

type recurringAcknowledgementFixture struct {
	clients *resource.FakeClientManager
	manager *resource.ResourceManager
	server  *RunServer
	job     *model.Job
	review  *recurringAcknowledgementReview
}

func newRecurringAcknowledgementFixture(t *testing.T) *recurringAcknowledgementFixture {
	t.Helper()
	initEnvVars()
	for key, value := range map[string]interface{}{
		common.MultiUserMode:                   true,
		common.MultiUserModeSharedReadAccess:   false,
		common.AllowedServiceAccountsFlag:      "custom-sa",
		common.ServiceAccountAuthorizationMode: "enforce",
		common.WorkflowIdentityMode:            "enforce",
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	clients := resource.NewFakeClientManagerOrFatal(util.NewFakeTime(time.Unix(200, 0)))
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	review := &recurringAcknowledgementReview{}
	clients.SubjectAccessReviewClientFake = review
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "acknowledgement", Namespace: "ns1"})
	require.NoError(t, err)
	pipeline, err := manager.CreatePipeline(&model.Pipeline{Name: "acknowledgement-source", Namespace: "ns1"})
	require.NoError(t, err)
	version, err := manager.CreatePipelineVersion(&model.PipelineVersion{
		Name: "pinned-source", PipelineId: pipeline.UUID, PipelineSpec: model.LargeText(v2SpecHelloWorld),
	})
	require.NoError(t, err)
	job, err := manager.CreateJob(scheduleContext("user@google.com"), &model.Job{
		DisplayName: "pinned-acknowledgement", Namespace: "ns1", ExperimentId: experiment.UUID,
		Enabled: true, MaxConcurrency: 1, ServiceAccount: "custom-sa",
		Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
			PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
		}},
		PipelineSpec: model.PipelineSpec{
			PipelineId: pipeline.UUID, PipelineVersionId: version.UUID,
			RuntimeConfig: model.RuntimeConfig{Parameters: `{"param1":"world"}`, PipelineRoot: "schedule-root"},
		},
	})
	require.NoError(t, err)
	return &recurringAcknowledgementFixture{clients: clients, manager: manager, server: createRunServer(manager), job: job, review: review}
}

func (f *recurringAcknowledgementFixture) submit(requestKey string) (*api.Run, error) {
	return f.server.CreateRun(scheduleContext(scheduleControllerIdentity), &api.CreateRunRequest{Run: &api.Run{
		DisplayName: requestKey, RecurringRunId: f.job.UUID,
	}})
}

func (f *recurringAcknowledgementFixture) deleteCompletedTickAndSource(t *testing.T) *api.Run {
	t.Helper()
	first, err := f.submit("same-tick")
	require.NoError(t, err)
	require.NoError(t, f.manager.DeleteRun(scheduleContext(scheduleControllerIdentity), first.RunId))
	require.NoError(t, f.manager.DeletePipelineVersion(f.job.PipelineVersionId))
	state, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
	require.NoError(t, err)
	require.False(t, state.Pending)
	require.Equal(t, "same-tick", state.RequestKey)
	return first
}

func TestRecurringRunAcknowledgementAfterPinnedVersionDeletion(t *testing.T) {
	f := newRecurringAcknowledgementFixture(t)
	first := f.deleteCompletedTickAndSource(t)
	before, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
	require.NoError(t, err)

	acknowledged, err := f.submit("same-tick")
	require.NoError(t, err)
	require.Equal(t, first.RunId, acknowledged.RunId)
	require.Equal(t, first.CreatedAt, acknowledged.CreatedAt)
	require.Equal(t, first.ScheduledAt, acknowledged.ScheduledAt)
	require.Equal(t, "custom-sa", acknowledged.ServiceAccount)
	require.Equal(t, api.RuntimeState_RUNTIME_STATE_UNSPECIFIED, acknowledged.State)
	_, err = f.manager.GetRun(first.RunId)
	require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "acknowledgement must not recreate the deleted run: %v", err)
	require.Zero(t, f.clients.ExecClientFake.GetWorkflowCount())
	after, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
	require.NoError(t, err)
	require.Equal(t, before, after)

	// Source-independent acknowledgement cannot authorize another execution.
	_, err = f.submit("next-tick")
	require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "a new tick still requires its pinned version: %v", err)
	after, err = f.clients.JobStore().GetRecurringRunState(f.job.UUID)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Zero(t, f.clients.ExecClientFake.GetWorkflowCount())
}

func TestRecurringRunAcknowledgementAfterVersionDeletionRequiresPermissions(t *testing.T) {
	for _, test := range []struct {
		name, resource, verb string
	}{
		{name: "namespace run creation", resource: "runs", verb: "create"},
		{name: "service account use", resource: "serviceaccounts", verb: "use"},
		{name: "owning pipeline read", resource: "pipelines", verb: "get"},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newRecurringAcknowledgementFixture(t)
			first := f.deleteCompletedTickAndSource(t)
			before, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
			require.NoError(t, err)
			f.review.deniedResource, f.review.deniedVerb = test.resource, test.verb
			f.review.reviews = nil

			acknowledged, err := f.submit("same-tick")
			require.Nil(t, acknowledged)
			require.True(t, util.IsUserErrorCodeMatch(err, codes.PermissionDenied), "acknowledgement must preserve authorization: %v", err)
			require.NotEmpty(t, f.review.reviews)
			denied := f.review.reviews[len(f.review.reviews)-1]
			require.Equal(t, scheduleControllerIdentity, denied.Spec.User)
			require.Equal(t, "ns1", denied.Spec.ResourceAttributes.Namespace)
			require.Equal(t, test.resource, denied.Spec.ResourceAttributes.Resource)
			require.Equal(t, test.verb, denied.Spec.ResourceAttributes.Verb)
			after, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
			require.NoError(t, err)
			require.Equal(t, before, after)
			require.Zero(t, f.clients.ExecClientFake.GetWorkflowCount())

			f.review.deniedResource, f.review.deniedVerb = "", ""
			acknowledged, err = f.submit("same-tick")
			require.NoError(t, err)
			require.Equal(t, first.RunId, acknowledged.RunId)
		})
	}
}

func TestRecurringRunAcknowledgementAfterVersionDeletionInspectsRetainedExecution(t *testing.T) {
	for _, staticExecution := range []bool{false, true} {
		name := "dynamic patch requires source provenance"
		if staticExecution {
			name = "static execution can be inspected without its source"
		}
		t.Run(name, func(t *testing.T) {
			f := newRecurringAcknowledgementFixture(t)
			first, err := f.submit("same-tick")
			require.NoError(t, err)
			stored, err := f.manager.GetRun(first.RunId)
			require.NoError(t, err)
			if staticExecution {
				// Retained historical workflows need no compiler exemption when
				// every service-account-bearing field can be inspected directly.
				execution, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(stored.PipelineRuntimeManifest))
				require.NoError(t, err)
				workflow := execution.(*util.Workflow)
				workflow.Spec.PodSpecPatch = ""
				for i := range workflow.Spec.Templates {
					workflow.Spec.Templates[i].PodSpecPatch = ""
				}
				stored.WorkflowRuntimeManifest = model.LargeText(workflow.ToStringForStore())
				require.NoError(t, f.clients.RunStore().UpdateRun(stored))
			}
			require.NoError(t, f.manager.DeletePipelineVersion(f.job.PipelineVersionId))
			before, err := f.manager.GetRun(first.RunId)
			require.NoError(t, err)
			require.Equal(t, stored.WorkflowRuntimeManifest, before.WorkflowRuntimeManifest)
			stateBefore, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
			require.NoError(t, err)

			acknowledged, err := f.submit("same-tick")
			if staticExecution {
				require.NoError(t, err)
				require.Equal(t, first.RunId, acknowledged.RunId)
			} else {
				require.Nil(t, acknowledged)
				require.ErrorContains(t, err, "podSpecPatch contains a template expression")
			}
			after, err := f.manager.GetRun(first.RunId)
			require.NoError(t, err)
			require.Equal(t, before, after)
			stateAfter, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
			require.NoError(t, err)
			require.Equal(t, stateBefore, stateAfter)
			require.Equal(t, 1, f.clients.ExecClientFake.GetWorkflowCount())
		})
	}
}

func TestRecurringRunPendingAcknowledgementStillRequiresPinnedVersion(t *testing.T) {
	for _, deletedVersion := range []bool{false, true} {
		name := "pipeline permission denied"
		if deletedVersion {
			name = "pinned version deleted"
		}
		t.Run(name, func(t *testing.T) {
			f := newRecurringAcknowledgementFixture(t)
			// Simulate interruption after claiming a tick, before any workflow exists.
			pending, err := f.clients.JobStore().ClaimRecurringRun(f.job.UUID, "pending-tick", 0, 110, 200, f.job.PipelineVersionId)
			require.NoError(t, err)
			require.True(t, pending.Pending)
			wantCode := codes.PermissionDenied
			if deletedVersion {
				require.NoError(t, f.manager.DeletePipelineVersion(f.job.PipelineVersionId))
				wantCode = codes.NotFound
			} else {
				f.review.deniedResource, f.review.deniedVerb = "pipelines", "get"
			}

			acknowledged, err := f.submit("pending-tick")
			require.Nil(t, acknowledged)
			require.True(t, util.IsUserErrorCodeMatch(err, wantCode), "a pending tick must still authorize its execution source: %v", err)
			after, err := f.clients.JobStore().GetRecurringRunState(f.job.UUID)
			require.NoError(t, err)
			require.Equal(t, pending, after)
			require.Zero(t, f.clients.ExecClientFake.GetWorkflowCount())
		})
	}
}
