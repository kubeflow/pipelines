// Copyright 2018 Google LLC
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

package main

import (
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// ML pipeline API root URL
	mlPipelineBase = "/api/v1/proxy/namespaces/%s/services/ml-pipeline:8888/apis/v1alpha1/%s"
)

func getKubernetesClient() (*kubernetes.Clientset, error) {
	// use the current context in kubeconfig
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get cluster config during K8s client initialization")
	}
	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create client set during K8s client initialization")
	}

	return clientSet, nil
}

func waitForReady(namespace string, initializeTimeout time.Duration) error {
	clientSet, err := getKubernetesClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to get K8s client set when waiting for ML pipeline to be ready")
	}

	var operation = func() error {
		response := clientSet.RESTClient().Get().
			AbsPath(fmt.Sprintf(mlPipelineBase, namespace, "healthz")).Do()
		if response.Error() == nil {
			return nil
		}
		var code int
		response.StatusCode(&code)
		// we wait only on 503 service unavailable. Stop retry otherwise.
		if code != 503 {
			return backoff.Permanent(errors.Wrapf(response.Error(), "Waiting for ml pipeline failed with non retriable error."))
		}
		return response.Error()
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initializeTimeout
	err = backoff.Retry(operation, b)
	return errors.Wrapf(err, "Waiting for ml pipeline failed after all attempts.")
}

func initTest(namespace string, initializeTimeout time.Duration) error {
	return waitForReady(namespace, initializeTimeout)
}
