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
	scheduleclientset "github.com/kubeflow/pipelines/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/pkg/client/informers/externalversions/schedule/v1alpha1"
	"github.com/kubeflow/pipelines/resources/schedule/util"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/cache"
)

type ScheduleClient struct {
	clientSet scheduleclientset.Interface
	informer  v1alpha1.ScheduleInformer
}

func NewScheduleClient(clientSet scheduleclientset.Interface, informer v1alpha1.ScheduleInformer) *ScheduleClient {
	return &ScheduleClient{
		clientSet: clientSet,
		informer:  informer,
	}
}

func (p *ScheduleClient) AddEventHandler(funcs *cache.ResourceEventHandlerFuncs) {
	p.informer.Informer().AddEventHandler(funcs)
}

func (p *ScheduleClient) HasSynced() func() bool {
	return p.informer.Informer().HasSynced
}

func (p *ScheduleClient) Get(namespace string, name string) (*util.ScheduleWrap, error) {
	schedule, err := p.informer.Lister().Schedules(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return util.NewScheduleWrap(schedule), nil
}

func (p *ScheduleClient) Update(namespace string, schedule *util.ScheduleWrap) error {
	_, err := p.clientSet.ScheduleV1alpha1().Schedules(namespace).Update(schedule.Schedule())
	return err
}
