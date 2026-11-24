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

package util

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	workflowregister "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	argoinformer "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions"
	swfregister "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swffake "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/fake"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

// An object of one namespace that does not decode into the Go type must not
// keep the objects of other namespaces out of the informer cache.
func TestTolerantInformer_SkipsUndecodableObjects(t *testing.T) {
	tests := []struct {
		name     string
		resource schema.GroupVersionResource
		kind     string
		// A spec the CRD accepts and the Go type does not.
		undecodableSpec map[string]interface{}
		expectedType    interface{}
		newInformer     func(client dynamic.Interface) (cache.SharedIndexInformer, func(stopCh <-chan struct{}))
	}{
		{
			name:            "Workflow",
			resource:        workflowapi.SchemeGroupVersion.WithResource(workflowregister.WorkflowPlural),
			kind:            workflowregister.WorkflowKind,
			undecodableSpec: map[string]interface{}{"activeDeadlineSeconds": "not-a-number"},
			expectedType:    &workflowapi.Workflow{},
			newInformer: func(client dynamic.Interface) (cache.SharedIndexInformer, func(stopCh <-chan struct{})) {
				factory := argoinformer.NewSharedInformerFactory(&argofake.Clientset{}, 0)
				registerWorkflowInformer(factory, client, "")
				return factory.Argoproj().V1alpha1().Workflows().Informer(), factory.Start
			},
		},
		{
			name:            "ScheduledWorkflow",
			resource:        swfapi.SchemeGroupVersion.WithResource(swfregister.Plural),
			kind:            swfregister.Kind,
			undecodableSpec: map[string]interface{}{"maxConcurrency": "not-a-number"},
			expectedType:    &swfapi.ScheduledWorkflow{},
			newInformer: func(client dynamic.Interface) (cache.SharedIndexInformer, func(stopCh <-chan struct{})) {
				factory := swfinformers.NewSharedInformerFactory(swffake.NewSimpleClientset(), 0)
				RegisterScheduledWorkflowInformer(factory, client, "")
				return factory.Scheduledworkflow().V1beta1().ScheduledWorkflows().Informer(), factory.Start
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newObject := func(namespace, name string, spec map[string]interface{}) *unstructured.Unstructured {
				object := &unstructured.Unstructured{Object: map[string]interface{}{"spec": spec}}
				object.SetAPIVersion(test.resource.GroupVersion().String())
				object.SetKind(test.kind)
				object.SetNamespace(namespace)
				object.SetName(name)
				return object
			}
			client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
				map[schema.GroupVersionResource]string{test.resource: test.kind + "List"},
				newObject("tenant-a", "undecodable", test.undecodableSpec),
				newObject("tenant-b", "ordinary", map[string]interface{}{}))
			// The fake client does not replay what happened before a watch
			// started, so the test waits for the informer's watch.
			watching := make(chan struct{})
			var watchingOnce sync.Once
			client.PrependWatchReactor("*", func(action k8stesting.Action) (bool, watch.Interface, error) {
				watcher, err := client.Tracker().Watch(action.GetResource(), action.GetNamespace())
				watchingOnce.Do(func() { close(watching) })
				return true, watcher, err
			})

			informer, start := test.newInformer(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			start(ctx.Done())
			require.True(t, cache.WaitForCacheSync(ctx.Done(), informer.HasSynced))
			assert.Equal(t, []string{"tenant-b/ordinary"}, informer.GetStore().ListKeys())
			cached, exists, err := informer.GetStore().GetByKey("tenant-b/ordinary")
			require.NoError(t, err)
			require.True(t, exists)
			assert.IsType(t, test.expectedType, cached)

			select {
			case <-watching:
			case <-ctx.Done():
				require.FailNow(t, "The informer did not start watching")
			}
			resourceClient := func(namespace string) dynamic.ResourceInterface {
				return client.Resource(test.resource).Namespace(namespace)
			}
			isCached := func(key string) func() bool {
				return func() bool {
					_, exists, err := informer.GetStore().GetByKey(key)
					return err == nil && exists
				}
			}

			// An undecodable object added later does not end the watch.
			_, err = resourceClient("tenant-a").Create(ctx,
				newObject("tenant-a", "undecodable-later", test.undecodableSpec), v1.CreateOptions{})
			require.NoError(t, err)
			_, err = resourceClient("tenant-b").Create(ctx,
				newObject("tenant-b", "ordinary-later", map[string]interface{}{}), v1.CreateOptions{})
			require.NoError(t, err)
			require.Eventually(t, isCached("tenant-b/ordinary-later"), 10*time.Second, 10*time.Millisecond)

			// An object that becomes undecodable leaves the cache.
			_, err = resourceClient("tenant-b").Update(ctx,
				newObject("tenant-b", "ordinary", test.undecodableSpec), v1.UpdateOptions{})
			require.NoError(t, err)
			require.Eventually(t, func() bool { return !isCached("tenant-b/ordinary")() },
				10*time.Second, 10*time.Millisecond)
			assert.ElementsMatch(t, []string{"tenant-b/ordinary-later"}, informer.GetStore().ListKeys())
		})
	}
}

