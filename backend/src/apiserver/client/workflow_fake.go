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

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8schema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

type FakeWorkflowClient struct {
	workflows       map[string]*v1alpha1.Workflow
	lastGeneratedId int
	configMapClient corev1client.ConfigMapInterface
}

const pipelineParallelismConfigMapName = "kfp-argo-workflow-semaphores"

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

	// Handle ConfigMap creation for pipeline parallelism if annotations are present
	if c.configMapClient != nil && workflow.Workflow != nil && workflow.Annotations != nil {
		pipelineVersionID := workflow.Annotations[util.AnnotationKeyPipelineVersionID]
		maxActiveRunsStr := workflow.Annotations[util.AnnotationKeyMaxActiveRuns]

		if pipelineVersionID == "" || maxActiveRunsStr == "" {
			// Skip ConfigMap creation if annotations are missing (matches real implementation's early return)
		}
		if pipelineVersionID != "" && maxActiveRunsStr != "" {
			maxActiveRuns, err := strconv.Atoi(maxActiveRunsStr)
			if err != nil {
				glog.Warningf("Invalid max_active_runs annotation value: %v", err)
			}
			if maxActiveRuns <= 0 {
				glog.Warningf("max_active_runs must be greater than 0, got %d", maxActiveRuns)
			}
			namespace := workflow.Namespace
			if namespace == "" {
				glog.Warningf("Workflow namespace is required for ConfigMap operations")
				// Don't fail the workflow creation, just log the warning
			}
			if err == nil && maxActiveRuns > 0 && namespace != "" {
				value := strconv.Itoa(maxActiveRuns)
				configMapApply := applyv1.ConfigMap(pipelineParallelismConfigMapName, namespace).
					WithData(map[string]string{pipelineVersionID: value})
				_, err := c.configMapClient.Apply(ctx, configMapApply, v1.ApplyOptions{FieldManager: "kubeflow-pipelines", Force: false})
				if err != nil {
					glog.Warningf("Failed to persist max_active_runs configuration for pipeline version %s in namespace %s: %v", pipelineVersionID, namespace, err)
				}
			}
		}
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
