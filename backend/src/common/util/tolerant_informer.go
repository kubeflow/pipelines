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
	"time"

	workflowregister "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	argoinformer "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions"
	"github.com/golang/glog"
	swfregister "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
)

// The Workflow and ScheduledWorkflow CRDs do not validate the contents of spec
// and status, so the API server stores objects that do not decode into the Go
// types. The generated informers decode a whole list or watch stream at once:
// a single such object, in any namespace, fails every list and watch and
// leaves the informer unsynced or stale for all namespaces (see
// https://github.com/kubernetes/kubernetes/issues/57705). The informers
// registered below list and watch through the dynamic client instead and
// decode the objects one at a time, leaving out the ones that do not decode.

// errInformerAlreadyRequested reports a registration that came too late: the
// factory had already created its own informer, which decodes whole lists and
// would be the one in use.
var errInformerAlreadyRequested = errors.New("the informer was requested from the factory before the tolerant informer was registered")

// registerWorkflowInformer makes factory serve Workflows from an informer that
// tolerates undecodable objects. It must be called before the factory's
// Workflow informer is first requested, and returns an error otherwise.
func registerWorkflowInformer(factory argoinformer.SharedInformerFactory, client dynamic.Interface, namespace string) error {
	registered := false
	factory.InformerFor(&workflowapi.Workflow{},
		func(_ argoclient.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			registered = true
			return newTolerantInformer(client,
				workflowapi.SchemeGroupVersion.WithResource(workflowregister.WorkflowPlural), namespace, resyncPeriod,
				func() runtime.Object { return &workflowapi.Workflow{} },
				func() runtime.Object { return &workflowapi.WorkflowList{} })
		})
	if !registered {
		return errors.Wrap(errInformerAlreadyRequested, "Failed to register the Workflow informer")
	}
	return nil
}

// RegisterScheduledWorkflowInformer makes factory serve ScheduledWorkflows
// from an informer that tolerates undecodable objects. It must be called
// before the factory's ScheduledWorkflow informer is first requested, and
// returns an error otherwise.
func RegisterScheduledWorkflowInformer(factory swfinformers.SharedInformerFactory, client dynamic.Interface, namespace string) error {
	registered := false
	factory.InformerFor(&swfapi.ScheduledWorkflow{},
		func(_ swfclientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
			registered = true
			return newTolerantInformer(client,
				swfapi.SchemeGroupVersion.WithResource(swfregister.Plural), namespace, resyncPeriod,
				func() runtime.Object { return &swfapi.ScheduledWorkflow{} },
				func() runtime.Object { return &swfapi.ScheduledWorkflowList{} })
		})
	if !registered {
		return errors.Wrap(errInformerAlreadyRequested, "Failed to register the ScheduledWorkflow informer")
	}
	return nil
}

// newTolerantInformer returns an informer for the typed objects of a custom
// resource in the namespace (all namespaces if empty). Objects that do not
// decode into the type are logged and kept out of the cache.
func newTolerantInformer(client dynamic.Interface, resource schema.GroupVersionResource, namespace string,
	resyncPeriod time.Duration, newObject, newList func() runtime.Object,
) cache.SharedIndexInformer {
	resourceClient := client.Resource(resource).Namespace(namespace)
	decode := func(raw *unstructured.Unstructured) (runtime.Object, error) {
		data, err := raw.MarshalJSON()
		if err != nil {
			return nil, err
		}
		object := newObject()
		// The same decoding as the typed client applies to a response.
		if err := json.Unmarshal(data, object); err != nil {
			return nil, err
		}
		return object, nil
	}
	return cache.NewSharedIndexInformer(
		cache.ToListWatcherWithWatchListSemantics(&cache.ListWatch{
			ListWithContextFunc: func(ctx context.Context, options v1.ListOptions) (runtime.Object, error) {
				raw, err := resourceClient.List(ctx, options)
				if err != nil {
					return nil, err
				}
				objects := make([]runtime.Object, 0, len(raw.Items))
				for i := range raw.Items {
					object, err := decode(&raw.Items[i])
					if err != nil {
						logUndecodable(resource, &raw.Items[i], err)
						continue
					}
					objects = append(objects, object)
				}
				list := newList()
				if err := meta.SetList(list, objects); err != nil {
					return nil, err
				}
				listMeta, err := meta.ListAccessor(list)
				if err != nil {
					return nil, err
				}
				listMeta.SetResourceVersion(raw.GetResourceVersion())
				listMeta.SetContinue(raw.GetContinue())
				listMeta.SetRemainingItemCount(raw.GetRemainingItemCount())
				return list, nil
			},
			WatchFuncWithContext: func(ctx context.Context, options v1.ListOptions) (watch.Interface, error) {
				events, err := resourceClient.Watch(ctx, options)
				if err != nil {
					return nil, err
				}
				return watch.Filter(events, func(event watch.Event) (watch.Event, bool) {
					raw, ok := event.Object.(*unstructured.Unstructured)
					if !ok || event.Type == watch.Error {
						// Not an object of the resource: the reflector handles it.
						return event, true
					}
					object, err := decode(raw)
					if err != nil {
						logUndecodable(resource, raw, err)
						// Remove any version of the object cached before it
						// became undecodable. Its metadata, which the API
						// server does validate, identifies it.
						object, err = decode(&unstructured.Unstructured{Object: map[string]interface{}{
							"apiVersion": raw.GetAPIVersion(),
							"kind":       raw.GetKind(),
							"metadata":   raw.Object["metadata"],
						}})
						if err != nil {
							return event, false
						}
						event.Type = watch.Deleted
					}
					event.Object = object
					return event, true
				}), nil
			},
		}, client),
		newObject(),
		resyncPeriod,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)
}

func logUndecodable(resource schema.GroupVersionResource, raw *unstructured.Unstructured, err error) {
	glog.Warningf("Ignoring %s %s/%s, which cannot be decoded: %v",
		resource.GroupResource(), raw.GetNamespace(), raw.GetName(), err)
}
