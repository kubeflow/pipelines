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
package fake

import (
	v1alpha1 "github.com/googleprivate/ml/pkg/apis/pipeline/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePipelines implements PipelineInterface
type FakePipelines struct {
	Fake *FakePipelineV1alpha1
	ns   string
}

var pipelinesResource = schema.GroupVersionResource{Group: "pipeline.kubeflow.org", Version: "v1alpha1", Resource: "pipelines"}

var pipelinesKind = schema.GroupVersionKind{Group: "pipeline.kubeflow.org", Version: "v1alpha1", Kind: "Pipeline"}

// Get takes name of the pipeline, and returns the corresponding pipeline object, and an error if there is any.
func (c *FakePipelines) Get(name string, options v1.GetOptions) (result *v1alpha1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(pipelinesResource, c.ns, name), &v1alpha1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Pipeline), err
}

// List takes label and field selectors, and returns the list of Pipelines that match those selectors.
func (c *FakePipelines) List(opts v1.ListOptions) (result *v1alpha1.PipelineList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(pipelinesResource, pipelinesKind, c.ns, opts), &v1alpha1.PipelineList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PipelineList{}
	for _, item := range obj.(*v1alpha1.PipelineList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested pipelines.
func (c *FakePipelines) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(pipelinesResource, c.ns, opts))

}

// Create takes the representation of a pipeline and creates it.  Returns the server's representation of the pipeline, and an error, if there is any.
func (c *FakePipelines) Create(pipeline *v1alpha1.Pipeline) (result *v1alpha1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(pipelinesResource, c.ns, pipeline), &v1alpha1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Pipeline), err
}

// Update takes the representation of a pipeline and updates it. Returns the server's representation of the pipeline, and an error, if there is any.
func (c *FakePipelines) Update(pipeline *v1alpha1.Pipeline) (result *v1alpha1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(pipelinesResource, c.ns, pipeline), &v1alpha1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Pipeline), err
}

// Delete takes name of the pipeline and deletes it. Returns an error if one occurs.
func (c *FakePipelines) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(pipelinesResource, c.ns, name), &v1alpha1.Pipeline{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePipelines) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(pipelinesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.PipelineList{})
	return err
}

// Patch applies the patch and returns the patched pipeline.
func (c *FakePipelines) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pipelinesResource, c.ns, name, data, subresources...), &v1alpha1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Pipeline), err
}
