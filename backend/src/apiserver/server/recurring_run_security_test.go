// Copyright 2018 The Kubeflow Authors
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

	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"google.golang.org/protobuf/types/known/timestamppb"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const scheduleControllerIdentity = "system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow"

type scheduledAccountReview struct {
	controllerAllowed bool
	denyRunCreation   bool
	reviews           []*authv1.SubjectAccessReview
}

func (s *scheduledAccountReview) Create(_ context.Context, review *authv1.SubjectAccessReview, _ metav1.CreateOptions) (*authv1.SubjectAccessReview, error) {
	s.reviews = append(s.reviews, review.DeepCopy())
	allowed := !s.denyRunCreation || review.Spec.ResourceAttributes.Resource != "runs"
	if review.Spec.ResourceAttributes.Resource == "serviceaccounts" {
		allowed = review.Spec.ResourceAttributes.Name == "custom-sa" &&
			(review.Spec.User == "user@google.com" || (review.Spec.User == scheduleControllerIdentity && s.controllerAllowed))
	}
	return &authv1.SubjectAccessReview{Status: authv1.SubjectAccessReviewStatus{Allowed: allowed}}, nil
}
func scheduleContext(user string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(common.GoogleIAPUserIdentityHeader, common.GoogleIAPUserIdentityPrefix+user))
}
func newAuthorizedSchedule(t *testing.T) (*resource.FakeClientManager, *resource.ResourceManager, *model.Job, *scheduledAccountReview) {
	t.Helper()
	return newAuthorizedScheduleWithCatchupPolicy(t, false)
}

func newAuthorizedScheduleWithCatchupPolicy(t *testing.T, noCatchup bool) (*resource.FakeClientManager, *resource.ResourceManager, *model.Job, *scheduledAccountReview) {
	t.Helper()
	return newAuthorizedScheduleWithTrigger(t, noCatchup, model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
		PeriodicScheduleStartTimeInSec: util.Int64Pointer(90), IntervalSecond: util.Int64Pointer(10),
	}}, time.Unix(99, 0))
}

func newAuthorizedScheduleWithTrigger(t *testing.T, noCatchup bool, trigger model.Trigger, now time.Time) (*resource.FakeClientManager, *resource.ResourceManager, *model.Job, *scheduledAccountReview) {
	t.Helper()
	viper.Set(common.MultiUserMode, "true")
	viper.Set(common.AllowedServiceAccountsFlag, "custom-sa")
	t.Cleanup(func() { viper.Set(common.MultiUserMode, "false"); viper.Set(common.AllowedServiceAccountsFlag, "") })
	initEnvVars()
	clients := resource.NewFakeClientManagerOrFatal(util.NewFakeTime(now))
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	review := &scheduledAccountReview{}
	clients.SubjectAccessReviewClientFake = review
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "exp1", Namespace: "ns1"})
	require.NoError(t, err)
	job, err := manager.CreateJob(scheduleContext("user@google.com"), &model.Job{
		DisplayName: "authorized-schedule", Namespace: "ns1", ExperimentId: experiment.UUID, Enabled: true,
		MaxConcurrency: 1, NoCatchup: noCatchup,
		Trigger:        trigger,
		ServiceAccount: "custom-sa", PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore()), Parameters: `[{"name":"param1","value":"authorized-[[Index]]-[[ScheduledTime]]"}]`},
	})
	require.NoError(t, err)
	require.Equal(t, "custom-sa", job.ServiceAccount)
	return clients, manager, job, review
}

