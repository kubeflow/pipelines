// Copyright 2018 Google LLC
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
	"encoding/json"
	"strconv"

	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type FakeWorkflowClient struct {
	workflows       map[string]*v1alpha1.Workflow
	lastGeneratedId int
}

func NewWorkflowClientFake() *FakeWorkflowClient {
	return &FakeWorkflowClient{
		workflows:       make(map[string]*v1alpha1.Workflow),
		lastGeneratedId: -1,
	}
}

func (c *FakeWorkflowClient) Create(workflow *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	if workflow.GenerateName != "" {
		c.lastGeneratedId += 1
		workflow.Name = workflow.GenerateName + strconv.Itoa(c.lastGeneratedId)
		workflow.GenerateName = ""
	}
	c.workflows[workflow.Name] = workflow
	return workflow, nil
}

func (c *FakeWorkflowClient) Get(name string, options v1.GetOptions) (*v1alpha1.Workflow, error) {
	workflow, ok := c.workflows[name]
	if ok {
		return workflow, nil
	}
	return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
}

func (c *FakeWorkflowClient) List(opts v1.ListOptions) (*v1alpha1.WorkflowList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (c *FakeWorkflowClient) Update(workflow *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	name := workflow.GetObjectMeta().GetName()
	_, ok := c.workflows[name]
	if ok {
		return workflow, nil
	}
	return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
}

func (c *FakeWorkflowClient) Delete(name string, options *v1.DeleteOptions) error {
	_, ok := c.workflows[name]
	if ok {
		return nil
	}
	return k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
}

func (c *FakeWorkflowClient) DeleteCollection(options *v1.DeleteOptions,
	listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (c *FakeWorkflowClient) Patch(name string, pt types.PatchType, data []byte,
	subresources ...string) (*v1alpha1.Workflow, error) {

	_, ok := c.workflows[name]
	if !ok {
		return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
	}

	var dat map[string]interface{}
	json.Unmarshal(data, &dat)

	// TODO: Should we actually assert the type here, or just panic if it's wrong?

	if _, ok := dat["spec"]; ok {
		spec := dat["spec"].(map[string]interface{})
		activeDeadlineSeconds := spec["activeDeadlineSeconds"].(float64)

		// Simulate terminating a workflow
		if pt == types.MergePatchType && activeDeadlineSeconds == 0 {
			workflow, ok := c.workflows[name]
			if ok {
				newActiveDeadlineSeconds := int64(0)
				workflow.Spec.ActiveDeadlineSeconds = &newActiveDeadlineSeconds
				return workflow, nil
			}
		}
	}

	if _, ok := dat["metadata"]; ok {
		workflow, ok := c.workflows[name]
		if ok {
			if workflow.Labels == nil {
				workflow.Labels = map[string]string{}
			}
			workflow.Labels[util.LabelKeyWorkflowPersistedFinalState] = "true"
			return workflow, nil
		}
	}
	return nil, errors.New("Failed to patch workflow")
}

type FakeBadWorkflowClient struct {
	FakeWorkflowClient
}

func (FakeBadWorkflowClient) Create(*v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return nil, errors.New("some error")
}

func (FakeBadWorkflowClient) Get(name string, options v1.GetOptions) (*v1alpha1.Workflow, error) {
	return nil, errors.New("some error")
}

func (c *FakeBadWorkflowClient) Update(workflow *v1alpha1.Workflow) (*v1alpha1.Workflow, error) {
	return nil, errors.New("failed to update workflow")
}

func (c *FakeBadWorkflowClient) Delete(name string, options *v1.DeleteOptions) error {
	return errors.New("failed to delete workflow")
}
