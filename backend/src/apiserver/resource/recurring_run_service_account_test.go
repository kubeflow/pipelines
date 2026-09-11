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
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type recurringRunAccountReview struct {
	allowedAccount string
	reviews        []*authzv1.SubjectAccessReview
}

func (c *recurringRunAccountReview) Create(_ context.Context, review *authzv1.SubjectAccessReview, _ metav1.CreateOptions) (*authzv1.SubjectAccessReview, error) {
	c.reviews = append(c.reviews, review.DeepCopy())
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{
		Allowed: review.Spec.ResourceAttributes.Resource != "serviceaccounts" ||
			review.Spec.ResourceAttributes.Name == c.allowedAccount,
	}}, nil
}

type recurringRunPluginDispatcher struct {
	apiserverPlugins.NoOpDispatcher
}

func (recurringRunPluginDispatcher) PluginsRegistered() bool { return true }

func configureRecurringRunAccountTest(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		common.MultiUserMode:              "true",
		common.AllowedServiceAccountsFlag: "embedded-runner,override-runner",
		v1AllowedNamespaces:               "ns1",
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
}

func TestCreateJobFollowLatestV1AuthorizesEmbeddedServiceAccount(t *testing.T) {
	for _, permitted := range []bool{false, true} {
		name := "denied"
		if permitted {
			name = "authorized"
		}
		t.Run(name, func(t *testing.T) {
			configureRecurringRunAccountTest(t)
			store, _, experiment := initWithExperiment(t)
			defer store.Close()
			review := &recurringRunAccountReview{}
			if permitted {
				review.allowedAccount = "embedded-runner"
			}
			store.SubjectAccessReviewClientFake = review
			manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
			workflow := util.NewWorkflow(testWorkflow.DeepCopy())
			workflow.Spec.ServiceAccountName = "embedded-runner"
			pipeline, err := manager.CreatePipeline(createPipeline("schedule-pipeline", "", "ns1"))
			require.NoError(t, err)
			_, err = manager.CreatePipelineVersion(createPipelineVersion(
				pipeline.UUID, "schedule-pipeline/v1", "v1", "", workflow.ToStringForStore(), "", "ns1"))
			require.NoError(t, err)

			job, err := manager.CreateJob(multiUserContext(), &model.Job{
				DisplayName:  "follow-latest",
				Namespace:    "ns1",
				ExperimentId: experiment.UUID,
				Enabled:      true,
				PipelineSpec: model.PipelineSpec{PipelineId: pipeline.UUID},
			})
			if !permitted {
				require.ErrorContains(t, err, "Unauthorized")
				require.Nil(t, job)
			} else {
				require.NoError(t, err)
				stored, err := manager.GetJob(job.UUID)
				require.NoError(t, err)
				require.Equal(t, "embedded-runner", stored.ServiceAccount)
				require.Empty(t, stored.PipelineVersionId, "the schedule must continue following the latest version")
			}
			require.NotEmpty(t, review.reviews)
			last := review.reviews[len(review.reviews)-1]
			require.Equal(t, authzv1.ResourceAttributes{
				Verb: "use", Resource: "serviceaccounts", Name: "embedded-runner", Namespace: "ns1",
			}, *last.Spec.ResourceAttributes)
		})
	}
}

