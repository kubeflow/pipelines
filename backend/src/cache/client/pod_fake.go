// Copyright 2020 Google LLC
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
	"errors"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type FakePodClient struct {
	watchIsCalled bool
	patchIsCalled bool
}

func (FakePodClient) GetEphemeralContainers(string, v1.GetOptions) (*corev1.EphemeralContainers, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) UpdateEphemeralContainers(string, *corev1.EphemeralContainers) (*corev1.EphemeralContainers, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Create(*corev1.Pod) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Update(*corev1.Pod) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) UpdateStatus(*corev1.Pod) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Delete(name string, options *v1.DeleteOptions) error {
	return nil
}

func (FakePodClient) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) Get(name string, options v1.GetOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) List(opts v1.ListOptions) (*corev1.PodList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakePodClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	f.watchIsCalled = true
	event := watch.Event{
		Type:   watch.Added,
		Object: &corev1.Pod{},
	}
	ch := make(chan watch.Event, 1)
	ch <- event
	return nil, nil
}

func (f FakePodClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *corev1.Pod, err error) {
	f.patchIsCalled = true
	return nil, nil
}

func (FakePodClient) Bind(binding *corev1.Binding) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) Evict(eviction *v1beta1.Eviction) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) GetLogs(name string, opts *corev1.PodLogOptions) *rest.Request {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

type FakeBadPodClient struct {
	FakePodClient
}

func (FakeBadPodClient) Delete(name string, options *v1.DeleteOptions) error {
	return errors.New("failed to delete pod")
}
