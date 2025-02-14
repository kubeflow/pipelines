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

package client

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type SwfClientInterface interface {
	ScheduledWorkflow(namespace string) v1beta1.ScheduledWorkflowInterface
}

type SwfClient struct {
	swfV1beta1Client v1beta1.ScheduledworkflowV1beta1Interface
}

func (swfClient *SwfClient) ScheduledWorkflow(namespace string) v1beta1.ScheduledWorkflowInterface {
	return swfClient.swfV1beta1Client.ScheduledWorkflows(namespace)
}

// creates a new client for the Kubernetes ScheduledWorkflow CRD.
func NewScheduledWorkflowClientOrFatal(initConnectionTimeout time.Duration, clientParams util.ClientParameters) *SwfClient {
	var swfClient v1beta1.ScheduledworkflowV1beta1Interface
	operation := func() error {
		restConfig, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		restConfig.QPS = float32(clientParams.QPS)
		restConfig.Burst = clientParams.Burst
		swfClientSet := swfclient.NewForConfigOrDie(restConfig)
		swfClient = swfClientSet.ScheduledworkflowV1beta1()
		return nil
	}

	swfClientInstance, err := newOutOfClusterSwfClient()
	if err == nil {
		return swfClientInstance
	}

	glog.Infof("Starting to create scheduled workflow client by in cluster config.")
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	if err := backoff.Retry(operation, b); err != nil {
		// all failed
		glog.Fatalf("Failed to create scheduled workflow client. Error: %v", err)
	}

	return &SwfClient{swfClient}
}

// Use out of cluster config for local testing purposes.
func newOutOfClusterSwfClient() (*SwfClient, error) {
	home := homedir.HomeDir()
	if home == "" {
		return nil, errors.New("Cannot get home dir")
	}

	defaultKubeConfigPath := filepath.Join(home, ".kube", "config")
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", defaultKubeConfigPath)
	if err != nil {
		return nil, err
	}

	// create the clientset
	swfClientSet, err := swfclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create swf client set: %w", err)
	}

	return &SwfClient{swfClientSet.ScheduledworkflowV1beta1()}, nil
}
