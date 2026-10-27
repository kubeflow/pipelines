// Copyright 2026 The Kubeflow Authors
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

package main

import (
	"fmt"
	"testing"
	"time"

	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swffake "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/fake"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	fcache "k8s.io/client-go/tools/cache/testing"
)

const takeoverNamespace = "kubeflow"

// noopExecutionInformer stands in for the Workflow informer, which these
// tests do not use.
type noopExecutionInformer struct{}

func (noopExecutionInformer) AddEventHandler(cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	return nil, nil
}
func (noopExecutionInformer) HasSynced() func() bool { return func() bool { return true } }
func (noopExecutionInformer) Get(string, string) (commonutil.ExecutionSpec, bool, error) {
	return nil, false, nil
}
func (noopExecutionInformer) List(string, *labels.Selector) (commonutil.ExecutionSpecList, error) {
	return nil, nil
}
func (noopExecutionInformer) InformerFactoryStart(<-chan struct{}) {}

// scheduleSource is a ScheduledWorkflow informer fed from a fake source. The
// generated fake clientset cannot list ScheduledWorkflows, so the factory gets
// a custom informer for that type.
type scheduleSource struct {
	source          *fcache.FakeControllerSource
	informerFactory swfinformers.SharedInformerFactory
	stopChannel     chan struct{}
}

func newScheduleSource(t *testing.T) *scheduleSource {
	t.Helper()
	source := fcache.NewFakeControllerSource()
	informerFactory := swfinformers.NewSharedInformerFactory(swffake.NewSimpleClientset(), 0)
	informerFactory.InformerFor(&swfapi.ScheduledWorkflow{}, func(swfclientset.Interface, time.Duration) cache.SharedIndexInformer {
		return cache.NewSharedIndexInformer(source, &swfapi.ScheduledWorkflow{}, 0,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	})
	stopChannel := make(chan struct{})
	t.Cleanup(func() { close(stopChannel) })
	return &scheduleSource{source: source, informerFactory: informerFactory, stopChannel: stopChannel}
}

func (scheduleSource *scheduleSource) start() {
	scheduleSource.informerFactory.Start(scheduleSource.stopChannel)
	scheduleSource.informerFactory.WaitForCacheSync(scheduleSource.stopChannel)
}

// createAndDelete creates the live schedules and creates then deletes the
// churned ones, and waits until the informer cache shows only the live ones.
func (scheduleSource *scheduleSource) createAndDelete(t *testing.T, liveCount, churnCount int) {
	t.Helper()
	for index := 0; index < churnCount; index++ {
		schedule := &swfapi.ScheduledWorkflow{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("deleted-%d", index), Namespace: takeoverNamespace}}
		scheduleSource.source.Add(schedule)
		// Delete updates the object it is given, which the informer may still be reading.
		scheduleSource.source.Delete(schedule.DeepCopy())
	}
	for index := 0; index < liveCount; index++ {
		scheduleSource.source.Add(&swfapi.ScheduledWorkflow{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("live-%d", index), Namespace: takeoverNamespace}})
	}
	lister := scheduleSource.informerFactory.Scheduledworkflow().V1beta1().ScheduledWorkflows().Lister()
	require.Eventually(t, func() bool {
		cached, listError := lister.List(labels.Everything())
		return listError == nil && len(cached) == liveCount
	}, 5*time.Second, 10*time.Millisecond)
}

func (scheduleSource *scheduleSource) newController(t *testing.T) *Controller {
	t.Helper()
	controller, controllerError := NewController(k8sfake.NewClientset(), swffake.NewSimpleClientset(), nil, nil,
		scheduleSource.informerFactory, noopExecutionInformer{}, commonutil.NewRealTime(), time.UTC, nil, "", "", false)
	require.NoError(t, controllerError)
	t.Cleanup(controller.workqueue.ShutDown)
	return controller
}

// queuedKeys returns how many of the given keys were added to the work queue.
// Keys are added with AddRateLimited, which delays them by DefaultJobBackOff,
// so this checks the rate limiter instead of waiting for them to become ready.
func queuedKeys(controller *Controller, prefix string, count int) int {
	queued := 0
	for index := 0; index < count; index++ {
		if controller.workqueue.NumRequeues(fmt.Sprintf("%s/%s-%d", takeoverNamespace, prefix, index)) > 0 {
			queued++
		}
	}
	return queued
}

// A standby creates its controller only when it becomes leader, so deletion
// churn before the takeover leaves nothing in the new leader's work queue.
func TestTakeoverQueuesOnlyCurrentScheduledWorkflows(t *testing.T) {
	const liveCount, churnCount = 2, 100
	scheduleSource := newScheduleSource(t)
	scheduleSource.start()
	scheduleSource.createAndDelete(t, liveCount, churnCount)

	controller := scheduleSource.newController(t)

	assert.Eventually(t, func() bool { return queuedKeys(controller, "live", liveCount) == liveCount },
		5*time.Second, 10*time.Millisecond, "the new leader should queue every current schedule")
	assert.Equal(t, 0, queuedKeys(controller, "deleted", churnCount), "the new leader must not queue deleted schedules")
}
