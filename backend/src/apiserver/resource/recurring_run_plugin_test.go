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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type concurrentCreationPluginHandler struct {
	calls        atomic.Int32
	sharedParent bool
	secondHook   chan struct{}
	resumeSecond chan struct{}
	endedParents chan string
}

func (*concurrentCreationPluginHandler) Name() string { return "test" }
func (*concurrentCreationPluginHandler) ResolveRunPluginInput(*string) (interface{}, bool, error) {
	return nil, true, nil
}
func (*concurrentCreationPluginHandler) ResolveRunPluginConfig(context.Context, kubernetes.Interface, string, string) (interface{}, error) {
	return struct{}{}, nil
}
func (*concurrentCreationPluginHandler) GetPluginOperationTimeout(interface{}) time.Duration {
	return time.Minute
}
func (h *concurrentCreationPluginHandler) OnBeforeRunCreation(context.Context, *apiserverPlugins.PendingRun, interface{}, interface{}) (*apiv2beta1.PluginOutput, []corev1.EnvVar, error) {
	call := h.calls.Add(1)
	parentID := fmt.Sprintf("parent-%d", call)
	if h.sharedParent {
		parentID = "parent-1"
	}
	if call == 2 {
		close(h.secondHook)
		<-h.resumeSecond
	}
	return &apiv2beta1.PluginOutput{Entries: map[string]*apiv2beta1.MetadataValue{
		apiserverPlugins.EntryRootRunID: {Value: structpb.NewStringValue(parentID)},
	}}, nil, nil
}
func (*concurrentCreationPluginHandler) HandleRetry(context.Context, *apiserverPlugins.PersistedRun, interface{}) error {
	return nil
}
func (h *concurrentCreationPluginHandler) OnRunEnd(_ context.Context, run *apiserverPlugins.PersistedRun, _ interface{}) (bool, error) {
	h.endedParents <- apiserverPlugins.GetParentRunID(run.PluginsOutput[h.Name()])
	run.PluginsOutput[h.Name()].State = apiv2beta1.PluginState_PLUGIN_SUCCEEDED
	return false, nil
}
func (*concurrentCreationPluginHandler) GetGenericFailedPluginOutput(string, string, interface{}) *apiv2beta1.PluginOutput {
	return nil
}

func TestCreateRunConcurrentPluginParents(t *testing.T) {
	for _, creatorPersisted := range []bool{false, true} {
		for _, sharedParent := range []bool{false, true} {
			t.Run(fmt.Sprintf("creatorPersisted=%t/sharedParent=%t", creatorPersisted, sharedParent), func(t *testing.T) {
				store, manager, experiment := initWithExperiment(t)
				defer store.Close()
				for key, value := range map[string]string{common.MultiUserMode: "true", v1AllowedNamespaces: "ns1"} {
					previous := viper.Get(key)
					viper.Set(key, value)
					t.Cleanup(func() { viper.Set(key, previous) })
				}
				manager.time = fixedRecurringTime{epoch: 200}
				ctx := multiUserContext()
				workflow := util.NewWorkflow(testWorkflow.DeepCopy())
				workflow.SetExecutionName("fixed-plugin-workflow")
				job, err := manager.CreateJob(ctx, &model.Job{
					DisplayName: "plugin-schedule", Namespace: "ns1", ExperimentId: experiment.UUID,
					Enabled: true, MaxConcurrency: 1,
					PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: model.LargeText(workflow.ToStringForStore())},
					Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
						PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
					}},
				})
				require.NoError(t, err)
				client := &pausedRecurringExecutionClient{
					ExecutionInterface: &uniqueRecurringWorkflowClient{ExecutionInterface: store.ExecClientFake.Execution(job.Namespace)},
					created:            make(chan util.ExecutionSpec, 1), resume: make(chan struct{}),
				}
				manager.execClient = &retryWorkflowExecClient{workflowClient: client}
				resumeCreator := sync.OnceFunc(func() { close(client.resume) })
				defer resumeCreator()
				handler := &concurrentCreationPluginHandler{
					sharedParent: sharedParent, secondHook: make(chan struct{}), resumeSecond: make(chan struct{}), endedParents: make(chan string, 4),
				}
				resumeLoser := sync.OnceFunc(func() { close(handler.resumeSecond) })
				defer resumeLoser()
				manager.pluginDispatcher, err = apiserverPlugins.NewRunPluginDispatcherImpl(
					[]apiserverPlugins.RunPluginHandler{handler}, manager.k8sCoreClient, manager.runStore)
				require.NoError(t, err)
				makeRun := func() *model.Run {
					run := &model.Run{DisplayName: "same-tick", RecurringRunId: job.UUID}
					require.NoError(t, manager.PrepareRecurringRun(ctx, run))
					return run
				}
				firstInput, secondInput := makeRun(), makeRun()
				type result struct {
					run *model.Run
					err error
				}
				create := func(run *model.Run) <-chan result {
					done := make(chan result, 1)
					go func() {
						created, err := manager.CreateRun(ctx, run)
						done <- result{created, err}
					}()
					return done
				}
				firstDone := create(firstInput)
				var execution util.ExecutionSpec
				select {
				case execution = <-client.created:
				case first := <-firstDone:
					t.Fatalf("creator did not submit its Workflow: %v", first.err)
				case <-time.After(10 * time.Second):
					t.Fatal("creator did not submit its Workflow")
				}
				secondDone := create(secondInput)
				select {
				case <-handler.secondHook:
				case second := <-secondDone:
					t.Fatalf("second request did not reach its plugin hook: %v", second.err)
				case <-time.After(10 * time.Second):
					t.Fatal("second request did not reach its plugin hook")
				}
				var first result
				if creatorPersisted {
					resumeCreator()
					first = <-firstDone
					require.NoError(t, first.err)
				}
				resumeLoser()
				second := <-secondDone
				if creatorPersisted {
					require.NoError(t, second.err)
					require.Equal(t, first.run.PluginsOutputString, second.run.PluginsOutputString)
				} else {
					require.True(t, util.IsUserErrorCodeMatch(second.err, codes.Unavailable), "error: %v", second.err)
					_, err := manager.GetRun(firstInput.UUID)
					require.True(t, util.IsUserErrorCodeMatch(err, codes.NotFound), "loser must not insert a run: %v", err)
					resumeCreator()
					first = <-firstDone
					require.NoError(t, first.err)
				}
				if sharedParent {
					require.Empty(t, handler.endedParents, "the shared parent must remain open")
				} else {
					require.Len(t, handler.endedParents, 1)
					require.Equal(t, "parent-2", <-handler.endedParents)
				}
				persisted, err := manager.GetRun(firstInput.UUID)
				require.NoError(t, err)
				require.Equal(t, first.run.PluginsOutputString, persisted.PluginsOutputString)
				require.Contains(t, string(*persisted.PluginsOutputString), "parent-1")
				require.NotContains(t, string(*persisted.PluginsOutputString), "parent-2")
				require.JSONEq(t, `{"test":"parent-1"}`, execution.ExecutionObjectMeta().Annotations[apiserverPlugins.AnnotationKeyPluginParents])
				require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
				ended, err := apiserverPlugins.ModelToPersistedRun(persisted, job.Namespace)
				require.NoError(t, err)
				ended.State = string(model.RuntimeStateSucceeded)
				require.True(t, manager.pluginDispatcher.OnRunEnd(ctx, ended))
				require.Len(t, handler.endedParents, 1)
				require.Equal(t, "parent-1", <-handler.endedParents)
			})
		}
	}
}
