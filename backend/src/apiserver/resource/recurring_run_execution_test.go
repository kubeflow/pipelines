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

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	apiserverPlugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// The shared fake overwrites fixed names. Match Kubernetes' atomic name
// uniqueness here so a second submission cannot silently replace a workflow.
type uniqueRecurringWorkflowClient struct {
	util.ExecutionInterface
	mu sync.Mutex
}

func (c *uniqueRecurringWorkflowClient) Create(ctx context.Context, execution util.ExecutionSpec, options metav1.CreateOptions) (util.ExecutionSpec, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if execution.ExecutionName() != "" {
		_, err := c.Get(ctx, execution.ExecutionName(), metav1.GetOptions{})
		if err == nil {
			return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "argoproj.io", Resource: "workflows"}, execution.ExecutionName())
		}
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
	}
	return c.ExecutionInterface.Create(ctx, execution, options)
}

func recurringExecutionFixture(run *model.Run) *util.Workflow {
	workflow := util.NewWorkflow(testWorkflow.DeepCopy())
	workflow.SetExecutionNamespace("ns1")
	workflow.SetLabels(util.LabelKeyWorkflowRunId, run.UUID)
	workflow.SetAnnotations(util.AnnotationKeyRunName, run.DisplayName)
	workflow.SetOwnerReferences(&swfapi.ScheduledWorkflow{ObjectMeta: metav1.ObjectMeta{
		Name: "schedule", UID: "schedule-uid",
	}})
	return workflow
}

func TestCreateRunExecutionReusesRecurringWorkflow(t *testing.T) {
	store, manager, _ := initWithExperiment(t)
	defer store.Close()
	manager.execClient = &retryWorkflowExecClient{workflowClient: &uniqueRecurringWorkflowClient{ExecutionInterface: store.ExecClientFake.Execution("ns1")}}
	run := &model.Run{UUID: "tick-uuid", DisplayName: "tick", RecurringRunId: "schedule-uid"}
	ctx := context.Background()
	first, created, err := manager.createRunExecution(ctx, run, recurringExecutionFixture(run))
	require.NoError(t, err)
	require.True(t, created)
	second, created, err := manager.createRunExecution(ctx, run, recurringExecutionFixture(run))
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, first.ExecutionUID(), second.ExecutionUID())
	require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())

	run.UUID = "next-tick-uuid"
	run.DisplayName = "next-tick"
	next, created, err := manager.createRunExecution(ctx, run, recurringExecutionFixture(run))
	require.NoError(t, err)
	require.True(t, created)
	require.NotEqual(t, first.ExecutionName(), next.ExecutionName())
	require.Equal(t, 2, store.ExecClientFake.GetWorkflowCount())
}

func TestCreateRunExecutionPreservesOrdinaryWorkflowNames(t *testing.T) {
	store, manager, _ := initWithExperiment(t)
	defer store.Close()
	run := &model.Run{UUID: "ordinary-run"}
	workflow := recurringExecutionFixture(run)
	workflow.Name = ""
	workflow.GenerateName = "ordinary-"
	created, newExecution, err := manager.createRunExecution(context.Background(), run, workflow)
	require.NoError(t, err)
	require.True(t, newExecution)
	require.Equal(t, "ordinary-0", created.ExecutionName())
}

func TestCreateRunExecutionRejectsConflictingWorkflow(t *testing.T) {
	for name, mutate := range map[string]func(*util.Workflow){
		"name":          func(w *util.Workflow) { w.Name = "another-workflow" },
		"namespace":     func(w *util.Workflow) { w.Namespace = "another-namespace" },
		"missing UID":   func(w *util.Workflow) { w.UID = "" },
		"account":       func(w *util.Workflow) { w.Spec.ServiceAccountName = "another-account" },
		"run ID":        func(w *util.Workflow) { w.Labels[util.LabelKeyWorkflowRunId] = "another-run" },
		"run name":      func(w *util.Workflow) { w.Annotations[util.AnnotationKeyRunName] = "another-tick" },
		"owner UID":     func(w *util.Workflow) { w.OwnerReferences[0].UID = "another-schedule-uid" },
		"owner name":    func(w *util.Workflow) { w.OwnerReferences[0].Name = "another-schedule" },
		"owner kind":    func(w *util.Workflow) { w.OwnerReferences[0].Kind = "Pod" },
		"owner version": func(w *util.Workflow) { w.OwnerReferences[0].APIVersion = "v1" },
	} {
		t.Run(name, func(t *testing.T) {
			store, manager, _ := initWithExperiment(t)
			defer store.Close()
			manager.execClient = &retryWorkflowExecClient{workflowClient: &uniqueRecurringWorkflowClient{ExecutionInterface: store.ExecClientFake.Execution("ns1")}}
			run := &model.Run{UUID: "tick-uuid", DisplayName: "tick", RecurringRunId: "schedule-uid"}
			first := recurringExecutionFixture(run)
			_, _, err := manager.createRunExecution(context.Background(), run, first)
			require.NoError(t, err)
			mutate(first)
			_, created, err := manager.createRunExecution(context.Background(), run, recurringExecutionFixture(run))
			require.ErrorContains(t, err, "existing workflow identity does not match")
			require.False(t, created)
			require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
		})
	}
}

