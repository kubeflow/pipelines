// Copyright 2018 Google LLC
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

	workflowapi "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	client "github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

type FakeEventHandler struct {
	handler cache.ResourceEventHandler
}

func NewFakeEventHandler() *FakeEventHandler {
	return &FakeEventHandler{}
}

func (h *FakeEventHandler) AddEventHandler(handler cache.ResourceEventHandler) {
	h.handler = handler
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

	// Set up peristence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)

	// Test
	eventHandler.handler.OnAdd(workflow)
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

	// Set up peristence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)

	// Test
	eventHandler.handler.OnAdd(workflow)
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

	// Set up peristence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)

	// Test
	eventHandler.handler.OnAdd(workflow)
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

	// Set up peristence worker
	saver := NewWorkflowSaver(workflowClient, pipelineClient, 100)
	eventHandler := NewFakeEventHandler()
	worker := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)

	// Test
	eventHandler.handler.OnAdd(workflow)
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
	worker := NewPersistenceWorker(
		util.NewFakeTimeForEpoch(),
		"PERSISTENCE_WORKER",
		eventHandler,
		false,
		saver)

	// Test
	eventHandler.handler.OnAdd(workflow)
	worker.processNextWorkItem()
	assert.Nil(t, pipelineClient.GetWorkflow("MY_NAMESPACE", "MY_NAME"))
	assert.Equal(t, 0, worker.Len())
}
