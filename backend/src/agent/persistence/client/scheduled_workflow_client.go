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
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

type ScheduledWorkflowClientInterface interface {
	Get(namespace string, name string) (swf *util.ScheduledWorkflow, err error)
}

// ScheduledWorkflowClient is a client to call the ScheduledWorkflow API.
type ScheduledWorkflowClient struct {
	informer v1beta1.ScheduledWorkflowInformer
}

// NewScheduledWorkflowClient creates an instance of the client.
func NewScheduledWorkflowClient(informer v1beta1.ScheduledWorkflowInformer) *ScheduledWorkflowClient {
	return &ScheduledWorkflowClient{
		informer: informer,
	}
}

// AddEventHandler adds an event handler.
func (c *ScheduledWorkflowClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	c.informer.Informer().AddEventHandler(funcs)
}

// HasSynced returns true if the shared informer's store has synced.
func (c *ScheduledWorkflowClient) HasSynced() func() bool {
	return c.informer.Informer().HasSynced
}

// Get returns a ScheduledWorkflow, given a namespace and a name.
func (c *ScheduledWorkflowClient) Get(namespace string, name string) (
	swf *util.ScheduledWorkflow, err error) {
	schedule, err := c.informer.Lister().ScheduledWorkflows(namespace).Get(name)
	if err != nil {
		var code util.CustomCode
		if util.IsNotFound(err) {
			code = util.CUSTOM_CODE_NOT_FOUND
		} else {
			code = util.CUSTOM_CODE_GENERIC
		}
		return nil, util.NewCustomError(err, code,
			"Error retrieving scheduled workflow (%v) in namespace (%v): %v", name, namespace, err)
	}

	return util.NewScheduledWorkflow(schedule), nil
}
