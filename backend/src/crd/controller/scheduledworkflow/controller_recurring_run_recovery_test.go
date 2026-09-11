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

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	apicommon "github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/client"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swffake "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/fake"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ktesting "k8s.io/client-go/testing"
)

type retryingRunServiceClient struct {
	fakeRunServiceClient
	calls int
	keys  []string
	hints []int64
}

func (f *retryingRunServiceClient) CreateRun(ctx context.Context, request *api.CreateRunRequest, opts ...grpc.CallOption) (*api.Run, error) {
	f.calls++
	f.keys = append(f.keys, request.Run.DisplayName)
	f.hints = append(f.hints, request.Run.ScheduledAt.AsTime().Unix())
	if f.calls == 1 {
		return nil, errors.New("workflow creation temporarily failed after claim")
	}
	return &api.Run{DisplayName: f.keys[0], ScheduledAt: timestamppb.New(time.Unix(200, 0))}, nil
}

func TestSyncHandlerNoCatchupRetainsRequestKeyAfterSubmissionAndStatusFailures(t *testing.T) {
	swf := newTestSWFForAPIPath()
	swf.APIVersion = swfapi.SchemeGroupVersion.String()
	swf.Spec.Enabled = true
	swf.CreationTimestamp = metav1.NewTime(time.Unix(100, 0))
	swf.Spec.NoCatchup = commonutil.BoolPointer(true)
	swf.Spec.PeriodicSchedule = &swfapi.PeriodicSchedule{IntervalSecond: 10}
	fakeSWF := swffake.NewSimpleClientset(swf.Get())
	informer := swfinformers.NewSharedInformerFactory(fakeSWF, 0).Scheduledworkflow().V1beta1().ScheduledWorkflows()
	require.NoError(t, informer.Informer().GetStore().Add(swf.Get()))
	updates := 0
	var updated *swfapi.ScheduledWorkflow
	fakeSWF.PrependReactor("update", "scheduledworkflows", func(action ktesting.Action) (bool, runtime.Object, error) {
		updates++
		if updates == 1 {
			return true, nil, errors.New("status update temporarily failed")
		}
		updated = action.(ktesting.UpdateAction).GetObject().(*swfapi.ScheduledWorkflow).DeepCopy()
		return true, updated, nil
	})
	runs := &retryingRunServiceClient{}
	c := &Controller{
		swfClient:      client.NewScheduledWorkflowClient(fakeSWF, informer),
		workflowClient: client.NewWorkflowClient(&fakeExecutionClient{}, &fakeExecutionInformer{}),
		runClient:      runs, multiUser: true, location: time.UTC,
	}
	for i, now := range []int64{200, 300, 400} {
		c.time = &recurringRunRecoveryClock{now: now}
		_, retry, _, err := c.syncHandler(context.Background(), swf.Namespace+"/"+swf.Name)
		if i < 2 {
			require.Error(t, err)
			require.True(t, retry)
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, []string{swf.NextResourceName(), swf.NextResourceName(), swf.NextResourceName()}, runs.keys)
	require.Equal(t, []int64{200, 300, 400}, runs.hints)
	require.NotNil(t, updated)
	require.Equal(t, int64(200), updated.Status.Trigger.LastTriggeredTime.Unix())
	require.Equal(t, int64(1), *updated.Status.Trigger.LastIndex)
}

type recurringRunRecoveryClock struct{ now int64 }

func (c *recurringRunRecoveryClock) Now() time.Time { return time.Unix(c.now, 0) }

type inProcessRunServiceClient struct {
	fakeRunServiceClient
	server *server.RunServer
	run    *api.Run
	keys   []string
}

func (f *inProcessRunServiceClient) CreateRun(ctx context.Context, req *api.CreateRunRequest, opts ...grpc.CallOption) (*api.Run, error) {
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(apicommon.GoogleIAPUserIdentityHeader, apicommon.GoogleIAPUserIdentityPrefix+"user@google.com"))
	f.keys = append(f.keys, req.Run.DisplayName)
	run, err := f.server.CreateRun(ctx, req)
	if err == nil {
		f.run = run
	}
	return run, err
}

