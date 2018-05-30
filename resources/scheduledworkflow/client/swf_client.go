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

package client

import (
	swfclientset "github.com/kubeflow/pipelines/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/pkg/client/informers/externalversions/scheduledworkflow/v1alpha1"
	"github.com/kubeflow/pipelines/resources/scheduledworkflow/util"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

// ScheduledWorkflowClient is a client to call the ScheduledWorkflow API.
type ScheduledWorkflowClient struct {
	clientSet swfclientset.Interface
	informer  v1alpha1.ScheduledWorkflowInformer
}

func NewScheduledWorkflowClient(clientSet swfclientset.Interface,
		informer v1alpha1.ScheduledWorkflowInformer) *ScheduledWorkflowClient {
	return &ScheduledWorkflowClient{
		clientSet: clientSet,
		informer:  informer,
	}
}

func (p *ScheduledWorkflowClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	p.informer.Informer().AddEventHandler(funcs)
}

func (p *ScheduledWorkflowClient) HasSynced() func() bool {
	return p.informer.Informer().HasSynced
}

func (p *ScheduledWorkflowClient) Get(namespace string, name string) (*util.ScheduledWorkflowWrap, error) {
	schedule, err := p.informer.Lister().ScheduledWorkflows(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return util.NewScheduledWorkflowWrap(schedule), nil
}

func (p *ScheduledWorkflowClient) Update(namespace string,
		schedule *util.ScheduledWorkflowWrap) error {
	_, err := p.clientSet.ScheduledworkflowV1alpha1().ScheduledWorkflows(namespace).
		Update(schedule.ScheduledWorkflow())
	return err
}