func TestCreateRunRecurringTicksReplaceFixedWorkflowName(t *testing.T) {
	store, manager, job := initWithJob(t)
	defer store.Close()
	manager.execClient = &retryWorkflowExecClient{workflowClient: &uniqueRecurringWorkflowClient{ExecutionInterface: store.ExecClientFake.Execution(job.Namespace)}}
	var names []string
	for _, displayName := range []string{"first-tick", "second-tick"} {
		run, err := manager.CreateRun(context.Background(), &model.Run{
			DisplayName: displayName, RecurringRunId: job.UUID, ExperimentId: job.ExperimentId, PipelineSpec: job.PipelineSpec,
		})
		require.NoError(t, err)
		names = append(names, run.K8SName)
	}
	require.NotEqual(t, names[0], names[1])
	require.Equal(t, 2, store.ExecClientFake.GetWorkflowCount())
}

type fixedRecurringTime struct{ epoch int64 }

func (c fixedRecurringTime) Now() time.Time { return time.Unix(c.epoch, 0) }

type pausedRecurringRunDispatcher struct {
	apiserverPlugins.NoOpDispatcher
	calls       atomic.Int32
	endCalls    atomic.Int32
	outputs     bool
	firstHook   chan struct{}
	resumeFirst chan struct{}
}

func (d *pausedRecurringRunDispatcher) OnBeforeRunCreation(_ context.Context, run *apiserverPlugins.PendingRun, execution util.ExecutionSpec) error {
	call := d.calls.Add(1)
	if d.outputs {
		output := fmt.Sprintf(`{"test":{"entries":{"id":{"value":"parent-%d"}}}}`, call)
		run.PluginsOutput = &output
		execution.SetAnnotations("test/plugin-parent", fmt.Sprintf("parent-%d", call))
	}
	if call == 1 && d.firstHook != nil {
		close(d.firstHook)
		<-d.resumeFirst
	}
	return nil
}

func (d *pausedRecurringRunDispatcher) OnRunEnd(context.Context, *apiserverPlugins.PersistedRun) bool {
	d.endCalls.Add(1)
	return true
}

func TestCreateRunConcurrentAuthorizedTickCreatesOneWorkflow(t *testing.T) {
	for _, test := range []struct {
		name    string
		version string
		plugins bool
	}{
		{name: "v1", version: "v1"},
		{name: "v2", version: "v2"},
		{name: "v2 with plugin output", version: "v2", plugins: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, manager, experiment := initWithExperiment(t)
			defer store.Close()
			for key, value := range map[string]string{common.MultiUserMode: "true", v1AllowedNamespaces: "ns1"} {
				previous := viper.Get(key)
				viper.Set(key, value)
				t.Cleanup(func() { viper.Set(key, previous) })
			}
			manager.time = fixedRecurringTime{epoch: 200}
			ctx := multiUserContext()
			job := &model.Job{
				DisplayName: "pinned-schedule", Namespace: "ns1", ExperimentId: experiment.UUID,
				Enabled: true, MaxConcurrency: 1,
				Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
					PeriodicScheduleStartTimeInSec: util.Int64Pointer(100), IntervalSecond: util.Int64Pointer(10),
				}},
				PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: model.LargeText(testWorkflow.ToStringForStore())},
			}
			if test.version == "v2" {
				job.PipelineSpec = model.PipelineSpec{
					PipelineSpecManifest: model.LargeText(v2SpecHelloWorld),
					RuntimeConfig:        model.RuntimeConfig{Parameters: `{"text":"world"}`, PipelineRoot: "schedule-root"},
				}
			}
			job, err := manager.CreateJob(ctx, job)
			require.NoError(t, err)
			manager.execClient = &retryWorkflowExecClient{workflowClient: &uniqueRecurringWorkflowClient{ExecutionInterface: store.ExecClientFake.Execution(job.Namespace)}}
			paused := &pausedRecurringRunDispatcher{firstHook: make(chan struct{}), resumeFirst: make(chan struct{}), outputs: test.plugins}
			manager.pluginDispatcher = paused
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
			firstDone := make(chan result, 1)
			go func() {
				run, err := manager.CreateRun(ctx, firstInput)
				firstDone <- result{run, err}
			}()
			select {
			case <-paused.firstHook:
			case first := <-firstDone:
				t.Fatalf("first request did not reach workflow submission: %v", first.err)
			case <-time.After(10 * time.Second):
				t.Fatal("first request did not reach workflow submission")
			}
			second, secondErr := manager.CreateRun(ctx, secondInput)
			close(paused.resumeFirst)
			first := <-firstDone
			require.NoError(t, secondErr)
			require.NoError(t, first.err)
			require.Equal(t, first.run.UUID, second.UUID)
			require.Equal(t, first.run.K8SName, second.K8SName)
			require.Equal(t, int64(110), second.ScheduledAtInSec)
			require.Equal(t, 1, store.ExecClientFake.GetWorkflowCount())
			if test.plugins {
				require.NotNil(t, second.PluginsOutputString)
				require.Contains(t, string(*second.PluginsOutputString), "parent-2")
				require.Equal(t, second.PluginsOutputString, first.run.PluginsOutputString)
				workflow, err := manager.getWorkflowClient(job.Namespace).Get(ctx, second.K8SName, metav1.GetOptions{})
				require.NoError(t, err)
				require.Equal(t, "parent-2", workflow.ExecutionObjectMeta().Annotations["test/plugin-parent"])
				require.Zero(t, paused.endCalls.Load())
			}
			state, err := store.JobStore().GetRecurringRunState(job.UUID)
			require.NoError(t, err)
			require.Equal(t, int64(1), state.LastRunIndex)
			require.False(t, state.Pending)
		})
	}
}
