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
	"github.com/googleprivate/ml/backend/src/crd/controller/scheduledworkflow/util"
	swfclientset "github.com/googleprivate/ml/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/googleprivate/ml/backend/src/crd/pkg/client/informers/externalversions/scheduledworkflow/v1alpha1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

// ScheduledWorkflowClient is a client to call the ScheduledWorkflow API.
type ScheduledWorkflowClient struct {
	clientSet swfclientset.Interface
	informer  v1alpha1.ScheduledWorkflowInformer
}

// NewScheduledWorkflowClient creates an instance of the client.
func NewScheduledWorkflowClient(clientSet swfclientset.Interface,
	informer v1alpha1.ScheduledWorkflowInformer) *ScheduledWorkflowClient {
	return &ScheduledWorkflowClient{
		clientSet: clientSet,
		informer:  informer,
	}
}

// AddEventHandler adds an event handler.
func (p *ScheduledWorkflowClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	p.informer.Informer().AddEventHandler(funcs)
}

// HasSynced returns true if the shared informer's store has synced.
func (p *ScheduledWorkflowClient) HasSynced() func() bool {
	return p.informer.Informer().HasSynced
}

// Get returns a ScheduledWorkflow, given a namespace and a name.
func (p *ScheduledWorkflowClient) Get(namespace string, name string) (*util.ScheduledWorkflow, error) {
	schedule, err := p.informer.Lister().ScheduledWorkflows(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return util.NewScheduledWorkflow(schedule), nil
}

// Update Updates a ScheduledWorkflow in the Kubernetes API server.
func (p *ScheduledWorkflowClient) Update(namespace string,
	schedule *util.ScheduledWorkflow) error {
	_, err := p.clientSet.ScheduledworkflowV1alpha1().ScheduledWorkflows(namespace).
		Update(schedule.Get())
	return err
}
