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
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
)

type FakeExecClient struct {
	workflowClientFake *FakeWorkflowClient
}

func NewFakeExecClient() *FakeExecClient {
	return &FakeExecClient{NewWorkflowClientFake()}
}

func (c *FakeExecClient) Execution(namespace string) util.ExecutionInterface {
	if len(namespace) == 0 {
		panic(util.NewResourceNotFoundError("Namespace", namespace))
	}
	return c.workflowClientFake
}

func (c *FakeExecClient) Compare(old, new interface{}) bool {
	return false
}

func (c *FakeExecClient) GetWorkflowCount() int {
	return len(c.workflowClientFake.workflows)
}

func (c *FakeExecClient) GetWorkflowKeys() map[string]bool {
	result := map[string]bool{}
	for key := range c.workflowClientFake.workflows {
		result[key] = true
	}
	return result
}

func (c *FakeExecClient) IsTerminated(name string) (bool, error) {
	workflow, ok := c.workflowClientFake.workflows[name]
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