func TestSyncHandlerAcknowledgesDeletedRunAfterStatusFailure(t *testing.T) {
	originalMultiUser := viper.Get(apicommon.MultiUserMode)
	originalNamespace := viper.Get(apicommon.PodNamespace)
	originalV1Block := viper.Get(commonutil.BlockV1Pipelines)
	viper.Set(apicommon.MultiUserMode, "true")
	viper.Set(apicommon.PodNamespace, "ns1")
	viper.Set(commonutil.BlockV1Pipelines, "false")
	t.Cleanup(func() {
		viper.Set(apicommon.MultiUserMode, originalMultiUser)
		viper.Set(apicommon.PodNamespace, originalNamespace)
		viper.Set(commonutil.BlockV1Pipelines, originalV1Block)
	})
	proxy.InitializeConfigWithEmptyForTests()
	clock := &recurringRunRecoveryClock{now: 200}
	clients, err := resource.NewFakeClientManager(clock, commonutil.NewUUIDGenerator())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	experiment, err := manager.CreateExperiment(&model.Experiment{Name: "recovery", Namespace: "ns1"})
	require.NoError(t, err)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(apicommon.GoogleIAPUserIdentityHeader, apicommon.GoogleIAPUserIdentityPrefix+"user@google.com"))
	job, err := manager.CreateJob(ctx, &model.Job{
		DisplayName: "recovery", K8SName: "recovery", Namespace: "ns1", ExperimentId: experiment.UUID,
		Enabled: true, MaxConcurrency: 1, NoCatchup: true,
		Trigger: model.Trigger{PeriodicSchedule: model.PeriodicSchedule{
			PeriodicScheduleStartTimeInSec: commonutil.Int64Pointer(90), IntervalSecond: commonutil.Int64Pointer(10),
		}},
		PipelineSpec: model.PipelineSpec{WorkflowSpecManifest: model.LargeText(`{
   "apiVersion":"argoproj.io/v1alpha1", "kind":"Workflow",
   "metadata":{"generateName":"recovery-"},
   "spec":{"entrypoint":"main", "templates":[{"name":"main", "container":{"image":"alpine"}}]}
  }`)},
	})
	require.NoError(t, err)
	swf, err := clients.SwfClient().ScheduledWorkflow(job.Namespace).Get(ctx, job.K8SName, metav1.GetOptions{})
	require.NoError(t, err)
	fakeSWF := swffake.NewSimpleClientset()
	informer := swfinformers.NewSharedInformerFactory(fakeSWF, 0).Scheduledworkflow().V1beta1().ScheduledWorkflows()
	require.NoError(t, informer.Informer().GetStore().Add(swf))
	updateCount := 0
	var updated *swfapi.ScheduledWorkflow
	fakeSWF.PrependReactor("update", "scheduledworkflows", func(action ktesting.Action) (bool, runtime.Object, error) {
		updateCount++
		if updateCount == 1 {
			return true, nil, errors.New("status temporarily unavailable")
		}
		updated = action.(ktesting.UpdateAction).GetObject().(*swfapi.ScheduledWorkflow).DeepCopy()
		require.NoError(t, informer.Informer().GetStore().Update(updated))
		_, err := clients.SwfClient().ScheduledWorkflow(job.Namespace).Update(ctx, updated)
		return true, updated, err
	})
	runs := &inProcessRunServiceClient{server: server.NewRunServer(manager, &server.RunServerOptions{CollectMetrics: false})}
	c := &Controller{
		swfClient:      client.NewScheduledWorkflowClient(fakeSWF, informer),
		workflowClient: client.NewWorkflowClient(&fakeExecutionClient{}, &fakeExecutionInformer{}),
		runClient:      runs, multiUser: true, location: time.UTC, time: clock,
	}
	key := swf.Namespace + "/" + swf.Name
	_, retry, _, err := c.syncHandler(ctx, key)
	require.ErrorContains(t, err, "status temporarily unavailable")
	require.True(t, retry)
	require.NotNil(t, runs.run)
	firstRun := runs.run
	require.Equal(t, int64(200), firstRun.ScheduledAt.Seconds)
	state, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.False(t, state.Pending)
	require.Equal(t, int64(1), state.LastRunIndex)
	// The API committed the tick, but the controller has not acknowledged it.
	// Retention or a user may delete the run while the controller is recovering.
	require.NoError(t, manager.DeleteRun(ctx, firstRun.RunId))

	clock.now = 300
	again, retry, _, err := c.syncHandler(ctx, key)
	require.NoError(t, err)
	require.True(t, again)
	require.False(t, retry)
	require.Equal(t, firstRun.RunId, runs.run.RunId)
	require.Equal(t, firstRun.DisplayName, runs.run.DisplayName)
	require.Equal(t, firstRun.ScheduledAt, runs.run.ScheduledAt)
	require.NotNil(t, updated)
	require.Equal(t, int64(1), *updated.Status.Trigger.LastIndex)
	require.Equal(t, int64(200), updated.Status.Trigger.LastTriggeredTime.Unix(), "acknowledge the consumed tick's original API time")
	require.Equal(t, 2, updateCount)
	require.Equal(t, []string{runs.keys[0], runs.keys[0]}, runs.keys)
	require.Zero(t, clients.ExecClientFake.GetWorkflowCount(), "acknowledgement must not recreate the deleted Workflow")
	_, err = manager.GetRun(firstRun.RunId)
	require.Error(t, err, "acknowledgement must not recreate the deleted database row")
	afterAcknowledgement, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, state, afterAcknowledgement)

	// The refreshed controller cursor requests a new key for the next due tick.
	clock.now = 400
	again, retry, _, err = c.syncHandler(ctx, key)
	require.NoError(t, err)
	require.True(t, again)
	require.False(t, retry)
	require.NotEqual(t, firstRun.RunId, runs.run.RunId)
	require.Equal(t, int64(400), runs.run.ScheduledAt.Seconds)
	require.NotEqual(t, runs.keys[0], runs.keys[2])
	require.Equal(t, 1, clients.ExecClientFake.GetWorkflowCount())
	nextState, err := clients.JobStore().GetRecurringRunState(job.UUID)
	require.NoError(t, err)
	require.Equal(t, int64(2), nextState.LastRunIndex)
	require.Equal(t, int64(400), nextState.LastScheduledAtInSec)
	require.False(t, nextState.Pending)
}
