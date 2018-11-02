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
	"github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

type WorkflowClientInterface interface {
	Get(namespace string, name string) (wf *util.Workflow, err error)
}

// WorkflowClient is a client to call the Workflow API.
type WorkflowClient struct {
	informer v1alpha1.WorkflowInformer
}

// NewWorkflowClient creates an instance of the WorkflowClient.
func NewWorkflowClient(informer v1alpha1.WorkflowInformer) *WorkflowClient {
	return &WorkflowClient{
		informer: informer,
	}
}

// AddEventHandler adds an event handler.
func (c *WorkflowClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	c.informer.Informer().AddEventHandler(funcs)
}

// HasSynced returns true if the shared informer's store has synced.
func (c *WorkflowClient) HasSynced() func() bool {
	return c.informer.Informer().HasSynced
}

// Get returns a Workflow, given a namespace and name.
func (c *WorkflowClient) Get(namespace string, name string) (
	wf *util.Workflow, err error) {
	workflow, err := c.informer.Lister().Workflows(namespace).Get(name)
	if err != nil {
		var code util.CustomCode
		if util.IsNotFound(err) {
			code = util.CUSTOM_CODE_NOT_FOUND
		} else {
			code = util.CUSTOM_CODE_GENERIC
		}
		return nil, util.NewCustomError(err, code,
			"Error retrieving workflow (%v) in namespace (%v): %v", name, namespace, err)
	}
	return util.NewWorkflow(workflow), nil
}
