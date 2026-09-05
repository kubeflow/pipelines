// Copyright 2019 The Kubeflow Authors
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

package client

import (
	"sync"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

type FakeExecClient struct {
	mu              sync.RWMutex
	workflowClients map[string]*FakeWorkflowClient
}

func NewFakeExecClient() *FakeExecClient {
	return &FakeExecClient{workflowClients: make(map[string]*FakeWorkflowClient)}
}

func (c *FakeExecClient) Execution(namespace string) util.ExecutionInterface {
	if len(namespace) == 0 {
		panic(util.NewResourceNotFoundError("Namespace", namespace))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	workflowClient, ok := c.workflowClients[namespace]
	if !ok {
		workflowClient = NewWorkflowClientFake()
		c.workflowClients[namespace] = workflowClient
	}
	return workflowClient
}

func (c *FakeExecClient) Compare(old, new interface{}) bool {
	return false
}

func (c *FakeExecClient) GetWorkflowCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	count := 0
	for _, workflowClient := range c.workflowClients {
		workflowClient.mu.RLock()
		count += len(workflowClient.workflows)
		workflowClient.mu.RUnlock()
	}
	return count
}

func (c *FakeExecClient) GetWorkflowKeysInNamespace(namespace string) map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := map[string]bool{}
	workflowClient, ok := c.workflowClients[namespace]
	if !ok {
		return result
	}
	workflowClient.mu.RLock()
	defer workflowClient.mu.RUnlock()
	for key := range workflowClient.workflows {
		result[key] = true
	}
	return result
}

func (c *FakeExecClient) GetWorkflowDeleteCountInNamespace(namespace, name string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	workflowClient, ok := c.workflowClients[namespace]
	if !ok {
		return 0
	}
	workflowClient.mu.RLock()
	defer workflowClient.mu.RUnlock()
	return workflowClient.deleteCalls[name]
}

func (c *FakeExecClient) IsTerminatedInNamespace(namespace, name string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	workflowClient, ok := c.workflowClients[namespace]
	if !ok {
		return false, errors.New("No workflow found with name: " + name)
	}
	workflowClient.mu.RLock()
	defer workflowClient.mu.RUnlock()
	workflow, ok := workflowClient.workflows[name]
	if !ok {
		return false, errors.New("No workflow found with name: " + name)
	}

	activeDeadlineSeconds := workflow.Spec.ActiveDeadlineSeconds
	if activeDeadlineSeconds == nil {
		return false, errors.New("No ActiveDeadlineSeconds found in workflow with name: " + name)
	}

	return *activeDeadlineSeconds == 0, nil
}

type FakeExecClientWithBadWorkflow struct {
	workflowClientFake *FakeBadWorkflowClient
}

func NewFakeExecClientWithBadWorkflow() *FakeExecClientWithBadWorkflow {
	return &FakeExecClientWithBadWorkflow{&FakeBadWorkflowClient{}}
}

func (c *FakeExecClientWithBadWorkflow) Execution(namespace string) util.ExecutionInterface {
	return c.workflowClientFake
}

func (c *FakeExecClientWithBadWorkflow) Compare(old, new interface{}) bool {
	return false
}
