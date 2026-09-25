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
	"strings"
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type replayCountingPlugin struct {
	beforeCalls int
	endCalls    int
}

func (*replayCountingPlugin) Name() string    { return "replay-test" }
func (*replayCountingPlugin) IsEnabled() bool { return true }
func (p *replayCountingPlugin) Create() (plugins.RunPluginHandler, error) {
	return p, nil
}
func (*replayCountingPlugin) ResolveRunPluginInput(*string) (interface{}, bool, error) {
	return nil, true, nil
}
func (*replayCountingPlugin) ResolveRunPluginConfig(context.Context, kubernetes.Interface, string, string) (interface{}, error) {
	return struct{}{}, nil
}
func (*replayCountingPlugin) GetPluginOperationTimeout(interface{}) time.Duration {
	return time.Minute
}
func (p *replayCountingPlugin) OnBeforeRunCreation(context.Context, *plugins.PendingRun, interface{}, interface{}) (*api.PluginOutput, []corev1.EnvVar, error) {
	p.beforeCalls++
	return &api.PluginOutput{Entries: map[string]*api.MetadataValue{
		plugins.EntryRootRunID: {Value: structpb.NewStringValue("original-parent")},
	}}, nil, nil
}
func (*replayCountingPlugin) HandleRetry(context.Context, *plugins.PersistedRun, interface{}) error {
	return nil
}
func (p *replayCountingPlugin) OnRunEnd(context.Context, *plugins.PersistedRun, interface{}) (bool, error) {
	p.endCalls++
	return false, nil
}
func (*replayCountingPlugin) GetGenericFailedPluginOutput(string, string, interface{}) *api.PluginOutput {
	return nil
}

type singleUserRecurringReplay struct {
	clients *resource.FakeClientManager
	manager *resource.ResourceManager
	server  *RunServer
	plugin  *replayCountingPlugin
	request *api.CreateRunRequest
	first   *model.Run
}

func newSingleUserRecurringReplay(t *testing.T, pinned bool, serviceAccount string) *singleUserRecurringReplay {
	t.Helper()
	initEnvVars()
	for key, value := range map[string]string{
		common.MultiUserMode:              "false",
		common.AllowedServiceAccountsFlag: serviceAccount,
	} {
		previous := viper.Get(key)
		viper.Set(key, value)
		t.Cleanup(func() { viper.Set(key, previous) })
	}
	factories := plugins.RegisteredFactories()
	plugins.ResetRegistry()
	t.Cleanup(func() {
		plugins.ResetRegistry()
		for _, factory := range factories {
			plugins.RegisterHandlerFactory(factory)
		}
	})
	plugin := &replayCountingPlugin{}
	plugins.RegisterHandlerFactory(plugin)
	clients, err := resource.NewFakeClientManager(util.NewFakeTime(time.Unix(200, 0)), util.NewUUIDGenerator())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "replay", Namespace: "ns1"})
	require.NoError(t, err)
	pipeline, err := manager.CreatePipeline(&model.Pipeline{Name: "single-user-replay", Namespace: "ns1"})
	require.NoError(t, err)
	version, err := manager.CreatePipelineVersion(&model.PipelineVersion{
		Name: "version-a", PipelineId: pipeline.UUID, PipelineSpec: model.LargeText(v2SpecHelloWorld),
	})
	require.NoError(t, err)
	jobSpec := model.PipelineSpec{
		PipelineId: pipeline.UUID,
		RuntimeConfig: model.RuntimeConfig{
			Parameters: `{"param1":"world"}`, PipelineRoot: "minio://pipeline-root",
		},
	}
	if pinned {
		jobSpec.PipelineVersionId = version.UUID
	}
	ctx := context.Background()
	job, err := manager.CreateJob(ctx, &model.Job{
		DisplayName: "replay-schedule", Namespace: "ns1", ExperimentId: experiment.UUID, Enabled: true,
		Trigger:      model.Trigger{PeriodicSchedule: model.PeriodicSchedule{IntervalSecond: util.Int64Pointer(10)}},
		PipelineSpec: jobSpec, ServiceAccount: serviceAccount,
	})
	require.NoError(t, err)
	swf, err := clients.SwfClient().ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	if swf.Spec.Workflow != nil {
		require.Nil(t, swf.Spec.Workflow.Spec, "this schedule submits through the CreateRun API")
	}
	request := &api.CreateRunRequest{Run: &api.Run{
		DisplayName: "same-tick", RecurringRunId: job.UUID, ExperimentId: job.ExperimentId,
		PipelineSource: &api.Run_PipelineVersionReference{PipelineVersionReference: &api.PipelineVersionReference{
			PipelineId: swf.Spec.PipelineId, PipelineVersionId: swf.Spec.PipelineVersionId,
		}},
		RuntimeConfig: &api.RuntimeConfig{
			Parameters: map[string]*structpb.Value{"param1": structpb.NewStringValue("world")}, PipelineRoot: "minio://pipeline-root",
		},
		ServiceAccount: swf.Spec.ServiceAccount,
	}}
	server := createRunServer(manager)
	first, err := server.CreateRun(ctx, request)
	require.NoError(t, err)
	stored, err := manager.GetRun(first.RunId)
	require.NoError(t, err)
	require.Equal(t, version.UUID, stored.PipelineVersionId)
	require.Equal(t, 1, plugin.beforeCalls)
	require.Zero(t, plugin.endCalls)
	require.NotNil(t, stored.PluginsOutputString)
	require.Contains(t, string(*stored.PluginsOutputString), "original-parent")
	return &singleUserRecurringReplay{clients: clients, manager: manager, server: server, plugin: plugin, request: request, first: stored}
}

