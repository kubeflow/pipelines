// Copyright 2018-2023 The Kubeflow Authors
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

package testutil

import (
	"os"
	"time"

	"github.com/kubeflow/pipelines/backend/test/config"
	backendtest "github.com/kubeflow/pipelines/backend/test/v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// WaitForReady uses the same endpoint, API prefix, and TLS settings as the clients.
func WaitForReady(initializeTimeout time.Duration) error {
	return backendtest.WaitForReady(initializeTimeout)
}

func GetClientConfig(namespace string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{Context: clientcmdapi.Context{Namespace: namespace}}
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules,
		&overrides, os.Stdin)
}

func GetDefaultPipelineRunnerServiceAccount() string {
	if *config.KubeflowMode || *config.MultiUserMode {
		return *config.UserServiceAccountName
	} else {
		return *config.DefaultServiceAccountName
	}
}
