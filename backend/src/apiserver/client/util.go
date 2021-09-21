// Copyright 2021 Arrikto Inc.
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
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getKubernetesClientset(clientParams util.ClientParameters) (*kubernetes.Clientset, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}
	restConfig.QPS = float32(clientParams.QPS)
	restConfig.Burst = clientParams.Burst

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return clientSet, nil
}
