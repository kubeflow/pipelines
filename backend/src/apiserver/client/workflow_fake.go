// Copyright 2018 The Kubeflow Authors
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
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
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

type jsonPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

func NewWorkflowClientFake() *FakeWorkflowClient {
	return &FakeWorkflowClient{
		workflows:       make(map[string]*v1alpha1.Workflow),
		lastGeneratedId: -1,
	}
}

func (c *FakeWorkflowClient) Create(ctx context.Context, execSpec util.ExecutionSpec, opts v1.CreateOptions) (util.ExecutionSpec, error) {
	workflow, ok := execSpec.(*util.Workflow)
	if !ok {
		return nil, fmt.Errorf("not a valid ExecutionSpec for Workflow")
	}
	if workflow.GenerateName != "" {
		c.lastGeneratedId += 1
		workflow.Name = workflow.GenerateName + strconv.Itoa(c.lastGeneratedId)
		workflow.GenerateName = ""
	}
	c.workflows[workflow.Name] = workflow.Workflow
	return workflow, nil
}

func (c *FakeWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (util.ExecutionSpec, error) {
	workflow, ok := c.workflows[name]
	if ok {
		return util.NewWorkflow(workflow), nil
	}
	return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
}

func (c *FakeWorkflowClient) List(ctx context.Context, opts v1.ListOptions) (*util.ExecutionSpecList, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}

func (c *FakeWorkflowClient) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}

func (c *FakeWorkflowClient) Update(ctx context.Context, execSpec util.ExecutionSpec, opts v1.UpdateOptions) (util.ExecutionSpec, error) {
	workflow, ok := execSpec.(*util.Workflow)
	if !ok {
		return nil, fmt.Errorf("not a valid ExecutionSpec for Workflow")
	}
	name := workflow.GetObjectMeta().GetName()
	_, ok = c.workflows[name]
	if ok {
		c.workflows[name] = workflow.Workflow
		return workflow, nil
	}
	return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
}

func (c *FakeWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	_, ok := c.workflows[name]
	if ok {
		return nil
	}
	return k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
}

func (c *FakeWorkflowClient) DeleteCollection(ctx context.Context, options v1.DeleteOptions,
	listOptions v1.ListOptions,
) error {
	glog.Error("This fake method is not yet implemented")
	return nil
}

func (c *FakeWorkflowClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions,
	subresources ...string,
) (util.ExecutionSpec, error) {
	workflow, ok := c.workflows[name]
	if !ok {
		return nil, k8errors.NewNotFound(k8schema.ParseGroupResource("workflows.argoproj.io"), name)
	}

	if pt == types.JSONPatchType {
		return applyJSONPatchToFakeWorkflow(workflow, name, data)
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
				return util.NewWorkflow(workflow), nil
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
			return util.NewWorkflow(workflow), nil
		}
	}
	return nil, errors.New("Failed to patch workflow")
}

func applyJSONPatchToFakeWorkflow(workflow *v1alpha1.Workflow, name string, data []byte) (util.ExecutionSpec, error) {
	var patchOperations []jsonPatchOperation
	if err := json.Unmarshal(data, &patchOperations); err != nil {
		return nil, err
	}

	for _, patchOperation := range patchOperations {
		switch patchOperation.Op {
		case "test":
			actualValue, ok := fakeWorkflowJSONPatchValue(workflow, patchOperation.Path)
			if !ok || actualValue != fmt.Sprint(patchOperation.Value) {
				return nil, k8errors.NewConflict(k8schema.ParseGroupResource("workflows.argoproj.io"), name, fmt.Errorf("json patch test failed for %s", patchOperation.Path))
			}
		case "add":
			if !strings.HasPrefix(patchOperation.Path, "/metadata/labels/") {
				return nil, fmt.Errorf("unsupported fake JSON patch add path %q", patchOperation.Path)
			}
			labelKey := unescapeJSONPointerPathPart(strings.TrimPrefix(patchOperation.Path, "/metadata/labels/"))
			if workflow.Labels == nil {
				workflow.Labels = map[string]string{}
			}
			workflow.Labels[labelKey] = fmt.Sprint(patchOperation.Value)
		default:
			return nil, fmt.Errorf("unsupported fake JSON patch operation %q", patchOperation.Op)
		}
	}

	return util.NewWorkflow(workflow), nil
}

func fakeWorkflowJSONPatchValue(workflow *v1alpha1.Workflow, path string) (string, bool) {
	switch path {
	case "/metadata/resourceVersion":
		return workflow.ResourceVersion, true
	case "/status/phase":
		return string(workflow.Status.Phase), true
	default:
		return "", false
	}
}

func unescapeJSONPointerPathPart(pathPart string) string {
	return strings.ReplaceAll(strings.ReplaceAll(pathPart, "~1", "/"), "~0", "~")
}

type FakeBadWorkflowClient struct {
	FakeWorkflowClient
}

func (FakeBadWorkflowClient) Create(context.Context, util.ExecutionSpec, v1.CreateOptions) (util.ExecutionSpec, error) {
	return nil, errors.New("some error")
}

func (FakeBadWorkflowClient) Get(ctx context.Context, name string, options v1.GetOptions) (util.ExecutionSpec, error) {
	return nil, errors.New("some error")
}

func (c *FakeBadWorkflowClient) Update(ctx context.Context, workflow util.ExecutionSpec, opts v1.UpdateOptions) (util.ExecutionSpec, error) {
	return nil, errors.New("failed to update workflow")
}

func (c *FakeBadWorkflowClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	return errors.New("failed to delete workflow")
}
