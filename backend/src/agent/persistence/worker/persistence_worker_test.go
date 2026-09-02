// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	client "github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	testingclock "k8s.io/utils/clock/testing"
)

type FakeResourceEventHandlerRegistration struct {
	synced chan struct{}
}

func (h *FakeResourceEventHandlerRegistration) HasSynced() bool {
	return true
}

func (h *FakeResourceEventHandlerRegistration) HasSyncedChecker() cache.DoneChecker {
	return h
}

func (h *FakeResourceEventHandlerRegistration) Name() string {
	return "fake resource event handler registration"
}

func (h *FakeResourceEventHandlerRegistration) Done() <-chan struct{} {
	return h.synced
}

func NewFakeResourceEventHandlerRegistration() *FakeResourceEventHandlerRegistration {
	synced := make(chan struct{})
	close(synced)
	return &FakeResourceEventHandlerRegistration{synced: synced}
}

type FakeEventHandler struct {
	handler cache.ResourceEventHandler
}

func NewFakeEventHandler() *FakeEventHandler {
	return &FakeEventHandler{}
}

func TestFakeResourceEventHandlerRegistration_HasSyncedChecker(t *testing.T) {
	registration := NewFakeResourceEventHandlerRegistration()

	assert.True(t, registration.HasSynced())
	assert.True(t, cache.IsDone(registration.HasSyncedChecker()))
}

func (h *FakeEventHandler) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	h.handler = handler
	return NewFakeResourceEventHandlerRegistration(), nil
}

func TestPersistenceWorker_Success(t *testing.T) {
	// Set up workflow client
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
	})
	workflowClient := client.NewWorkflowClientFake()
	workflowClient.Put("MY_NAMESPACE", "MY_NAME", workflow)

	// Set up pipeline client
	pipelineClient := client.NewPipelineClientFake()

	// Set up persistence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker, err := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)
	require.NoError(t, err)

	// Test
	eventHandler.handler.OnAdd(workflow, true)
	worker.processNextWorkItem()
	assert.Equal(t, workflow, pipelineClient.GetWorkflow("MY_NAMESPACE", "MY_NAME"))
	assert.Equal(t, 0, worker.Len())
}

func TestPersistenceWorker_NotFoundError(t *testing.T) {
	// Set up workflow client
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})
	workflowClient := client.NewWorkflowClientFake()

	// Set up pipeline client
	pipelineClient := client.NewPipelineClientFake()

	// Set up persistence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker, err := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)
	require.NoError(t, err)

	// Test
	eventHandler.handler.OnAdd(workflow, true)
	worker.processNextWorkItem()
	assert.Nil(t, pipelineClient.GetWorkflow("MY_NAMESPACE", "MY_NAME"))
	assert.Equal(t, 0, worker.Len())
}

func TestPersistenceWorker_GetWorklowError(t *testing.T) {
	// Set up workflow client
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})
	workflowClient := client.NewWorkflowClientFake()
	workflowClient.Put("MY_NAMESPACE", "MY_NAME", nil)

	// Set up pipeline client
	pipelineClient := client.NewPipelineClientFake()

	// Set up persistence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker, err := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)
	require.NoError(t, err)

	// Test
	eventHandler.handler.OnAdd(workflow, true)
	worker.processNextWorkItem()
	assert.Nil(t, pipelineClient.GetWorkflow("MY_NAMESPACE", "MY_NAME"))
	assert.Equal(t, 1, worker.Len())
}

func TestPersistenceWorker_ReportWorkflowRetryableError(t *testing.T) {
	// Set up workflow client
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
			Labels:    map[string]string{util.LabelKeyWorkflowRunId: "MY_UUID"},
		},
	})
	workflowClient := client.NewWorkflowClientFake()
	workflowClient.Put("MY_NAMESPACE", "MY_NAME", workflow)

	// Set up pipeline client
	pipelineClient := client.NewPipelineClientFake()
	pipelineClient.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_TRANSIENT,
		"My Retriable Error"))

	// Set up persistence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker, err := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)
	require.NoError(t, err)

	// Test
	eventHandler.handler.OnAdd(workflow, true)
	worker.processNextWorkItem()
	assert.Nil(t, pipelineClient.GetWorkflow("MY_NAMESPACE", "MY_NAME"))
	assert.Equal(t, 1, worker.Len())
}

func TestPersistenceWorker_ReportWorkflowNonRetryableError(t *testing.T) {
	// Set up workflow client
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "MY_NAMESPACE",
			Name:      "MY_NAME",
		},
	})
	workflowClient := client.NewWorkflowClientFake()
	workflowClient.Put("MY_NAMESPACE", "MY_NAME", workflow)

	// Set up pipeline client
	pipelineClient := client.NewPipelineClientFake()
	pipelineClient.SetError(util.NewCustomError(fmt.Errorf("Error"), util.CUSTOM_CODE_PERMANENT,
		"My Permanent Error"))

	// Set up peristence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker, err := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)
	require.NoError(t, err)

	// Test
	eventHandler.handler.OnAdd(workflow, true)
	worker.processNextWorkItem()
	assert.Nil(t, pipelineClient.GetWorkflow("MY_NAMESPACE", "MY_NAME"))
	assert.Equal(t, 0, worker.Len())
}

