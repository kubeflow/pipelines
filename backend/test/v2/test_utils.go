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

package test

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func WaitForReady(initializeTimeout time.Duration) error {
	var tlsConfig *tls.Config
	var err error
	if *config.TLSEnabled {
		tlsConfig, err = GetTLSConfig(*config.CaCertPath)
		if err != nil {
			return errors.Wrap(err, "Failed to configure API readiness TLS")
		}
	}
	client, err := api_server.NewHealthzClient(GetClientConfig(*config.Namespace), *config.DebugMode, tlsConfig)
	if err != nil {
		return errors.Wrap(err, "Failed to create API readiness client")
	}
	operation := func() error {
		_, err := client.GetHealthz()
		return err
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initializeTimeout
	err = backoff.Retry(operation, b)
	return errors.Wrapf(err, "Waiting for ml pipeline API server failed after all attempts.")
}

func GetClientConfig(namespace string) clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{Context: clientcmdapi.Context{Namespace: namespace}}
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules,
		&overrides, os.Stdin)
}

func GetDefaultPipelineRunnerServiceAccount(isKubeflowMode bool) string {
	if isKubeflowMode {
		return "default-editor"
	} else {
		return "pipeline-runner"
	}
}

func GetPipelineUploadClient(
	uploadPipelinesWithKubernetes bool,
	isKubeflowMode bool,
	isDebugMode bool,
	namespace string,
	clientConfig clientcmd.ClientConfig,
	tlsCfg *tls.Config,
) (api_server.PipelineUploadInterface, error) {
	if uploadPipelinesWithKubernetes {
		return api_server.NewPipelineUploadClientKubernetes(clientConfig, namespace)
	}

	if isKubeflowMode {
		return api_server.NewKubeflowInClusterPipelineUploadClient(namespace, isDebugMode, tlsCfg)
	}

	return api_server.NewPipelineUploadClient(clientConfig, isDebugMode, tlsCfg)
}

// GetTLSConfig returns TLS config set with system CA certs as well as custom CA stored at input caCertPath if provided.
func GetTLSConfig(caCertPath string) (*tls.Config, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, err
		}
	}
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}
