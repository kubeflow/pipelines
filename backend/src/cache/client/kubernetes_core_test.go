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

package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestKubernetesCorePodClient(t *testing.T) {
	fakeClientset := fakeclientset.NewSimpleClientset()
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}

	podClient := kubernetesCore.PodClient("test-namespace")
	assert.NotNil(t, podClient)
}

func TestKubernetesCorePodClientDifferentNamespaces(t *testing.T) {
	fakeClientset := fakeclientset.NewSimpleClientset()
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}

	clientA := kubernetesCore.PodClient("namespace-a")
	clientB := kubernetesCore.PodClient("namespace-b")

	assert.NotNil(t, clientA)
	assert.NotNil(t, clientB)
}

func TestKubernetesCoreImplementsInterface(t *testing.T) {
	fakeClientset := fakeclientset.NewSimpleClientset()
	kubernetesCore := &KubernetesCore{coreV1Client: fakeClientset.CoreV1()}

	var _ KubernetesCoreInterface = kubernetesCore
}