func TestRecurringRunUsesStoredInputsAndStillRequiresCallerServiceAccountPermission(t *testing.T) {
	clients, manager, job, review := newAuthorizedSchedule(t)
	ctx := scheduleContext(scheduleControllerIdentity)
	swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
	swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	swf.Spec.ServiceAccount = "privileged-sa"
	swf.Spec.PipelineId = "attacker-pipeline"
	swf.Spec.Workflow.PipelineRoot = "s3://attacker"
	swf.Spec.Workflow.Parameters = []swfapi.Parameter{{Name: "param1", Value: "attacker"}}
	_, err = swfs.Update(ctx, swf)
	require.NoError(t, err)
	require.NoError(t, manager.ReportScheduledWorkflowResource(util.NewScheduledWorkflow(swf)))
	stored, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, job.PipelineSpec, stored.PipelineSpec)
	badPlugins := model.LargeText(`{"attacker":{"value":"untrusted"}}`)
	forged := &model.Run{RecurringRunId: job.UUID, PipelineSpec: model.PipelineSpec{PipelineId: "attacker", RuntimeConfig: model.RuntimeConfig{PipelineRoot: "s3://attacker"}}, PluginsInputString: &badPlugins}
	require.NoError(t, manager.PrepareRecurringRun(ctx, forged))
	require.Equal(t, job.PipelineSpec, forged.PipelineSpec)
	require.Nil(t, forged.PluginsInputString)
	server := createRunServer(manager)
	request := &api.CreateRunRequest{Run: &api.Run{DisplayName: "tick", RecurringRunId: job.UUID, ServiceAccount: "privileged-sa", ScheduledAt: timestamppb.New(time.Unix(100, 0))}}
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "Unauthorized")
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
	state, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Zero(t, state.LastRunIndex, "authorization failure must not claim a tick")
	last := review.reviews[len(review.reviews)-1]
	require.Equal(t, scheduleControllerIdentity, last.Spec.User)
	require.Equal(t, authv1.ResourceAttributes{Verb: "use", Namespace: "ns1", Resource: "serviceaccounts", Name: "custom-sa"}, *last.Spec.ResourceAttributes)
	review.controllerAllowed = true
	run, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	require.Equal(t, "custom-sa", run.ServiceAccount)
	require.Equal(t, job.ExperimentId, run.ExperimentId)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
	storedRun, err := manager.GetRun(run.RunId)
	require.NoError(t, err)
	workflow, err := util.NewExecutionSpecJSON(util.ArgoWorkflow, []byte(storedRun.WorkflowRuntimeManifest))
	require.NoError(t, err)
	require.Equal(t, "authorized-1-19700101000140", workflow.(*util.Workflow).GetWorkflowParametersAsMap()["param1"])
	// Supplying a recurring-run ID does not let a different user borrow the controller grant.
	// Even an idempotent replay must retain the service-account check.
	_, err = server.CreateRun(scheduleContext("unprivileged@google.com"), request)
	require.ErrorContains(t, err, "Unauthorized")
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
	advanceScheduledRunClock(clients, 200)
	review.controllerAllowed = false
	request.Run.DisplayName = "revoked-tick"
	_, err = server.CreateRun(ctx, request)
	require.ErrorContains(t, err, "Unauthorized")
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
	review.controllerAllowed = true
	viper.Set(common.AllowedServiceAccountsFlag, "")
	request.Run.DisplayName = "no-longer-allowlisted"
	_, err = server.CreateRun(ctx, request)
	require.Error(t, err)
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
}

func TestPrepareRecurringRunRejectsUntrustedIdentityAndDisabledSchedules(t *testing.T) {
	for _, kind := range []string{"unknown", "namespace", "experiment", "replaced", "disabled-db", "disabled-cr", "denied-namespace"} {
		t.Run(kind, func(t *testing.T) {
			clients, manager, job, review := newAuthorizedSchedule(t)
			ctx := scheduleContext(scheduleControllerIdentity)
			run := &model.Run{DisplayName: "tick", RecurringRunId: job.UUID}
			swfs := clients.SwfClient().ScheduledWorkflow(job.Namespace)
			switch kind {
			case "denied-namespace":
				review.denyRunCreation = true
			case "unknown":
				run.RecurringRunId = "unregistered-cr-uid"
			case "namespace":
				run.Namespace = "another-tenant"
			case "experiment":
				run.ExperimentId = "another-experiment"
			case "disabled-db":
				require.NoError(t, clients.JobStore().ChangeJobMode(job.UUID, false))
			case "disabled-cr":
				swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
				require.NoError(t, err)
				swf.Spec.Enabled = false
				_, err = swfs.Update(ctx, swf)
				require.NoError(t, err)
			case "replaced":
				swf, err := swfs.Get(ctx, job.K8SName, metav1.GetOptions{})
				require.NoError(t, err)
				swf.UID = "different-uid"
				_, err = swfs.Update(ctx, swf)
				require.NoError(t, err)
			}
			require.Error(t, manager.PrepareRecurringRun(ctx, run))
			require.Zero(t, clients.ExecClientFake.GetWorkflowCount())
		})
	}
}
