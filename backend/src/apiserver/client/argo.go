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

package client

import (
	"time"

	argoclient "github.com/argoproj/argo/pkg/client/clientset/versioned"
	argoprojv1alpha1 "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

type ArgoClientInterface interface {
	Workflow(namespace string) argoprojv1alpha1.WorkflowInterface
}

type ArgoClient struct {
	argoProjClient argoprojv1alpha1.ArgoprojV1alpha1Interface
}

func (argoClient *ArgoClient) Workflow(namespace string) argoprojv1alpha1.WorkflowInterface {
	return argoClient.argoProjClient.Workflows(namespace)
}

func NewArgoClientOrFatal(initConnectionTimeout time.Duration) *ArgoClient {
	var argoProjClient argoprojv1alpha1.ArgoprojV1alpha1Interface
	var operation = func() error {
		restConfig, err := rest.InClusterConfig()
		if err != nil {
			return errors.Wrap(err, "Failed to initialize the RestConfig")
		}
		argoProjClient = argoclient.NewForConfigOrDie(restConfig).ArgoprojV1alpha1()
		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err := backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create ArgoClient. Error: %v", err)
	}
	return &ArgoClient{argoProjClient}
}