// Ordinary informer updates must not consume retry backoff or delay a terminal
// state just because the workflow emitted many intermediate status updates.
func TestPersistenceWorker_NormalUpdatesPersistTerminalStateImmediately(t *testing.T) {
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team", Name: "run", Labels: map[string]string{util.LabelKeyWorkflowRunId: "run-id"}},
		Status:     workflowapi.WorkflowStatus{Phase: workflowapi.WorkflowRunning},
	})
	workflowClient := client.NewWorkflowClientFake()
	pipelineClient := client.NewPipelineClientFake()
	events := NewFakeEventHandler()
	worker, err := NewPersistenceWorker(util.NewFakeTimeForEpoch(), "normal-events", events, true,
		NewWorkflowSaver(workflowClient, pipelineClient, 100))
	require.NoError(t, err)
	t.Cleanup(worker.Shutdown)
	workflowClient.Put("team", "run", workflow)
	events.handler.OnAdd(workflow, true)
	for version := 1; version <= 10; version++ {
		next := util.NewWorkflow(workflow.DeepCopy())
		next.ResourceVersion = fmt.Sprint(version)
		if version == 10 {
			next.Status.Phase = workflowapi.WorkflowSucceeded
		}
		workflowClient.Put("team", "run", next)
		events.handler.OnUpdate(workflow, next)
		workflow = next
	}
	assert.Zero(t, worker.workqueue.NumRequeues("team/run"), "normal events must not consume failure retry budget")
	require.Equal(t, 1, worker.Len(), "the terminal update must be immediately ready, without waiting for a retry timer")
	require.True(t, worker.processNextWorkItem())
	require.NotNil(t, pipelineClient.GetWorkflow("team", "run"))
	assert.Equal(t, workflowapi.WorkflowSucceeded, workflowapi.WorkflowPhase(pipelineClient.GetWorkflow("team", "run").ExecutionStatus().Condition()))
	assert.Zero(t, worker.Len())
}

type recordingPersistenceRateLimiter struct {
	workqueue.TypedRateLimiter[any]
	delays []time.Duration
}

func (r *recordingPersistenceRateLimiter) When(item any) time.Duration {
	delay := r.TypedRateLimiter.When(item)
	r.delays = append(r.delays, delay)
	return delay
}

func TestPersistenceWorker_TransientFailureRetainsBackoffAndSuccessResetsIt(t *testing.T) {
	workflow := util.NewWorkflow(&workflowapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team", Name: "run", Labels: map[string]string{util.LabelKeyWorkflowRunId: "run-id"}},
		Status:     workflowapi.WorkflowStatus{Phase: workflowapi.WorkflowSucceeded},
	})
	workflowClient := client.NewWorkflowClientFake()
	workflowClient.Put("team", "run", workflow)
	pipelineClient := client.NewPipelineClientFake()
	pipelineClient.SetError(util.NewCustomError(fmt.Errorf("unavailable"), util.CUSTOM_CODE_TRANSIENT, "retry report"))
	events := NewFakeEventHandler()
	worker, err := NewPersistenceWorker(util.NewFakeTimeForEpoch(), "retry-events", events, true,
		NewWorkflowSaver(workflowClient, pipelineClient, 100))
	require.NoError(t, err)
	worker.Shutdown()
	clock := testingclock.NewFakeClock(time.Now())
	limiter := &recordingPersistenceRateLimiter{TypedRateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[any](DefaultJobBackOff, MaxJobBackOff)}
	worker.workqueue = workqueue.NewTypedRateLimitingQueueWithConfig[any](
		limiter,
		workqueue.TypedRateLimitingQueueConfig[any]{Clock: clock})
	t.Cleanup(worker.Shutdown)
	// A burst of ordinary informer updates precedes the first actual failure.
	events.handler.OnAdd(workflow, true)
	for update := 0; update < 10; update++ {
		events.handler.OnUpdate(workflow, workflow)
	}
	clock.Step(DefaultJobBackOff)
	require.Eventually(t, func() bool { return worker.Len() == 1 }, time.Second, time.Millisecond)
	require.True(t, worker.processNextWorkItem())
	assert.Nil(t, pipelineClient.GetWorkflow("team", "run"))
	assert.Equal(t, 1, worker.workqueue.NumRequeues("team/run"))
	require.Equal(t, DefaultJobBackOff, limiter.delays[len(limiter.delays)-1], "normal updates must not turn the first transient failure into a maximum-backoff retry")
	assert.Zero(t, worker.Len(), "transient errors must not retry immediately")
	pipelineClient.SetError(nil)
	clock.Step(DefaultJobBackOff)
	require.Eventually(t, func() bool { return worker.Len() == 1 }, time.Second, time.Millisecond)
	require.True(t, worker.processNextWorkItem())
	assert.Equal(t, workflow, pipelineClient.GetWorkflow("team", "run"))
	assert.Zero(t, worker.workqueue.NumRequeues("team/run"))
	assert.Zero(t, worker.Len())
}
