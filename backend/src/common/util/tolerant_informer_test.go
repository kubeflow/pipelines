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
	"sync"
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