func (f *singleUserRecurringReplay) requireOriginalRunUnchanged(t *testing.T) {
	t.Helper()
	stored, err := f.manager.GetRun(f.first.UUID)
	require.NoError(t, err)
	require.Equal(t, f.first, stored)
	require.Equal(t, 1, f.clients.ExecClientFake.GetWorkflowCount())
	require.Equal(t, 1, f.plugin.beforeCalls)
	require.Zero(t, f.plugin.endCalls)
}

func TestSingleUserRecurringRunReplayAfterIncompatibleLatestVersion(t *testing.T) {
	fixture := newSingleUserRecurringReplay(t, false, "")
	versionB, err := fixture.manager.CreatePipelineVersion(&model.PipelineVersion{
		Name: "version-b", PipelineId: fixture.first.PipelineId,
		PipelineSpec: model.LargeText(strings.ReplaceAll(v2SpecHelloWorld, "param1", "renamed")),
	})
	require.NoError(t, err)
	latest, err := fixture.manager.GetLatestPipelineVersion(fixture.first.PipelineId)
	require.NoError(t, err)
	require.Equal(t, versionB.UUID, latest.UUID)

	replayed, err := fixture.server.CreateRun(context.Background(), fixture.request)
	require.NoError(t, err)
	require.Equal(t, fixture.first.UUID, replayed.RunId)
	fixture.requireOriginalRunUnchanged(t)

	// Acknowledging the old tick must not exempt a new tick from input validation.
	fixture.request.Run.DisplayName = "next-tick"
	_, err = fixture.server.CreateRun(context.Background(), fixture.request)
	require.ErrorContains(t, err, "parameter renamed is not optional")
	fixture.requireOriginalRunUnchanged(t)
}

func TestSingleUserRecurringRunReplayAfterPinnedVersionDeletion(t *testing.T) {
	fixture := newSingleUserRecurringReplay(t, true, "")
	require.NoError(t, fixture.manager.DeletePipelineVersion(fixture.first.PipelineVersionId))

	replayed, err := fixture.server.CreateRun(context.Background(), fixture.request)
	require.NoError(t, err)
	require.Equal(t, fixture.first.UUID, replayed.RunId)
	fixture.requireOriginalRunUnchanged(t)

	fixture.request.Run.DisplayName = "next-tick"
	_, err = fixture.server.CreateRun(context.Background(), fixture.request)
	require.ErrorContains(t, err, "not found")
	fixture.requireOriginalRunUnchanged(t)
}

func TestSingleUserRecurringRunReplayAuthorizesStoredAccount(t *testing.T) {
	for _, storedAccountAllowed := range []bool{false, true} {
		name := "removed-from-allowlist"
		if storedAccountAllowed {
			name = "still-allowed"
		}
		t.Run(name, func(t *testing.T) {
			fixture := newSingleUserRecurringReplay(t, false, "original-runner")
			if storedAccountAllowed {
				// A replay does not execute the account supplied by the new request.
				fixture.request.Run.ServiceAccount = "unlisted-request-account"
			} else {
				viper.Set(common.AllowedServiceAccountsFlag, "")
				fixture.request.Run.ServiceAccount = common.DefaultPipelineRunnerServiceAccount
			}
			replayed, err := fixture.server.CreateRun(context.Background(), fixture.request)
			if storedAccountAllowed {
				require.NoError(t, err)
				require.Equal(t, fixture.first.UUID, replayed.RunId)
				require.Equal(t, "original-runner", replayed.ServiceAccount)
			} else {
				require.ErrorContains(t, err, `service account "original-runner" is not allowed`)
				require.Nil(t, replayed)
			}
			fixture.requireOriginalRunUnchanged(t)
		})
	}
}