// The informer must come back from failures that an API server can produce at
// any time: a failed list, a watch that ends, and an object that stops and
// later starts decoding again.
func TestTolerantInformer_RecoversFromListAndWatchFailures(t *testing.T) {
	resource := workflowapi.SchemeGroupVersion.WithResource(workflowregister.WorkflowPlural)
	newWorkflow := func(namespace, name string, spec map[string]interface{}) *unstructured.Unstructured {
		object := &unstructured.Unstructured{Object: map[string]interface{}{"spec": spec}}
		object.SetAPIVersion(resource.GroupVersion().String())
		object.SetKind(workflowregister.WorkflowKind)
		object.SetNamespace(namespace)
		object.SetName(name)
		return object
	}
	undecodable := map[string]interface{}{"activeDeadlineSeconds": "not-a-number"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{resource: workflowregister.WorkflowKind + "List"},
		newWorkflow("tenant-b", "ordinary", map[string]interface{}{}))

	// The first lists fail.
	var failingLists atomic.Int32
	failingLists.Store(2)
	client.PrependReactor("list", "*", func(k8stesting.Action) (bool, runtime.Object, error) {
		if failingLists.Add(-1) >= 0 {
			return true, nil, errors.New("the API server is unavailable")
		}
		return false, nil, nil
	})
	// The fake client does not replay what happened before a watch started, so
	// the test follows each watch the informer opens.
	watchers := make(chan watch.Interface, 10)
	client.PrependWatchReactor("*", func(action k8stesting.Action) (bool, watch.Interface, error) {
		watcher, err := client.Tracker().Watch(action.GetResource(), action.GetNamespace())
		if err == nil {
			watchers <- watcher
		}
		return true, watcher, err
	})

	factory := argoinformer.NewSharedInformerFactory(&argofake.Clientset{}, 0)
	registerWorkflowInformer(factory, client, "")
	informer := factory.Argoproj().V1alpha1().Workflows().Informer()
	deleted := make(chan interface{}, 10)
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(object interface{}) { deleted <- object },
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	factory.Start(ctx.Done())
	nextWatch := func() watch.Interface {
		select {
		case watcher := <-watchers:
			return watcher
		case <-ctx.Done():
			require.FailNow(t, "The informer did not start watching")
			return nil
		}
	}
	isCached := func(key string) func() bool {
		return func() bool {
			_, exists, err := informer.GetStore().GetByKey(key)
			return err == nil && exists
		}
	}
	workflows := func(namespace string) dynamic.ResourceInterface {
		return client.Resource(resource).Namespace(namespace)
	}

	// Failed lists are retried until one succeeds.
	require.True(t, cache.WaitForCacheSync(ctx.Done(), informer.HasSynced))
	assert.Negative(t, failingLists.Load())
	assert.Equal(t, []string{"tenant-b/ordinary"}, informer.GetStore().ListKeys())

	// A watch that ends is replaced, and the next one is tolerant as well.
	nextWatch().Stop()
	nextWatch()
	_, err = workflows("tenant-a").Create(ctx, newWorkflow("tenant-a", "undecodable", undecodable), v1.CreateOptions{})
	require.NoError(t, err)
	_, err = workflows("tenant-b").Create(ctx, newWorkflow("tenant-b", "after-reconnect", map[string]interface{}{}), v1.CreateOptions{})
	require.NoError(t, err)
	require.Eventually(t, isCached("tenant-b/after-reconnect"), 20*time.Second, 10*time.Millisecond)
	assert.False(t, isCached("tenant-a/undecodable")())

	// An object that stops decoding is reported to handlers as a typed delete.
	// Every undecodable event is reported that way, so handlers also get a
	// delete for tenant-a/undecodable, which was never in the cache. They key
	// their work by namespace and name, so such a delete finds nothing to do.
	_, err = workflows("tenant-b").Update(ctx, newWorkflow("tenant-b", "ordinary", undecodable), v1.UpdateOptions{})
	require.NoError(t, err)
	deletedKeys := map[string]bool{}
	for !deletedKeys["tenant-b/ordinary"] {
		select {
		case object := <-deleted:
			workflow, ok := object.(*workflowapi.Workflow)
			require.True(t, ok, "handlers receive typed Workflows, got %T", object)
			deletedKeys[workflow.Namespace+"/"+workflow.Name] = true
		case <-ctx.Done():
			require.FailNow(t, "No delete was reported for the object that stopped decoding")
		}
	}
	assert.True(t, deletedKeys["tenant-a/undecodable"])
	assert.False(t, isCached("tenant-b/ordinary")())

	// Once it decodes again it is back in the cache.
	_, err = workflows("tenant-b").Update(ctx, newWorkflow("tenant-b", "ordinary", map[string]interface{}{}), v1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, isCached("tenant-b/ordinary"), 20*time.Second, 10*time.Millisecond)
}
