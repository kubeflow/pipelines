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

package resource

import (
	"errors"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type FakeScheduledWorkflowClient struct {
	workflows map[string]*v1beta1.ScheduledWorkflow
}

func NewScheduledWorkflowClientFake() *FakeScheduledWorkflowClient {
	return &FakeScheduledWorkflowClient{
		workflows: make(map[string]*v1beta1.ScheduledWorkflow),
	}
}

func (c *FakeScheduledWorkflowClient) Create(workflow *v1beta1.ScheduledWorkflow) (*v1beta1.ScheduledWorkflow, error) {
	workflow.UID = "123"
	workflow.Namespace = "default"
	workflow.Name = workflow.GenerateName
	c.workflows[workflow.Name] = workflow
	return workflow, nil
}

func (c *FakeScheduledWorkflowClient) Delete(name string, options *v1.DeleteOptions) error {
	delete(c.workflows, name)
	return nil
}

func (c *FakeScheduledWorkflowClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.ScheduledWorkflow, err error) {
	return nil, nil
}

func (c *FakeScheduledWorkflowClient) Get(name string, options v1.GetOptions) (*v1beta1.ScheduledWorkflow, error) {
	workflow, ok := c.workflows[name]
	if ok {
		return workflow, nil
	}
	return nil, errors.New("not found")
}

func (c *FakeScheduledWorkflowClient) Update(*v1beta1.ScheduledWorkflow) (*v1beta1.ScheduledWorkflow, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeScheduledWorkflowClient) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (c *FakeScheduledWorkflowClient) List(opts v1.ListOptions) (*v1beta1.ScheduledWorkflowList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeScheduledWorkflowClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

type FakeBadScheduledWorkflowClient struct {
	FakeScheduledWorkflowClient
}

func (FakeBadScheduledWorkflowClient) Create(workflow *v1beta1.ScheduledWorkflow) (*v1beta1.ScheduledWorkflow, error) {
	return nil, errors.New("some error")
}

func (FakeBadScheduledWorkflowClient) Get(name string, options v1.GetOptions) (*v1beta1.ScheduledWorkflow, error) {
	return nil, errors.New("some error")
}

func (FakeBadScheduledWorkflowClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.ScheduledWorkflow, err error) {
	return nil, errors.New("some error")
}