func TestCreateJobPluginV1PreservesAuthorizedServiceAccountOverride(t *testing.T) {
	configureRecurringRunAccountTest(t)
	store, _, experiment := initWithExperiment(t)
	defer store.Close()
	review := &recurringRunAccountReview{allowedAccount: "override-runner"}
	store.SubjectAccessReviewClientFake = review
	manager := NewResourceManager(store, &ResourceManagerOptions{CollectMetrics: false})
	manager.pluginDispatcher = recurringRunPluginDispatcher{}
	workflow := util.NewWorkflow(testWorkflow.DeepCopy())
	workflow.Spec.ServiceAccountName = "embedded-runner"
	job, err := manager.CreateJob(multiUserContext(), &model.Job{
		DisplayName:    "plugin-schedule",
		Namespace:      "ns1",
		ExperimentId:   experiment.UUID,
		Enabled:        true,
		ServiceAccount: "override-runner",
		PipelineSpec:   model.PipelineSpec{WorkflowSpecManifest: model.LargeText(workflow.ToStringForStore())},
	})
	require.NoError(t, err)
	stored, err := manager.GetJob(job.UUID)
	require.NoError(t, err)
	require.Equal(t, "override-runner", stored.ServiceAccount)
	swf, err := store.SwfClient().ScheduledWorkflow(job.Namespace).Get(context.Background(), job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Nil(t, swf.Spec.Workflow, "plugin-enabled schedules use the API instead of embedding a workflow")

	run := &model.Run{DisplayName: "scheduled-tick", RecurringRunId: job.UUID, ServiceAccount: "embedded-runner"}
	require.NoError(t, manager.PrepareRecurringRun(multiUserContext(), run))
	require.Equal(t, "override-runner", run.ServiceAccount)
	require.Equal(t, stored.PipelineSpec, run.PipelineSpec)
	created, err := manager.CreateRun(multiUserContext(), run)
	require.NoError(t, err)
	require.Equal(t, "override-runner", created.ServiceAccount)
	require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
	require.GreaterOrEqual(t, len(review.reviews), 2)
	for _, check := range review.reviews {
		if check.Spec.ResourceAttributes.Resource == "serviceaccounts" {
			require.Equal(t, "override-runner", check.Spec.ResourceAttributes.Name)
		}
	}
}

func TestAuthorizeStoredRunServiceAccount(t *testing.T) {
	for key, value := range map[string]string{
		common.MultiUserMode:                           "false",
		common.DefaultPipelineRunnerServiceAccountFlag: "pipeline-runner",
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	manifest := func(account string) model.LargeText {
		workflow := util.NewWorkflow(testWorkflow.DeepCopy())
		workflow.Spec.ServiceAccountName = account
		workflow.SetExecutionName("stored-workflow")
		workflow.SetExecutionNamespace("ns1")
		return model.LargeText(workflow.ToStringForStore())
	}
	for _, test := range []struct {
		name             string
		storedAccount    string
		workflowManifest model.LargeText
		pipelineManifest model.LargeText
		allowedAccount   string
		wantError        string
	}{
		{
			name: "stored account takes precedence", storedAccount: "stored-runner",
			workflowManifest: manifest("embedded-runner"), allowedAccount: "stored-runner",
		},
		{
			name: "stored account cannot be bypassed by an allowed manifest account", storedAccount: "stored-runner",
			workflowManifest: manifest("embedded-runner"), allowedAccount: "embedded-runner", wantError: "not allowed",
		},
		{
			name: "stored account does not require a runtime manifest", storedAccount: "stored-runner",
			allowedAccount: "stored-runner",
		},
		{
			name: "stored account does not parse an unused manifest", storedAccount: "stored-runner",
			workflowManifest: "{", allowedAccount: "stored-runner",
		},
		{
			name: "workflow runtime account allowed", workflowManifest: manifest("embedded-runner"),
			allowedAccount: "embedded-runner",
		},
		{
			name: "workflow runtime account denied", workflowManifest: manifest("embedded-runner"),
			wantError: "not allowed",
		},
		{
			name: "pipeline runtime account allowed", pipelineManifest: manifest("embedded-runner"),
			allowedAccount: "embedded-runner",
		},
		{
			name: "pipeline runtime account denied", pipelineManifest: manifest("embedded-runner"),
			wantError: "not allowed",
		},
		{
			name:             "workflow runtime takes precedence over pipeline runtime",
			workflowManifest: manifest("stored-runner"), pipelineManifest: manifest("embedded-runner"),
			allowedAccount: "embedded-runner", wantError: "not allowed",
		},
		{name: "legitimate empty account", workflowManifest: manifest("")},
		{name: "missing execution identity", wantError: "execution identity is missing"},
		{name: "malformed workflow runtime", workflowManifest: "{", wantError: "stored execution account"},
		{name: "malformed pipeline runtime", pipelineManifest: "{", wantError: "stored execution account"},
		{name: "null execution identity", workflowManifest: "null", wantError: "execution name is missing"},
		{name: "empty execution identity", workflowManifest: "{}", wantError: "execution name is missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			previous := viper.Get(common.AllowedServiceAccountsFlag)
			viper.Set(common.AllowedServiceAccountsFlag, test.allowedAccount)
			t.Cleanup(func() { viper.Set(common.AllowedServiceAccountsFlag, previous) })
			run := &model.Run{
				UUID: "stored-run", Namespace: "ns1", ServiceAccount: test.storedAccount,
				RunDetails: model.RunDetails{
					WorkflowRuntimeManifest: test.workflowManifest,
					PipelineRuntimeManifest: test.pipelineManifest,
				},
			}
			before := *run
			manager := &ResourceManager{}
			err := manager.authorizeStoredRunServiceAccount(context.Background(), run)
			if test.wantError != "" {
				require.ErrorContains(t, err, test.wantError)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, before, *run, "authorization must not replace retained execution metadata")
		})
	}
}
