// Copyright 2019 Google LLC
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
	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
)

type FakeArgoClient struct {
	workflowClientFake FakeWorkflowClientInterface
}

func NewFakeArgoClient() *FakeArgoClient {
	return &FakeArgoClient{NewWorkflowClientFake()}
}

func (c *FakeArgoClient) Workflow(namespace string) argoprojv1alpha1.WorkflowInterface {
	return c.workflowClientFake
}

func (c *FakeArgoClient) FakeWorkflow() FakeWorkflowClientInterface {
	return c.workflowClientFake
}

func (c *FakeArgoClient) SetFakeWorkflow(fakeWorkflow FakeWorkflowClientInterface) {
	c.workflowClientFake = fakeWorkflow
}
