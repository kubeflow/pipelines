// Copyright 2019 The Kubeflow Authors
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
	"context"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type FakeKuberneteCoreClient struct {
	podClientFake *FakePodClient
}

func (c *FakeKuberneteCoreClient) PodClient(namespace string) v1.PodInterface {
	if len(namespace) == 0 {
		panic(util.NewResourceNotFoundError("Namespace", namespace))
	}
	return c.podClientFake
}

func (c *FakeKuberneteCoreClient) GetClientSet() kubernetes.Interface {
	// Return nil for fake implementation - tests that need this should use a mock
	return nil
}

func NewFakeKuberneteCoresClient() *FakeKuberneteCoreClient {
	return &FakeKuberneteCoreClient{&FakePodClient{}}
}

type FakeKubernetesCoreClientWithBadPodClient struct {
	podClientFake *FakeBadPodClient
}

func NewFakeKubernetesCoreClientWithBadPodClient() *FakeKubernetesCoreClientWithBadPodClient {
	return &FakeKubernetesCoreClientWithBadPodClient{&FakeBadPodClient{}}
}

func (c *FakeKubernetesCoreClientWithBadPodClient) PodClient(namespace string) v1.PodInterface {
	return c.podClientFake
}

func (c *FakeKubernetesCoreClientWithBadPodClient) GetClientSet() kubernetes.Interface {
	// Return nil for fake implementation
	return nil
}

func (c *FakePodClient) EvictV1(context.Context, *policyv1.Eviction) error {
	return nil
}

func (c *FakePodClient) EvictV1beta1(context.Context, *policyv1beta1.Eviction) error {
	return nil
}
