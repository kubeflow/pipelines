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

package storage

import (
	v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
)

type FakeWorkflowClient struct {
	workflows map[string]*v1alpha1.Workflow
}

func NewWorkflowClientFake() *FakeWorkflowClient {
	return &FakeWorkflowClient{
		workflows: make(map[string]*v1alpha1.Workflow),
	}
}

func (c *FakeWorkflowClient) Get(name string, options v1.GetOptions) (result *v1alpha1.Workflow, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) List(opts v1.ListOptions) (result *v1alpha1.WorkflowList, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) Create(workflow *v1alpha1.Workflow) (result *v1alpha1.Workflow, err error) {
	c.workflows[workflow.Name] = workflow
	return workflow, nil
}

func (c *FakeWorkflowClient) Update(workflow *v1alpha1.Workflow) (result *v1alpha1.Workflow, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) Delete(name string, options *v1.DeleteOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (c *FakeWorkflowClient) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (c *FakeWorkflowClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Workflow, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) GetWorkflowCount() int {
	return len(c.workflows)
}
