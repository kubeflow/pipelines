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
	"flag"
	"os"
	"time"

	workflowclientSet "github.com/argoproj/argo/pkg/client/clientset/versioned"
	workflowinformers "github.com/argoproj/argo/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/signals"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL                   string
	kubeconfig                  string
	namespace                   string
	initializeTimeout           time.Duration
	timeout                     time.Duration
	mlPipelineAPIServerName     string
	mlPipelineAPIServerPort     string
	mlPipelineAPIServerBasePath string
)

const (
	kubeconfigFlagName                  = "kubeconfig"
	masterFlagName                      = "master"
	namespaceFlagName                   = "namespace"
	initializationTimeoutFlagName       = "initializeTimeout"
	timeoutFlagName                     = "timeout"
	mlPipelineAPIServerBasePathFlagName = "mlPipelineAPIServerBasePath"
	mlPipelineAPIServerNameFlagName     = "mlPipelineAPIServerName"
	mlPipelineAPIServerPortFlagName     = "mlPipelineAPIServerPort"
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	swfClient, err := swfclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building schedule clientset: %s", err.Error())
	}

	workflowClient, err := workflowclientSet.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building workflow clientset: %s", err.Error())
	}

	swfInformerFactory := swfinformers.NewSharedInformerFactory(swfClient, time.Second*30)
	workflowInformerFactory := workflowinformers.NewSharedInformerFactory(workflowClient, time.Second*30)

	pipelineClient, err := client.NewPipelineClient(
		namespace,
		initializeTimeout,
		timeout,
		mlPipelineAPIServerBasePath,
		mlPipelineAPIServerName,
		mlPipelineAPIServerPort,
		masterURL,
		kubeconfig)
	if err != nil {
		log.Fatalf("Error creating ML pipeline API Server client: %v", err)
	}

	controller := NewPersistenceAgent(
		swfInformerFactory,
		workflowInformerFactory,
		pipelineClient,
		util.NewRealTime())

	go swfInformerFactory.Start(stopCh)
	go workflowInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, kubeconfigFlagName, "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, masterFlagName, "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&namespace, namespaceFlagName, os.Getenv("POD_NAMESPACE"), "The namespace the ML pipeline API server is deployed to")
	flag.DurationVar(&initializeTimeout, initializationTimeoutFlagName, 2*time.Minute, "Duration to wait for initialization of the ML pipeline API server.")
	flag.DurationVar(&timeout, timeoutFlagName, 1*time.Minute, "Duration to wait for calls to complete.")
	flag.StringVar(&mlPipelineAPIServerName, mlPipelineAPIServerNameFlagName, "ml-pipeline", "Name of the ML pipeline API server.")
	flag.StringVar(&mlPipelineAPIServerPort, mlPipelineAPIServerPortFlagName, "8887", "Port of the ML pipeline API server.")
	flag.StringVar(&mlPipelineAPIServerBasePath, mlPipelineAPIServerBasePathFlagName,
		"/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/apis/v1beta1/%s",
		"The base path for the ML pipeline API server.")
}
